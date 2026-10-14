# {{DISPLAY_NAME}}

{{DESCRIPTION}}

## Develop

```bash
cortex dev
```

`cortex dev` installs the theme into your vault (`<vault>/.cortex/themes/{{ID}}`) and copies every
change. Cortex reloads the theme automatically; select **{{DISPLAY_NAME}}** in Settings.

- `theme-dark.css` styles `body.theme-{{ID}}-dark`
- `theme-light.css` styles `body.theme-{{ID}}-light`

Variables you do not set fall back to the Cortex defaults.

## Release

```bash
cortex check            # every Marketplace check
cortex release patch    # bump, commit, tag and push; GitHub Actions publishes the release
cortex submit           # first release only: open the Marketplace registry pull request
```

New releases reach Cortex users automatically once the theme is listed. Run `cortex submit` again
only when the name, description, author, cover image or screenshots change.

## License

MIT
