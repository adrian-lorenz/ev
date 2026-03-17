package cmd

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode/utf8"

	"envault/detector"

	"github.com/spf13/cobra"
)

const (
	ansiReset  = "\033[0m"
	ansiRed    = "\033[31m"
	ansiYellow = "\033[33m"
	ansiGreen  = "\033[32m"
	ansiCyan   = "\033[36m"
	ansiBold   = "\033[1m"
	ansiGray   = "\033[90m"
)

func init() {
	rootCmd.AddCommand(newScanCmd())
}

type fileFinding struct {
	file       string
	line       int
	col        int
	findType   string
	ruleID     string
	text       string
	confidence float64
}

func newScanCmd() *cobra.Command {
	var reportPath string

	cmd := &cobra.Command{
		Use:   "scan [path...]",
		Short: "Scan files for leaked secrets and credentials",
		Long: `Scans files for leaked secrets and credentials (API keys, tokens, passwords, …).

Paths default to the current directory. Dot-directories (.git, .idea, …),
binary files, and files larger than 10 MB are skipped automatically.

Exit code 1 when findings are present.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			sc := detector.NewScanner(nil)

			roots := args
			if len(roots) == 0 {
				roots = []string{"."}
			}

			var findings []fileFinding
			filesScanned := 0

			for _, root := range roots {
				_ = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
					if err != nil {
						return nil
					}
					if d.IsDir() {
						if d.Name() != "." && strings.HasPrefix(d.Name(), ".") {
							return filepath.SkipDir
						}
						return nil
					}
					info, err := d.Info()
					if err != nil || info.Size() > 10*1024*1024 {
						return nil
					}
					content, err := os.ReadFile(path)
					if err != nil {
						return nil
					}
					if !looksLikeText(content) {
						return nil
					}

					text := blankNoscanLines(string(content))
					_, hits := sc.Scan(text)
					filesScanned++

					for _, h := range hits {
						line, col := byteOffsetToLineCol(text, h.Start)
						findings = append(findings, fileFinding{
							file:       path,
							line:       line,
							col:        col,
							findType:   string(h.Type),
							ruleID:     h.RuleID,
							text:       h.Text,
							confidence: h.Confidence,
						})
					}
					return nil
				})
			}

			useColor := stdoutIsTerminal()
			output := renderOutput(findings, filesScanned, useColor)

			if reportPath != "" {
				if err := os.WriteFile(reportPath, []byte(stripANSI(output)), 0600); err != nil {
					return fmt.Errorf("writing report: %w", err)
				}
				fmt.Fprintf(os.Stderr, "Report saved to %s\n", reportPath)
			}

			fmt.Print(output)

			if len(findings) > 0 {
				os.Exit(1)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&reportPath, "report", "", "save plain-text report to file")
	return cmd
}

func renderOutput(findings []fileFinding, filesScanned int, color bool) string {
	col := func(code, s string) string {
		if color {
			return code + s + ansiReset
		}
		return s
	}
	bold := func(s string) string { return col(ansiBold, s) }
	red := func(s string) string { return col(ansiRed, s) }
	yellow := func(s string) string { return col(ansiYellow, s) }
	green := func(s string) string { return col(ansiGreen, s) }
	cyan := func(s string) string { return col(ansiCyan, s) }
	gray := func(s string) string { return col(ansiGray, s) }

	var sb strings.Builder
	sb.WriteString(bold("ev scan") + "\n")
	sb.WriteString(strings.Repeat("─", 60) + "\n\n")

	if len(findings) == 0 {
		sb.WriteString(green("✓ No findings — ") + fmt.Sprintf("%d file(s) scanned.\n", filesScanned))
		return sb.String()
	}

	// Group by file, preserving sort order
	byFile := map[string][]fileFinding{}
	fileOrder := []string{}
	for _, f := range findings {
		if _, seen := byFile[f.file]; !seen {
			fileOrder = append(fileOrder, f.file)
		}
		byFile[f.file] = append(byFile[f.file], f)
	}
	sort.Strings(fileOrder)

	for _, file := range fileOrder {
		sb.WriteString(bold(cyan(file)) + "\n")
		for _, f := range byFile[file] {
			var conf string
			switch {
			case f.confidence >= 0.9:
				conf = red("HIGH")
			case f.confidence >= 0.7:
				conf = yellow(" MED")
			default:
				conf = gray(" LOW")
			}

			label := f.findType
			if f.ruleID != "" {
				label = f.ruleID
			}

			snippet := redact(f.text)
			fmt.Fprintf(&sb, "  %s  %s:%d:%d  %s\n",
				conf,
				gray(file),
				f.line, f.col,
				bold(label)+" "+gray(snippet),
			)
		}
		sb.WriteString("\n")
	}

	sb.WriteString(strings.Repeat("─", 60) + "\n")
	fmt.Fprintf(&sb, "%s across %d file(s)  %s\n",
		red(fmt.Sprintf("%d finding(s)", len(findings))),
		len(fileOrder),
		gray(fmt.Sprintf("(%d files scanned)", filesScanned)),
	)
	return sb.String()
}

// redact shows the first 6 characters and masks the rest.
func redact(s string) string {
	s = strings.TrimSpace(s)
	runes := []rune(s)
	if len(runes) <= 6 {
		return strings.Repeat("*", len(runes))
	}
	tail := min(len(runes)-6, 20)
	return string(runes[:6]) + strings.Repeat("*", tail) + "…"
}

// byteOffsetToLineCol converts a byte offset into 1-based line and column numbers.
func byteOffsetToLineCol(text string, offset int) (line, col int) {
	line, col = 1, 1
	for i, ch := range text {
		if i >= offset {
			break
		}
		if ch == '\n' {
			line++
			col = 1
		} else {
			col++
		}
	}
	return
}

// looksLikeText returns true if the content is valid UTF-8 or has few non-printable bytes.
func looksLikeText(b []byte) bool {
	if utf8.Valid(b) {
		return true
	}
	sample := b
	if len(sample) > 512 {
		sample = sample[:512]
	}
	nonPrint := 0
	for _, c := range sample {
		if c < 0x09 || (c > 0x0d && c < 0x20) {
			nonPrint++
		}
	}
	return float64(nonPrint)/float64(len(sample)) < 0.1
}

// stdoutIsTerminal returns true when stdout is connected to a terminal.
func stdoutIsTerminal() bool {
	fi, err := os.Stdout.Stat()
	return err == nil && fi.Mode()&os.ModeCharDevice != 0
}

// blankNoscanLines replaces lines containing "# noscan" or "// noscan" with
// spaces of equal byte length, preserving byte offsets so findings on other
// lines are unaffected. The marker is case-insensitive.
func blankNoscanLines(text string) string {
	lines := strings.Split(text, "\n")
	changed := false
	for i, line := range lines {
		lower := strings.ToLower(line)
		if strings.Contains(lower, "# noscan") || strings.Contains(lower, "// noscan") {
			lines[i] = strings.Repeat(" ", len(line))
			changed = true
		}
	}
	if !changed {
		return text
	}
	return strings.Join(lines, "\n")
}

// stripANSI removes ANSI escape sequences for plain-text output.
func stripANSI(s string) string {
	var b strings.Builder
	i := 0
	for i < len(s) {
		if s[i] == '\033' && i+1 < len(s) && s[i+1] == '[' {
			j := i + 2
			for j < len(s) && s[j] != 'm' {
				j++
			}
			i = j + 1
		} else {
			b.WriteByte(s[i])
			i++
		}
	}
	return b.String()
}
