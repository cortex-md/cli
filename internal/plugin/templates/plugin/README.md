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

The command writes `dist/cortex-build/build.json`, release notes, release assets, and an
installable ZIP. The CD workflow reads that build manifest, creates the matching version tag when
needed, and creates or updates the GitHub Release:

```bash
git push origin main
```

After the release is live, submit the Marketplace registry PR from your machine:

```bash
cortex login
cortex registry submit
```

## GitHub Actions

This template includes automated workflows in `.github/workflows/`:

- `ci-plugin.yml` validates plugin build and strict checks on push/PR
- `cd-plugin.yml` publishes on main pushes, tag pushes, or manual dispatch

The CD workflow uses GitHub's built-in `GITHUB_TOKEN` to publish the release. No `CORTEX_TOKEN`
secret is required unless you intentionally customize the workflow to submit registry PRs in CI.

## License

MIT
