# CLAUDE.md

## Project Overview

`apps/cli` hosts the Cortex official CLI built in Go for plugin and theme developers.

Primary goals:
- standardize plugin and theme workflows
- improve local development DX
- validate quality and security before publication
- automate publish and registry update flows

## Architecture

```text
apps/cli/
  cmd/cortex/           # CLI entry point
  internal/
    app/                # Command wiring (cobra commands)
    auth/               # GitHub device flow authentication
    config/             # Global config loading (~/.config/cortex/)
    dev/                # Development mode: file watcher, link/unlink
    fsx/                # Filesystem utilities
    github/             # GitHub API client (go-github wrapper)
    plugin/             # Plugin operations: create, build, validate, publish
    registry/           # Registry operations (future)
    theme/              # Theme operations: create, validate, publish
    ux/                 # Terminal output: colors, banner, lipgloss styling
  pkg/
    manifest/           # PluginManifest and ThemeManifest types
    semver/             # Semantic version parsing and comparison
    zipx/               # ZIP archive creation utilities
```

Guidelines:
- keep command wiring in `internal/app`
- keep external integrations behind focused adapters
- keep reusable logic in `pkg`
- keep business rules in `internal/*` use-case packages

## Implemented Commands

### Plugin Commands
- `cortex plugin create [name]` - Scaffold new plugin from template
- `cortex plugin build [dir]` - Build plugin with detected bundler (bun/npm/yarn/pnpm)
- `cortex plugin validate [dir]` - Validate structure, manifest, security, bundle size
- `cortex plugin dev [dir]` - Development mode with file watcher and auto-rebuild
- `cortex plugin link [dir]` - Symlink plugin to ~/.cortex/plugins/
- `cortex plugin unlink [dir|id]` - Remove symlink
- `cortex plugin publish [dir]` - Validate, build, zip, create GitHub release

### Theme Commands
- `cortex theme create [name]` - Scaffold new CSS theme from template
- `cortex theme validate [dir]` - Validate structure, manifest schema, CSS variables
- `cortex theme publish [dir]` - Validate, zip, create GitHub release

### Auth Commands
- `cortex login` - GitHub device flow authentication
- `cortex logout` - Remove stored credentials

## Code Standards

- no comments in source code
- use clear names for functions, types, variables, and files
- keep functions small and single-purpose
- return typed errors with contextual wrapping
- avoid hidden global state
- pass `context.Context` through IO and network boundaries

## Template System

Templates are embedded using `go:embed` and stored in:
- `internal/plugin/templates/plugin/` - Plugin scaffold files
- `internal/theme/templates/theme/` - Theme scaffold files

Template variables use `{{PLACEHOLDER}}` syntax:
- `{{ID}}` - Plugin/theme ID (lowercase, alphanumeric, hyphens)
- `{{NAME}}` - Display name
- `{{VERSION}}` - Initial version (default: 0.1.0)
- `{{DESCRIPTION}}` - Description text
- `{{AUTHOR}}` - Author name
- `{{CLASS_NAME}}` - PascalCase class name for TypeScript

## Security Validation

The validate command scans built output for forbidden patterns:

**Critical (blocks publish):**
- `eval()`, `new Function()` - Code injection risk
- `child_process`, `exec`, `spawn` - Shell access
- `@tauri-apps/api`, `@tauri-apps/plugin-*` - Direct Tauri access
- `__TAURI__`, `tauri.invoke` - Tauri globals
- `react-native`, `expo-*` - Direct React Native access
- `fs`, `node:fs` - Direct Node.js filesystem

**Warning:**
- `path`, `node:path` - Should use plugin-api utilities
- `dangerouslySetInnerHTML` - XSS risk
- `<script>` tags - Inline scripts

Plugins must use `@cortex/plugin-api` for all platform operations.

## Theme Validation

The theme validate command checks:

**Required CSS variables (error if missing):**
- `--bg-primary`, `--bg-secondary`
- `--text-primary`, `--text-secondary`
- `--accent-default`
- `--border-default`

**Recommended CSS variables (warning if missing):**
- `--bg-tertiary`, `--bg-elevated`, `--bg-hover`, `--bg-active`
- `--text-muted`, `--text-disabled`
- `--accent-hover`, `--accent-active`
- `--border-subtle`, `--border-strong`
- `--syntax-keyword`, `--syntax-string`, `--syntax-comment`, `--syntax-number`, `--syntax-function`

**Structure checks:**
- manifest.json must exist with valid schema
- All CSS files in colorschemes must exist
- CSS files should define variables in `:root`

**Size limits:**
- Warning: > 50 KB per CSS file
- Error (strict): > 200 KB per CSS file

## Config and Paths

- global config path: `~/.config/cortex/config.json`
- cache path: `~/.config/cortex/cache/`
- dev links path: `~/.cortex/plugins/`
- all path resolution must be centralized in `internal/config` and `internal/fsx`

## Authentication

- use GitHub device flow for terminal-first UX
- store tokens via OS keychain
- support `login`, `logout`, and auth status checks
- never persist tokens in plain text files

## Dependencies

Key external dependencies:
- `github.com/spf13/cobra` - CLI framework
- `github.com/AlecAivazis/survey/v2` - Interactive prompts
- `github.com/google/go-github/v60` - GitHub API
- `github.com/fsnotify/fsnotify` - File watching
- `github.com/charmbracelet/lipgloss` - Terminal styling
- `golang.org/x/oauth2` - OAuth2 for GitHub

## Build and Run

```bash
cd apps/cli

go build -o cortex cmd/cortex/main.go

./cortex --help
./cortex -v
./cortex plugin create my-plugin
./cortex plugin build
./cortex plugin validate
./cortex plugin dev
./cortex plugin publish --dry-run
./cortex theme create my-theme
./cortex theme validate
./cortex theme publish --dry-run
```

## Testing Strategy

- unit tests per package with table-driven patterns
- integration tests for HTTP/device flow with `httptest`
- fixture-based tests for validator and packaging flows
- keep tests parallel-safe when possible

Run tests:
```bash
go test ./...
go test ./pkg/semver/...
```

## Security and Reliability

- sanitize shell arguments and user-provided paths
- enforce request timeouts and retries where needed
- handle partial failures with resumable or idempotent behavior
- log structured events without sensitive data

## Documentation Expectations

When changing architecture, command contracts, or workflows in `apps/cli`, update:
- `apps/cli/CLAUDE.md` (this file)
- Root `CLAUDE.md` if CLI affects overall project
