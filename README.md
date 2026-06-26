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
npm install -g @cortex.md/cli 
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
cortex plugin dev --vault /path/to/vault
```

### Create a Theme

```bash
cortex theme create my-theme
cd my-theme
cortex theme dev --vault /path/to/vault
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

Publishing prepares local release artifacts in `dist/cortex-build`: `build.json`, release
notes, Marketplace-ready assets, and an installable ZIP. The scaffolded GitHub Actions CD workflow
reads the generated build manifest, creates the matching `v*` tag when needed, and publishes those
artifacts to GitHub Releases. After the release is live, run `cortex registry submit` locally to
open the registry pull request from the generated build manifest.

## Commands

### Global

- `cortex init` - Initialize a new project interactively
- `cortex login` - Authenticate with GitHub
- `cortex logout` - Remove stored credentials

### Plugin

- `cortex plugin create [name]` - Create a new plugin
- `cortex plugin dev --vault <path>` - Start vault-scoped development mode with hot reload
- `cortex plugin build` - Build plugin for production
- `cortex plugin validate` - Validate plugin structure and security
- `cortex plugin doctor` - Run diagnostics
- `cortex plugin publish` - Prepare local release artifacts
- `cortex plugin link --vault <path>` - Link for vault-scoped local development
- `cortex plugin unlink --vault <path>` - Unlink development plugin from a vault
- `cortex plugin search [query]` - Search registry
- `cortex plugin install [id]` - Install from registry
- `cortex plugin update [id]` - Update installed plugin

### Theme

- `cortex theme create [name]` - Create a new theme
- `cortex theme dev --vault <path>` - Start vault-scoped theme development with hot reload
- `cortex theme link --vault <path>` - Link theme for vault-scoped local development
- `cortex theme unlink --vault <path>` - Unlink development theme from a vault
- `cortex theme publish` - Prepare local release artifacts

### Registry

- `cortex registry submit [build-json]` - Open a registry pull request from prepared release build metadata

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
    publish/           Local publish artifact preparation
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
