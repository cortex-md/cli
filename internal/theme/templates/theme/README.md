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
Theme variables are scoped to `body.theme-{{ID}}-dark` and `body.theme-{{ID}}-light`, matching the
classes Cortex applies when each colorscheme is active.

## Publish

Prepare local release artifacts:

```bash
cortex theme publish
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

- `ci-theme.yml` runs strict theme validation on push/PR
- `cd-theme.yml` publishes on main pushes, tag pushes, or manual dispatch

The CD workflow uses GitHub's built-in `GITHUB_TOKEN` to publish the release. No `CORTEX_TOKEN`
secret is required unless you intentionally customize the workflow to submit registry PRs in CI.

## License

MIT
