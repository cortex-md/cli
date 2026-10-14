# {{NAME}}

{{DESCRIPTION}}

## Develop

```bash
bun install
cortex dev
```

`cortex dev` builds the plugin, installs it into your vault (`<vault>/.cortex/plugins/{{ID}}`) and
rebuilds on every change. Cortex reloads the plugin automatically. Enable community plugins in
Settings the first time, then enable **{{NAME}}**.

`manifest.json` is the single source of truth for the plugin identity: `src/index.ts` imports it,
so keep `id`, `version` and `permissions` there.

## Release

```bash
cortex check            # build + every Marketplace check
cortex release patch    # bump, commit, tag and push; GitHub Actions publishes the release
cortex submit           # first release only: open the Marketplace registry pull request
```

New releases reach Cortex users automatically once the plugin is listed. Run `cortex submit` again
only when the name, description, author, cover image or screenshots change.

## License

MIT
