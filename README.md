# Cortex CLI

Build, test and publish plugins and themes for [Cortex](https://github.com/cortex-md/cortex-app).

```bash
npm install -g @cortex.md/cli
```

## The whole workflow

```bash
cortex create              # scaffold a plugin or theme that loads in Cortex as-is
cortex dev --vault ~/Notes # install into a vault and reload on every change
cortex check               # build + every Marketplace check
cortex release patch       # bump, commit, tag and push; GitHub Actions publishes the release
cortex submit              # first release only: open the Marketplace registry pull request
```

Each command detects whether the current folder is a plugin or a theme from `manifest.json`.

## Commands

| Command | What it does |
| --- | --- |
| `cortex create [name]` | Scaffolds a plugin (`--plugin`) or theme (`--theme`) with CI and release workflows, initializes git and installs dependencies. `--update-ci` rewrites the workflows of an existing project. |
| `cortex dev` | Builds and copies the project into `<vault>/.cortex/plugins/<id>` or `<vault>/.cortex/themes/<id>`, then rebuilds on every change. Cortex reloads it automatically. `--once` installs and exits, `--clean` removes the development copy. |
| `cortex check` | Builds the plugin, loads the bundle exactly like Cortex does and runs the checks the registry applies. `--strict` fails on warnings. |
| `cortex release [patch\|minor\|major\|x.y.z]` | Runs the checks, bumps `manifest.json` and `package.json`, commits, tags `v<version>` and pushes. `--wait` follows the GitHub release and offers to submit it. |
| `cortex submit` | Verifies the latest GitHub release, then opens (or updates) a pull request to `cortex-md/registry` from your fork. `--dry-run` shows everything without touching GitHub. |
| `cortex stats` | Shows the Marketplace votes, downloads, stars, automated review result and block status of the current project. |
| `cortex pack` | Writes the release assets to `dist/cortex-build`. Used by the release workflow. |
| `cortex login` / `logout` / `whoami` | GitHub device login. `whoami` shows the account, where the token comes from and its scopes. |

`--yes` confirms prompts in scripts. Without a terminal (or with `CI` set) prompts are disabled and
missing values must be passed as flags.

## How publishing works

1. `cortex release` pushes a `v<version>` tag.
2. The `Release` workflow scaffolded by `cortex create` runs `cortex pack` and publishes the GitHub
   release with `manifest.json`, the bundle (or the two stylesheets) and `styles.css` when present.
3. Cortex reads the registry entry, downloads the latest release of `repo` and compares its tag with
   the installed version.

The registry entry has no version, so `cortex submit` is only needed for the first release and when
the name, description, author or cover image change. Every later release reaches users
automatically.

`cortex submit` also checks that you own the repository (or are a public member of its
organization) and that the id or version is not on the registry block list.

## Votes, downloads and review

The registry publishes `stats.json` on its `data` branch: 👍 votes from the item's GitHub
Discussion, downloads of the release bundle, stars and a daily automated review of the latest
release (the same static rules as `cortex check`). Maintainers can verify an item or pull a version.
Run `cortex stats` to see all of it for your project.

## Plugins

`manifest.json` is the single source of truth. The scaffolded `src/index.ts` imports it:

```ts
import { definePlugin, type PluginManifest } from "@cortex.md/api"
import manifest from "../manifest.json"

export default definePlugin({
	manifest: manifest as PluginManifest,
	contributes: { commands: [/* ... */] },
})
```

- Bundles are CommonJS for the browser. Only `@cortex.md/api`, `@codemirror/state` and
  `@codemirror/view` may stay external.
- `permissions` accepts `vault:read`, `vault:write`, `vault:delete`, `vault:watch`, `editor:read`,
  `editor:write`, `workspace:tabs`, `theme:read`, `bookmarks:read`, `bookmarks:write`, `data` and
  `notifications`. Contributions (commands, views, settings, markdown) need no permission.
- `styles.css` is only loaded for plugins with markdown contributions.
- Versions are plain `MAJOR.MINOR.PATCH`; pre-release suffixes break update checks.

## Themes

- `name` must equal `id`; Cortex applies `body.theme-<name>-dark` and `body.theme-<name>-light`.
- `colorschemes.dark` and `colorschemes.light` point at the two stylesheets.
- Variables you do not set fall back to the Cortex defaults.

## Authentication

`cortex submit` needs a GitHub token with `public_repo`. It uses, in order: `CORTEX_TOKEN`,
`GITHUB_TOKEN`, the token stored by `cortex login` (system keychain) and `gh auth token`.

## Development

```bash
make build-dev   # bin/cortex without GitHub login
make test
make lint
```

Production builds need the OAuth app client id: `make build GITHUB_CLIENT_ID=...`.

## License

MIT
