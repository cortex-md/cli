# {{DISPLAY_NAME}}

{{DESCRIPTION}}

A custom theme for Cortex.

## Installation

Install via Cortex CLI or manually place this theme in your Cortex themes directory.

## Color Schemes

This theme includes both dark and light variants:

- `theme-dark.css` - Dark color scheme
- `theme-light.css` - Light color scheme

## Customization

You can customize the theme by editing the CSS variables in the theme files.

## Publish

Prepare local release artifacts:

```bash
cortex theme publish
```

The command writes `dist/cortex-build/build.json`, release notes, release assets, and an
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

- `ci-theme.yml` runs strict theme validation on push/PR
- `cd-theme.yml` publishes on tag push or manual dispatch

The CD workflow uses GitHub's built-in `GITHUB_TOKEN` to publish the release. No `CORTEX_TOKEN`
secret is required unless you intentionally customize the workflow to submit registry PRs in CI.

## License

MIT
