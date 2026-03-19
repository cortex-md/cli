# Cortex CLI

Official command-line tool for developing plugins and themes for Cortex.

## Features

- **Plugin Development**: Create, build, validate, and publish plugins
- **Theme Development**: Create and manage custom themes
- **GitHub Integration**: Automated publishing with GitHub releases
- **Developer Experience**: Hot reload, validation, and diagnostics
- **Registry Management**: Search, install, and update plugins/themes

## Installation

### Via npm (recommended)

```bash
npm install -g cortex-cli
```

### From source

```bash
git clone https://github.com/cortex/cli
cd cli
make build
make install
```

## Quick Start

### Create a Plugin

```bash
cortex plugin create my-awesome-plugin
cd my-awesome-plugin
bun install
bun run dev
```

### Create a Theme

```bash
cortex theme create my-theme
cd my-theme
```

### Authenticate with GitHub

```bash
cortex login
```

### Build and Validate

```bash
cortex plugin build
cortex plugin validate
```

### Publish

```bash
cortex plugin publish
```

## Commands

### Global

- `cortex init` - Initialize a new project interactively
- `cortex login` - Authenticate with GitHub
- `cortex logout` - Remove stored credentials

### Plugin

- `cortex plugin create [name]` - Create a new plugin
- `cortex plugin dev` - Start development mode with hot reload
- `cortex plugin build` - Build plugin for production
- `cortex plugin validate` - Validate plugin structure and security
- `cortex plugin doctor` - Run diagnostics
- `cortex plugin publish` - Publish to registry
- `cortex plugin link` - Link for local development
- `cortex plugin unlink` - Unlink development plugin
- `cortex plugin search [query]` - Search registry
- `cortex plugin install [id]` - Install from registry
- `cortex plugin update [id]` - Update installed plugin

### Theme

- `cortex theme create [name]` - Create a new theme

### Registry

- `cortex registry sync` - Sync local cache

## Configuration

Global config is stored at `~/.config/cortex/config.json`:

```json
{
  "github": {
    "client_id": "",
    "registry_repo": "cortex/registry",
    "default_owner": ""
  },
  "paths": {
    "plugins_dir": "~/.cortex/plugins",
    "themes_dir": "~/.cortex/themes"
  },
  "log": {
    "level": "info"
  }
}
```

## Development

### Build

```bash
make build
```

### Test

```bash
make test
```

### Format

```bash
make fmt
```

### Lint

```bash
make lint
```

## Architecture

```
apps/cli/
  cmd/cortex/          Entry point
  internal/
    app/               Command wiring
    auth/              GitHub device flow
    config/            Global configuration
    dev/               Development workflow
    fsx/               Filesystem utilities
    github/            GitHub API integration
    plugin/            Plugin operations
    registry/          Registry integration
    theme/             Theme operations
    ux/                Terminal output
  pkg/
    manifest/          Manifest parsing
    semver/            Semantic versioning
    zipx/              Archive operations
```

## License

MIT
