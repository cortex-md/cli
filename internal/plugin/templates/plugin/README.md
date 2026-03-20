# {{NAME}}

{{DESCRIPTION}}

## Development

Install dependencies:

```bash
bun install
```

Start development mode with hot reload:

```bash
bun run dev
```

## Build

Build the plugin for production:

```bash
bun run build
```

## Validate

Validate your plugin before publishing:

```bash
bun run validate
```

## Publish

Publish to the Cortex plugin registry:

```bash
cortex plugin publish
```

## GitHub Actions

This template includes automated workflows in `.github/workflows/`:

- `ci-plugin.yml` validates plugin build and strict checks on push/PR
- `cd-plugin.yml` publishes on tag push or manual dispatch

For automated publish, ensure `GITHUB_TOKEN` is available in workflow context.

## License

MIT
