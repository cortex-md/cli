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

## GitHub Actions

This template includes automated workflows in `.github/workflows/`:

- `ci-theme.yml` runs strict theme validation on push/PR
- `cd-theme.yml` publishes on tag push or manual dispatch

For automated publish, ensure `GITHUB_TOKEN` is available in workflow context.

## License

MIT
