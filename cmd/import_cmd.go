package cmd

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"envault/vault"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(newImportCmd())
}

func newImportCmd() *cobra.Command {
	var dryRun bool
	var noOverwrite bool

	cmd := &cobra.Command{
		Use:   "import <file>",
		Short: "Import secrets from a .env file",
		Example: `  envault import .env
  envault import secrets.env --project payment-service
  envault import .env --dry-run`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			f, err := os.Open(args[0])
			if err != nil {
				return fmt.Errorf("opening file: %w", err)
			}
			defer f.Close()

			parsed, err := parseDotEnv(f)
			if err != nil {
				return fmt.Errorf("parsing file: %w", err)
			}

			if len(parsed) == 0 {
				fmt.Fprintln(os.Stderr, "No variables found in file.")
				return nil
			}

			project := resolveProject()

			if dryRun {
				fmt.Fprintf(os.Stderr, "Dry run – would import into project %q:\n", project)
				for k, v := range parsed {
					fmt.Fprintf(os.Stderr, "  %s=%s\n", k, v)
				}
				return nil
			}

			v, path, password, err := openVault()
			if err != nil {
				return err
			}

			imported := 0
			skipped := 0
			for key, value := range parsed {
				// Normalize exactly like `ev set` so keys round-trip through get/delete
				key = normalizeKeyInput(key)
				if err := vault.ValidateName(key); err != nil {
					fmt.Fprintf(os.Stderr, "Skipping invalid key %q: %v\n", key, err)
					skipped++
					continue
				}
				if noOverwrite {
					if _, exists := v.Get(project, key); exists {
						skipped++
						continue
					}
				}
				v.Set(project, key, value)
				imported++
			}

			if err := v.Save(path, password); err != nil {
				return err
			}

			// Refresh active session so ev run/load pick up the new secrets immediately
			if err := vault.RefreshSession(project, v.GetAll(project)); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: could not refresh session: %v\n", err)
			}

			fmt.Fprintf(os.Stderr, "Imported %d secret(s) into project %q", imported, project)
			if skipped > 0 {
				fmt.Fprintf(os.Stderr, " (%d skipped)", skipped)
			}
			fmt.Fprintln(os.Stderr)

			printProjectHint(project, args[0])
			return nil
		},
	}

	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "print what would be imported without saving")
	cmd.Flags().BoolVar(&noOverwrite, "no-overwrite", false, "skip keys that already exist")
	return cmd
}

// parseDotEnv parses a .env / .tfvars file into a map of key=value pairs.
// Handles comments, quoted values, the "export " prefix, and multi-line { } blocks.
func parseDotEnv(r io.Reader) (map[string]string, error) {
	result := make(map[string]string)
	scanner := bufio.NewScanner(r)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip blank lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Strip "export " prefix
		line = strings.TrimPrefix(line, "export ")
		line = strings.TrimSpace(line)

		idx := strings.IndexByte(line, '=')
		if idx < 0 {
			continue
		}

		key := unquote(strings.TrimSpace(line[:idx]))
		value := strings.TrimSpace(line[idx+1:])

		// Multi-line block value: collect lines until matching closing brace
		if value == "{" || strings.HasSuffix(value, " {") || strings.HasSuffix(value, "\t{") {
			var lines []string
			depth := strings.Count(value, "{") - strings.Count(value, "}")
			for depth > 0 && scanner.Scan() {
				inner := scanner.Text()
				lines = append(lines, inner)
				depth += strings.Count(inner, "{") - strings.Count(inner, "}")
			}
			// Try to flatten: if block contains only "key" = "value" lines, import each as its own secret
			if flat, ok := parseFlatTFMap(lines); ok {
				for k, v := range flat {
					result[k] = v
				}
			} else if key != "" {
				// Store whole block as single value
				result[key] = "{" + "\n" + strings.Join(lines, "\n")
			}
			continue
		}

		// Strip inline comments (only outside quotes)
		value = stripInlineComment(value)

		// Strip surrounding quotes from value
		value = unquote(value)

		if key != "" {
			result[key] = value
		}
	}

	return result, scanner.Err()
}

func unquote(s string) string {
	if len(s) >= 2 {
		if (s[0] == '"' && s[len(s)-1] == '"') || (s[0] == '\'' && s[len(s)-1] == '\'') {
			return s[1 : len(s)-1]
		}
	}
	return s
}

func stripInlineComment(s string) string {
	if len(s) == 0 {
		return s
	}
	// Quoted values: strip block comments after the closing quote, but leave value intact
	if s[0] == '"' || s[0] == '\'' {
		q := s[0]
		// find closing quote
		end := strings.IndexByte(s[1:], q)
		if end >= 0 {
			return s[:end+2] // return just the quoted part
		}
		return s
	}
	// Unquoted: strip #, //, /* ... */ comments
	// Find earliest comment marker (// must be preceded by whitespace or be at start
	// to avoid treating URLs like https:// or postgres:// as comments)
	earliest := len(s)
	for _, marker := range []string{"#", "/*"} {
		if idx := strings.Index(s, marker); idx >= 0 && idx < earliest {
			earliest = idx
		}
	}
	// // is a comment only when at position 0 or preceded by whitespace
	for i := 0; i <= len(s)-2; i++ {
		if s[i] == '/' && s[i+1] == '/' {
			if i == 0 || s[i-1] == ' ' || s[i-1] == '\t' {
				if i < earliest {
					earliest = i
				}
				break
			}
		}
	}
	return strings.TrimSpace(s[:earliest])
}

// parseFlatTFMap tries to parse block lines as a flat map of "key" = "value" pairs.
// Returns (map, true) only if every non-blank, non-comment line matches that pattern
// AND every key (after normalization) passes ValidateName.
// Returns (nil, false) if the block contains nested structures, unrecognized lines,
// or keys that cannot be stored as valid secret names.
func parseFlatTFMap(lines []string) (map[string]string, bool) {
	result := make(map[string]string)
	for _, raw := range lines {
		line := strings.TrimSpace(raw)
		if line == "" || line == "}" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "//") {
			continue
		}
		// Reject nested blocks
		if strings.HasSuffix(line, "{") || line == "{" {
			return nil, false
		}
		idx := strings.Index(line, "=")
		if idx < 0 {
			return nil, false
		}
		k := unquote(strings.TrimSpace(line[:idx]))
		v := strings.TrimSpace(line[idx+1:])
		v = stripInlineComment(v)
		v = unquote(v)
		if k == "" {
			return nil, false
		}
		// Normalize and validate — if any key is invalid, fall back to whole-block storage
		k = strings.ReplaceAll(k, "-", "_")
		if err := vault.ValidateName(k); err != nil {
			return nil, false
		}
		result[k] = v
	}
	return result, true
}

// projectKind enumerates detected project types.
type projectKind int

const (
	kindUnknown    projectKind = iota
	kindTerraform              // .tf files
	kindPython                 // .py / pyproject.toml / requirements.txt
	kindNode                   // package.json + .js files
	kindTypeScript             // tsconfig.json / .ts files
	kindGo                     // go.mod
	kindDocker                 // docker-compose.yml / Dockerfile
)

func detectProjectKind(importedFile string) projectKind {
	// Detect from the imported filename first
	switch {
	case hasSuffix(importedFile, ".tfvars", ".tfvars.json"):
		return kindTerraform
	case hasSuffix(importedFile, ".py"):
		return kindPython
	case hasSuffix(importedFile, ".ts"):
		return kindTypeScript
	case hasSuffix(importedFile, ".js"):
		return kindNode
	}

	// Fall back to scanning CWD for known project files
	has := func(patterns ...string) bool {
		for _, p := range patterns {
			if matches, _ := filepath.Glob(p); len(matches) > 0 {
				return true
			}
		}
		return false
	}
	hasFile := func(names ...string) bool {
		for _, n := range names {
			if _, err := os.Stat(n); err == nil {
				return true
			}
		}
		return false
	}

	switch {
	case has("*.tf"):
		return kindTerraform
	case hasFile("tsconfig.json") || has("*.ts", "src/*.ts"):
		return kindTypeScript
	case hasFile("pyproject.toml", "requirements.txt", "setup.py", "setup.cfg", "uv.lock", "uv.toml") || has("*.py"):
		return kindPython
	case hasFile("package.json") || has("*.js", "src/*.js"):
		return kindNode
	case hasFile("go.mod") || has("*.go"):
		return kindGo
	case hasFile("docker-compose.yml", "docker-compose.yaml", "Dockerfile"):
		return kindDocker
	default:
		return kindUnknown
	}
}

func hasSuffix(name string, suffixes ...string) bool {
	lower := strings.ToLower(name)
	for _, s := range suffixes {
		if strings.HasSuffix(lower, s) {
			return true
		}
	}
	return false
}

func printProjectHint(project, importedFile string) {
	kind := detectProjectKind(importedFile)
	if kind == kindUnknown {
		return
	}

	fmt.Fprintln(os.Stderr)
	fmt.Fprintf(os.Stderr, "Detected project type: %s\n\n", kindLabel(kind))
	fmt.Fprintln(os.Stderr, "── How to use your secrets ───────────────────────────────────")

	switch kind {
	case kindTerraform:
		fmt.Fprintf(os.Stderr, `
  # Load secrets into current shell (one-time)
  eval "$(ev load)"

  # Or run terraform with secrets injected automatically
  ev run terraform plan
  ev run terraform apply

  # Unlock once for 8 hours (no password prompts)
  ev open
  ev run terraform plan
`)

	case kindPython:
		fmt.Fprintf(os.Stderr, `
  # Load secrets into current shell (one-time)
  eval "$(ev load)"

  # Run your app with secrets injected (no eval needed)
  ev run python main.py
  ev run uvicorn main:app --reload
  ev run uv run python -m pytest

  # Unlock once for 8 hours (e.g. for IDE / PyCharm)
  ev open --project %s
  # Then set run config to: ev run uvicorn main:app --reload
`, project)

	case kindTypeScript:
		fmt.Fprintf(os.Stderr, `
  # Load secrets into current shell (one-time)
  eval "$(ev load)"

  # Run your app with secrets injected
  ev run npm run dev
  ev run npx ts-node src/index.ts
  ev run node dist/index.js

  # Unlock once for 8 hours
  ev open --project %s
`, project)

	case kindNode:
		fmt.Fprintf(os.Stderr, `
  # Load secrets into current shell (one-time)
  eval "$(ev load)"

  # Run your app with secrets injected
  ev run node index.js
  ev run npm start
  ev run npm run dev

  # Unlock once for 8 hours
  ev open --project %s
`, project)

	case kindGo:
		fmt.Fprintf(os.Stderr, `
  # Load secrets into current shell (one-time)
  eval "$(ev load)"

  # Run your app with secrets injected
  ev run go run .

  # Unlock once for 8 hours
  ev open --project %s
`, project)

	case kindDocker:
		fmt.Fprintf(os.Stderr, `
  # Load secrets into current shell (one-time)
  eval "$(ev load)"

  # Run docker-compose with secrets injected
  ev run docker-compose up
  ev run docker-compose run --rm app bash

  # Unlock once for 8 hours
  ev open --project %s
`, project)
	}

	fmt.Fprintln(os.Stderr, "──────────────────────────────────────────────────────────────")
}

func kindLabel(k projectKind) string {
	switch k {
	case kindTerraform:
		return "Terraform"
	case kindPython:
		return "Python"
	case kindTypeScript:
		return "TypeScript"
	case kindNode:
		return "Node.js"
	case kindGo:
		return "Go"
	case kindDocker:
		return "Docker"
	default:
		return "unknown"
	}
}
