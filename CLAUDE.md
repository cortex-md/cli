# CLAUDE.md

## Project Overview

Cortex CLI is the Go tool plugin and theme authors use to build, test and publish to the Cortex
Marketplace. Its contract is the Cortex desktop app (`cortex-md/cortex-app`) and the registry
(`cortex-md/registry`): everything it generates must load in the app and pass the registry review.

User workflow: `create → dev → check → release → submit`. Keep the command surface that small.

## Architecture

```text
cmd/cortex/            entry point; prints errors through ux.PrintError
internal/
  app/                 cobra commands only (create, dev, check, pack, release, submit, login, logout, whoami, hidden legacy aliases)
  auth/                GitHub device flow, keychain storage, token resolution
  build/               release assets, build.json, metadata resolution, publish readiness checks
  config/              ~/.config/cortex/config.json (last vault)
  dev/                 vault resolution, atomic copy install, file watcher
  github/              go-github adapter (stateless; owner/repo per call)
  plugin/              plugin scaffold, build, bundle inspection, checks, release files
  project/             loads manifest.json and detects plugin vs theme
  registry/            registry entries, remote preflight, fork sync and pull request flow
  release/             version bump, git operations, waiting for the GitHub release
  report/              check results rendered as ✓ / ! / ✗ with fixes
  rules/               shared validation rules mirrored from the app and the registry
  scaffold/            template rendering and template versions
  theme/               theme scaffold, checks, release files
  ux/                  colors, spinner tasks, prompts (huh), problems with fixes
pkg/
  manifest/            plugin and theme manifest types
  semver/              semantic version parsing
```

## Contract with the app

- Plugins load as CommonJS from `<vault>/.cortex/plugins/<id>/<main>`; only `@cortex.md/api`,
  `@codemirror/state` and `@codemirror/view` resolve at runtime.
- The bundle's `definePlugin` manifest id must equal `manifest.json` id. Templates import
  `manifest.json` to keep one source of truth.
- `rules.PluginPermissions` mirrors `PLUGIN_PERMISSIONS` in `packages/plugin-api/src/definePlugin.ts`.
- The release `manifest.json` mirrors `permissions` into `capabilities` because the Marketplace
  catalog reads `capabilities`.
- `styles.css` requires `contributes.markdown`.
- Themes: `name == id`, CSS scoped to `body.theme-<name>-<scheme>`.
- Versions are plain `MAJOR.MINOR.PATCH`; the release tag is `v<version>`.
- `plugin/inspect.cjs` reproduces the app's CommonJS loader; keep it in sync with
  `packages/plugin-host-core/src/communityPlugins.ts`.

## Contract with the registry

- Entries: `id`, `name`, `author`, `authorUrl?`, `description`, `coverImageUrl`, `repo`, sorted by
  id, tab-indented. `registry.CanonicalJSON` must match the registry validator output.
- `registry.RunPreflight` mirrors the remote checks of `scripts/validate-registry.mjs`, plus the
  ownership rule of the registry PR labels workflow and `blocked.json`.
- The static bundle rules in `internal/rules` and `internal/plugin/check.go` are mirrored by
  `registry/scripts/lib/bundle-scan.mjs` (daily audit). Change both together.
- `registry.FetchStats` reads `stats.json` from the registry `data` branch (`cortex stats`).
- Pull requests come from a synced fork on the branch `registry/<kind>-<id>`; an open pull request
  for that branch is updated instead of duplicated.

## Code Standards

- no comments in source code
- clear names, small single-purpose functions
- user-facing failures are `ux.Problem{Title, Fix, Err}`; every error tells the user what to do
- long operations run inside `ux.Task`
- prompts go through `ux.Confirm`/`ux.RunForm` and must have a flag alternative for CI
- pass `context.Context` through network calls

## Terminal UX

- brand red `#FF3B38` for marks and active steps, gold `#E8A83C` for commands and highlights
- semantic colors for success, warning and error in `internal/ux`
- the large octopus logo only appears in interactive `cortex create`

## Testing

```bash
go test ./...
```

Do not add tests only to raise coverage.
