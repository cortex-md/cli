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

Prepare local release artifacts:

```bash
cortex plugin publish
```

The command writes `dist/cortex-publish/publish.json`, release notes, release assets, and an
installable ZIP. Push a version tag to let GitHub Actions create or update the GitHub Release:

```bash
git tag v{{VERSION}}
git push origin v{{VERSION}}
```

After the release is live, submit the Marketplace registry PR from your machine:

```bash
cortex login
cortex registry submit
```

## GitHub Actions

This template includes automated workflows in `.github/workflows/`:

- `ci-plugin.yml` validates plugin build and strict checks on push/PR
- `cd-plugin.yml` publishes on tag push or manual dispatch

The CD workflow uses GitHub's built-in `GITHUB_TOKEN` to publish the release. No `CORTEX_TOKEN`
secret is required unless you intentionally customize the workflow to submit registry PRs in CI.

## License

MIT
