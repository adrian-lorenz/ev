# ev - Local Encrypted Secret Manager

**DOCS.md**

## 1. Project Overview

`ev` (envault) is a local encrypted secret manager designed for modern AI-native development workflows. It securely stores secrets outside of your project repository while providing seamless access during application runtime.

### Key Features

- **Local Encryption**: Secrets are encrypted on your machine before storage
- **No Cloud Dependency**: Operates entirely locally without background services
- **AI-Safe**: Prevents accidental exposure of secrets to AI coding agents
- **Multi-Language Support**: Works with Go, JavaScript, Python, Terraform, and other ecosystems
- **1Password Integration**: Optional encrypted off-site backup
- **Zero Daemon**: Simple CLI tool with no persistent processes
- **IDE Friendly**: Compatible with IDE run configurations

### Problem Solved

Traditional `.env` files and secret storage methods expose sensitive data to:
- Version control systems
- AI coding assistants that scan project files
- Accidental commits
- Local file system vulnerabilities

`ev` replaces these insecure patterns with encrypted storage and runtime injection.

## 2. Architecture & Components

### Core Components

```
ev/
├── cmd/            # CLI command implementations (Cobra)
├── detector/       # Secret detection and injection logic
├── vault/          # Encryption/decryption core
├── web/            # Web interface components (JavaScript/HTML)
├── main.go         # Entry point
└── .github/        # CI/CD workflows
```

### Data Flow

1. **Encryption**:
   - User provides secrets via CLI or web interface
   - Secrets are encrypted using local keys
   - Encrypted data is stored outside the project directory
   - A minimal `.envault` marker file is created in the project

2. **Runtime**:
   - Application starts and detects `.envault` file
   - `ev` decrypts secrets and injects them into the environment
   - Application accesses secrets via standard environment variables

### Key Technologies

- **Go**: Primary implementation language
- **Cobra**: CLI framework (`github.com/spf13/cobra`)
- **NaCl**: Encryption via `golang.org/x/crypto`
- **JavaScript**: Web interface components
- **Shell/YAML**: Installation and CI/CD scripts

## 3. Getting Started

### Prerequisites

- Go 1.25+ (for building from source)
- Git
- (Optional) 1Password CLI for sync functionality

### Installation

#### Linux/macOS

```bash
curl -fsSL https://raw.githubusercontent.com/adrian-lorenz/ev/main/install.sh | bash
```

#### Windows (PowerShell)

```powershell
Invoke-WebRequest -Uri "https://raw.githubusercontent.com/adrian-lorenz/ev/main/install.ps1" -OutFile "install.ps1"
.\install.ps1
```

#### From Source

```bash
git clone https://github.com/adrian-lorenz/ev.git
cd ev
make build
sudo mv ./bin/ev /usr/local/bin/
```

### Running Locally

1. Initialize a new vault in your project:

```bash
cd my-project
ev init
```

2. Add secrets:

```bash
ev set DATABASE_URL "postgres://user:pass@localhost:5432/db"
ev set API_KEY "12345-abcde"
```

3. Run your application:

```bash
ev run -- go run main.go
# or for npm projects:
ev run -- npm start
```

## 4. Configuration

### Environment Variables

| Variable          | Description                          | Default       |
|-------------------|--------------------------------------|---------------|
| `EV_VAULT_PATH`   | Custom vault storage location        | `~/.ev/vault` |
| `EV_AUTO_INJECT`  | Enable automatic environment injection | `false`      |
| `EV_1PASSWORD`    | Enable 1Password sync                | `false`       |

### Configuration Files

#### `.envault`

Created in project root during `ev init`:

```yaml
name: my-project
version: 1
vault_id: abc123-def456
```

#### `~/.ev/config.yml` (User-level)

```yaml
vault_path: ~/.custom-vault-location
default_editor: vim
1password:
  enabled: true
  vault: "Personal"
```

### Command-Line Flags

| Flag              | Description                          |
|-------------------|--------------------------------------|
| `--vault-path`    | Override vault storage location      |
| `--no-verify`     | Skip secret verification             |
| `--editor`        | Specify editor for secret editing    |

## 5. API / Usage Reference

### CLI Commands

#### `ev init [name]`

Initialize a new vault for the current project.

```bash
ev init my-project
```

#### `ev set <key> <value>`

Add or update a secret.

```bash
ev set DATABASE_URL "postgres://user:pass@localhost:5432/db"
```

#### `ev get <key>`

Retrieve a secret value.

```bash
ev get DATABASE_URL
```

#### `ev list`

List all secrets in the current vault.

```bash
ev list
```

#### `ev run [--] <command>`

Run a command with secrets injected.

```bash
ev run -- go run main.go
ev run -- npm start
```

#### `ev edit`

Open secrets in default editor.

```bash
ev edit
```

#### `ev sync`

Synchronize with 1Password (if configured).

```bash
ev sync
```

#### `ev version`

Display version information.

```bash
ev version
```

### Programmatic Usage

#### Go Applications

```go
package main

import (
	"os"
	"fmt"
)

func main() {
	// Secrets are available as environment variables
	dbURL := os.Getenv("DATABASE_URL")
	fmt.Println("Database URL:", dbURL)
}
```

#### JavaScript Applications

```javascript
// Secrets are available in process.env
const apiKey = process.env.API_KEY;
console.log('API Key:', apiKey);
```

### Web Interface

Accessible via `ev web` (default port 3000):

- View and manage secrets
- Generate new encryption keys
- Configure 1Password sync
- Visualize vault structure

## 6. Contributing

### Development Setup

1. Clone the repository:

```bash
git clone https://github.com/adrian-lorenz/ev.git
cd ev
```

2. Install dependencies:

```bash
go mod download
```

3. Build the project:

```bash
make build
```

4. Run tests:

```bash
make test
```

### Code Style

- **Go**: Follow standard Go formatting (`gofmt`)
- **JavaScript**: Prettier with default settings
- **Markdown**: Consistent heading hierarchy
- **Shell**: POSIX-compliant scripts

### Testing

Run the full test suite:

```bash
make test
```

Test specific components:

```bash
# CLI tests
go test ./cmd/...

# Vault tests
go test ./vault/...

# Detector tests
go test ./detector/...
```

### Pull Request Process

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

### Release Process

1. Update `VERSION` file
2. Create release notes in `CHANGELOG.md`
3. Tag the release (`git tag v1.2.3`)
4. Push tags (`git push --tags`)
5. GitHub Actions will build and publish releases

### Security Considerations

- All encryption uses NaCl (via `golang.org/x/crypto`)
- Never commit unencrypted secrets
- Review all changes to cryptographic code carefully
- Report security vulnerabilities privately to maintainers