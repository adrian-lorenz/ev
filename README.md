# ev

**Local encrypted secret manager — built for the age of AI coding agents.**

```
$ cd my-project
$ ev set DATABASE_URL
Value for DATABASE_URL: ••••••••••••

$ ev run uvicorn main:app --reload   # no eval, no --, just works
```

---

## The Problem

Modern AI coding agents (Claude Code, Cursor, Copilot, Codex) have read access to your entire project directory. That means any `.env` file, `secrets.tfvars`, or `config.local.json` sitting in your repo is potentially visible to the agent — and could end up in logs, prompts, or completions.

ev moves secrets **out of your project** entirely.

```
before                          after
──────────────────────          ──────────────────────────────
my-project/                     my-project/
  .env          ← AI sees this    .envault      ← just the project name
  app.py                          app.py
  ...                             ...
                                ~/.envault/
                                  vault.json  ← AES-256 encrypted
```

---

## Features

- **AES-256-GCM encryption** with Argon2id key derivation
- **Zero secrets in your project** — only a `.envault` file with the project name
- **Auto-detects project** from directory name — searches up the tree like git, stops at `.git`
- **HTMX web UI** (`ev manage`) for visual secret management
- **Session support** — `ev open` unlocks once, no prompts for 8 hours
- **Session auto-refresh** — `set`, `delete`, `import`, and web UI keep the session in sync
- **Shell integration** — works with bash, zsh, fish
- **Terraform ready** — `ev run terraform plan` injects `TF_VAR_*` automatically
- **Import** from `.env`, `secrets.tfvars`, and Terraform map blocks
- **Backup / restore** — `ev backup` and `ev restore`
- **Change master password** — `ev passwd` (auto-creates a backup first)
- **Single binary, ~9 MB**, no daemon, no cloud

---

## Installation

```bash
curl -fsSL https://raw.githubusercontent.com/adrian-lorenz/ev/main/install.sh | bash
```

Or with Go:

```bash
go install github.com/adrian-lorenz/ev@latest
```

Or build from source:

```bash
git clone https://github.com/adrian-lorenz/ev
cd ev
make build
```

---

## Quick Start

```bash
# 1. Go to your project
cd my-project

# 2. (Optional) name the project — otherwise the directory name is used
ev init

# 3. Store secrets
ev set DATABASE_URL
ev set OPENAI_API_KEY

# 4a. Run directly (recommended)
ev run uvicorn main:app --reload

# 4b. Or load into your shell session
eval "$(ev load)"
python app.py
```

---

## Commands

| Command | Description |
|---|---|
| `ev init [name]` | Create `.envault` in the current directory |
| `ev info` | Show project, vault, session status and usage hints |
| `ev set KEY [VALUE]` | Add or update a secret |
| `ev get KEY` | Print a secret value |
| `ev list` | List all secret keys |
| `ev load` | Output `export` statements for `eval` |
| `ev run <cmd> [args...]` | Run a command with secrets injected |
| `ev delete KEY` | Remove a secret |
| `ev import <file>` | Import from `.env` or `.tfvars` |
| `ev open [--ttl 8h]` | Unlock vault for a timed session |
| `ev close` | Revoke the current session |
| `ev session` | Show session status |
| `ev passwd` | Change the master password |
| `ev backup [file]` | Back up the vault |
| `ev restore <file>` | Restore vault from a backup |
| `ev manage` | Open the web UI |
| `ev keychain save\|check\|delete` | macOS Keychain integration |

All commands accept:
- `-p, --project <name>` — override project (default: `.envault` file or directory name)
- `--vault <path>` — override vault location (default: `~/.envault/vault.json`)

---

## `ev run` — the right way to inject secrets

Unlike `eval "$(ev load)"`, `ev run` injects secrets directly into the child process without exposing them in the shell. No `--` needed.

```bash
ev run uvicorn main:app --reload
ev run python main.py
ev run go run ./...
ev run npm start
ev run terraform plan
```

With an active session (`ev open`), no password prompt appears.

**Terraform**: `ev run terraform plan` automatically adds `TF_VAR_<key>` for every secret, so Terraform picks them up without a prompt.

---

## Sessions — unlock once, work for hours

```bash
ev open           # prompt password once → valid for 8h
ev open --ttl 4h  # custom duration

ev run uvicorn main:app --reload   # no password prompt
eval "$(ev load)"                  # no password prompt
ev get DB_PASSWORD                 # no password prompt

ev session        # check remaining time
ev close          # revoke early
```

Sessions are automatically refreshed when you `set`, `delete`, or `import` secrets — no need to `ev close` and `ev open` again.

---

## `ev info` — see everything at a glance

```bash
$ ev info

  ev · local secret manager
  ─────────────────────────────────────────
  Project   : payment-service  (.envault file)
  Vault     : /Users/you/.envault/vault.json  (4.2 KB)
  Session   : open · expires 22:15:00  (7h 43m remaining)
  Keychain  : not set
  Secrets   : 8 (from session)
  .envault  : /Users/you/projects/payment-service/.envault

Detected project type: Python

── How to use your secrets ───────────────────────────────────
  ...
──────────────────────────────────────────────────────────────
```

Also shown automatically after `ev import`.

---

## Shell Integration

### bash / zsh

```bash
eval "$(ev load)"

# Shortcut for ~/.zshrc
evload() { eval "$(ev load "$@")"; }
```

### fish

```fish
ev load --shell fish | source
```

### direnv (auto-load on cd)

Add to your project's `.envrc`:

```bash
eval "$(ev load)"
```

Then run `direnv allow`.

---

## IDE Integration (PyCharm, VS Code, …)

IDE run configurations start their own processes — `eval "$(ev load)"` in a terminal doesn't help them.

**Recommended workflow:**

```bash
ev open    # once in terminal, valid 8h
```

Then in the PyCharm Run Configuration:

```
ev run uv run .venv/bin/python -m uvicorn main:app --reload
```

No password prompt, no `--`, PyCharm manages the process normally (debugger, hot reload all work).

### macOS Keychain (alternative to sessions)

```bash
ev keychain save    # store master password in Keychain
ev keychain check   # verify
ev keychain delete  # remove

ev run --keychain uvicorn main:app --reload   # silent, reads from Keychain
```

---

## Web UI

```bash
ev manage
```

Opens `http://localhost:7777` — browse projects, add / reveal / edit / delete secrets. Fully offline, bound to `127.0.0.1` only. Changes made in the UI automatically refresh any active session.

---

## Project Detection

ev finds the project name by searching **up the directory tree** for `.envault` — works from any subdirectory, just like git. The search stops at a `.git` boundary so it never leaks into unrelated parent repositories.

```bash
# project root has .envault with "project=payment-service"
cd ~/projects/payment-service/src/api
ev load    # correctly finds "payment-service"
```

Priority:
1. `--project` flag
2. `.envault` file (searched upward, stops at `.git`)
3. Current directory name

The `.envault` file contains no secrets — safe to commit.

---

## Import from `.env` or `.tfvars`

Already have secrets in a file? Import them in one step and remove the file for good. After import, ev prints a usage guide based on the detected project type.

### From .env

```bash
ev import .env --dry-run   # preview
ev import .env             # import
rm .env
echo ".env" >> .gitignore

ev run uvicorn main:app --reload
```

Supported formats:

```bash
DB_HOST=localhost
DB_PASSWORD="my secret"
export API_KEY=sk-abc123   # export prefix stripped
# comments and // and /* */ are stripped from values
```

### From secrets.tfvars

```bash
ev import secrets.tfvars --dry-run
ev import secrets.tfvars
rm secrets.tfvars
echo "*.tfvars" >> .gitignore

ev run terraform plan   # TF_VAR_* injected automatically
```

Supported tfvars formats:

```hcl
db_password = "secret"                  # simple string
db_password = "secret"  # comment       # inline comments stripped

global_secrets = {                      # flat map → imported as individual secrets
  "api-key"  = "sk-abc"                 #   keys: hyphens → underscores
  "db-pass"  = "hunter2"
}

github_repos = {                        # map with complex keys → stored as single value
  "org/repo" = "ref:refs/heads/main"
}
```

### Flags

```bash
ev import .env --dry-run        # preview without saving
ev import .env --no-overwrite   # skip keys that already exist
```

---

## Backup & Restore

```bash
ev backup                    # → ~/.envault/backups/vault-<timestamp>.json
ev backup ~/Desktop/ev.bak   # custom path

ev restore ~/.envault/backups/vault-2026-03-16T12-00-00.json
```

`ev restore` automatically saves a pre-restore backup before overwriting.

`ev passwd` also creates a timestamped backup before changing the password.

---

## Security

| Property | Detail |
|---|---|
| Encryption | AES-256-GCM (authenticated encryption) |
| Key derivation | Argon2id — `time=3, memory=64MB, threads=4` |
| Storage | `~/.envault/vault.json` with mode `0600` |
| Directory | `~/.envault/` with mode `0700` |
| Writes | Atomic via temp file + rename |
| Nonce | Fresh random nonce on every save |
| Password input | Terminal, no echo — never via CLI flags |
| Web server | Binds to `127.0.0.1` only, CSRF origin check |
| Sessions | AES-256-GCM encrypted, stored in `~/.envault/sessions/` |

**The master password is never stored.** There is no recovery. Keep it somewhere safe (1Password, etc.).

---

## How it works

```
ev set DB_PASSWORD
        │
        ▼
  prompt password
        │
        ▼
  Argon2id(password, random_salt) → 32-byte key
        │
        ▼
  AES-256-GCM encrypt JSON payload
        │
        ▼
  ~/.envault/vault.json
  {
    "version": 1,
    "salt":       "...",   // new every save
    "nonce":      "...",   // new every save
    "ciphertext": "..."    // encrypted project + secrets map
  }
```

---

## License

MIT
