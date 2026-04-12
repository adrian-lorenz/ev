# ev - Local Encrypted Secret Manager

## Project Overview

**ev** is a local encrypted secret manager designed for modern AI-native development workflows. It securely stores secrets outside of your project directory, encrypting them locally and only making them available when your application runs.

### Key Features

- **Local Encryption**: Secrets are encrypted on your machine before storage
- **No Cloud Dependency**: Operates entirely locally without requiring a background service
- **AI-Safe**: Prevents AI coding agents from accessing sensitive data by keeping secrets out of the project tree
- **Multi-Language Support**: Works with Go, JavaScript, Python, Terraform, and other ecosystems
- **1Password Integration**: Optional sync for encrypted off-site backup
- **GitWall Cloud Sync**: Optional self-hosted, end-to-end encrypted backup and multi-device sync
- **Developer Experience**: Seamless integration with `uv`, `go`, `npm`, IDE run configurations, and AI tools

### Problem Solved

Traditional secret management approaches (`.env` files, `secrets.tfvars`, etc.) expose sensitive data to:
- Version control systems
- AI coding assistants that scan project directories
- Accidental commits or leaks

ev solves this by:
1. Storing secrets in an encrypted vault outside the project directory
2. Only decrypting them at runtime
3. Maintaining a minimal `.envault` marker file in the project

## Architecture & Components

### File Structure

```
ev/
├── cmd/            # CLI command implementations
├── detector/       # Secret detection and scanning
├── vault/          # Core encryption and storage logic
├── web/            # Web interface components
├── main.go         # Entry point
├── go.mod          # Go module definition
└── ...             # Supporting files
```

### Core Components

1. **Vault System** (`vault/`):
   - Encryption/decryption using Go's `x/crypto` package
   - Local storage in `~/.envault/vault.json`
   - Key management and rotation

2. **CLI Interface** (`cmd/`):
   - Built with Cobra framework
   - Commands for secret management, sync, and project setup
   - Version management via build flags

3. **Secret Detection** (`detector/`):
   - Scans project files for potential secrets
   - Prevents accidental commits of sensitive data

4. **Web Components** (`web/`):
   - Optional web interface for management
   - JavaScript/HTML frontend components

5. **Installation Scripts**:
   - `install.sh`/`install.ps1`: Cross-platform installation
   - `setup.sh`/`setup.ps1`: Initial configuration

### Data Flow

1. **Project Initialization**:
   ```mermaid
   graph LR
     A[Project Directory] -->|creates| B[.envault marker]
     B -->|references| C[~/.envault/vault.json]
   ```

2. **Runtime Access**:
   ```mermaid
   sequenceDiagram
     participant App
     participant ev
     participant Vault
     App->>ev: Request secrets
     ev->>Vault: Decrypt with local key
     Vault-->>ev: Plaintext secrets
     ev-->>App: Inject environment
   ```

## Getting Started

### Prerequisites

- Go 1.25+ (for development)
- Node.js 18+ (for web components)
- Git

### Installation

#### Homebrew (Recommended)

```bash
brew install envault/tap/ev
```

#### Manual Installation

```bash
# Linux/macOS
curl -fsSL https://raw.githubusercontent.com/envault/ev/main/install.sh | bash

# Windows (PowerShell)
Invoke-WebRequest -Uri "https://raw.githubusercontent.com/envault/ev/main/install.ps1" -OutFile install.ps1
.\install.ps1
```

#### From Source

```bash
git clone https://github.com/envault/ev.git
cd ev
make install
```

### Initial Setup

1. Initialize a new vault:

```bash
ev init
```

2. Create your first project:

```bash
ev project add my-project
```

3. Add secrets to the project:

```bash
ev secret add my-project DB_PASSWORD=s3cr3t
ev secret add my-project API_KEY=12345-abcde
```

4. Link a project directory:

```bash
cd ~/projects/my-app
ev link
```

### Running Locally

Start the development server (for web interface):

```bash
make web
```

Run the CLI:

```bash
ev --help
```

## Configuration

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `EVAULT_DIR` | Vault storage directory | `~/.envault` |
| `EVAULT_KEY` | Encryption key (not recommended) | - |
| `EVAULT_1PASSWORD` | Enable 1Password sync | `false` |
| `EVAULT_GITWALL` | GitWall sync endpoint | - |

### Configuration Files

1. **Vault Configuration** (`~/.envault/config.json`):

```json
{
  "version": "1.0",
  "encryption": {
    "algorithm": "aes-256-gcm",
    "key_rotation": "90d"
  },
  "sync": {
    "1password": false,
    "gitwall": {
      "enabled": false,
      "endpoint": ""
    }
  }
}
```

2. **Project Marker** (`.envault`):

```yaml
project: my-project
version: 1
```

### Command-Line Options

```bash
ev [command] [flags]
```

Global flags:
- `--config string`: Path to config file
- `--vault string`: Path to vault directory
- `--debug`: Enable debug logging

## API / Usage Reference

### Core Commands

#### `ev init`
Initialize a new vault

```bash
ev init [flags]
```

Flags:
- `--force`: Overwrite existing vault
- `--key string`: Provide encryption key

#### `ev project`
Project management commands

```bash
ev project [command]
```

Subcommands:
- `add <name>`: Create new project
- `list`: List all projects
- `remove <name>`: Delete project
- `rename <old> <new>`: Rename project

#### `ev secret`
Secret management commands

```bash
ev secret [command] [project]
```

Subcommands:
- `add <project> <key=value>`: Add secret
- `get <project> <key>`: Retrieve secret
- `list <project>`: List all secrets
- `remove <project> <key>`: Delete secret
- `export <project>`: Export secrets as environment variables

#### `ev link`
Link current directory to a project

```bash
ev link [project]
```

#### `ev sync`
Synchronization commands

```bash
ev sync [command]
```

Subcommands:
- `1password`: Sync with 1Password
- `gitwall`: Sync with GitWall
- `status`: Show sync status

#### `ev detect`
Scan for secrets in project files

```bash
ev detect [path]
```

Flags:
- `--fix`: Automatically remove detected secrets
- `--json`: Output as JSON

### Programmatic Usage

#### Go API

```go
package main

import (
	"fmt"
	"envault/vault"
)

func main() {
	// Initialize vault
	v, err := vault.Open("~/.envault")
	if err != nil {
		panic(err)
	}

	// Add secret
	err = v.AddSecret("my-project", "API_KEY", "12345-abcde")
	if err != nil {
		panic(err)
	}

	// Retrieve secret
	value, err := v.GetSecret("my-project", "API_KEY")
	if err != nil {
		panic(err)
	}

	fmt.Println(value)
}
```

#### JavaScript API

```javascript
import { EvClient } from 'ev/web';

const client = new EvClient();

// Add secret
await client.addSecret('my-project', 'API_KEY', '12345-abcde');

// Get secret
const value = await client.getSecret('my-project', 'API_KEY');
console.log(value);
```

### Integration Examples

#### Python Project

```python
# app.py
import os
import subprocess

# Load secrets at runtime
subprocess.run(["ev", "secret", "export", "my-project"], check=True)

# Access secrets
db_password = os.getenv("DB_PASSWORD")
```

#### Go Project

```go
package main

import (
	"log"
	"os"
	"os/exec"
)

func main() {
	// Load secrets
	cmd := exec.Command("ev", "secret", "export", "my-project")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		log.Fatal(err)
	}

	// Access secrets
	apiKey := os.Getenv("API_KEY")
}
```

#### npm Scripts

```json
{
  "scripts": {
    "start": "ev secret export my-project && node app.js",
    "dev": "ev secret export my-project && nodemon app.js"
  }
}
```

## Contributing

### Development Setup

1. Clone the repository:

```bash
git clone https://github.com/envault/ev.git
cd ev
```

2. Install dependencies:

```bash
make deps
```

3. Build the project:

```bash
make build
```

### Code Style

- **Go**:
  - Follow [Effective Go](https://go.dev/doc/effective_go)
  - Use `gofmt` for formatting
  - Lint with `golangci-lint`

- **JavaScript**:
  - Follow [StandardJS](https://standardjs.com/)
  - Use Prettier for formatting

- **Shell**:
  - Follow [ShellCheck](https://www.shellcheck.net/) guidelines
  - Use `shfmt` for formatting

### Testing

Run all tests:

```bash
make test
```

Run specific test suites:

```bash
# Go tests
make test-go

# JavaScript tests
make test-js

# Integration tests
make test-integration
```

### Pull Request Process

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

### Release Process

1. Update `VERSION` file
2. Update `CHANGELOG.md`
3. Create a tag:
   ```bash
   git tag -a v1.2.3 -m "Release v1.2.3"
   git push origin v1.2.3
   ```
4. Build release artifacts:
   ```bash
   make release
   ```

### Code of Conduct

This project follows the [Contributor Covenant](https://www.contributor-covenant.org/) Code of Conduct. By participating, you are expected to uphold this code.