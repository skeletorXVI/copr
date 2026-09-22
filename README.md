# COPR packages

RPM packaging and GitHub release automation for COPR packages.

The packaging and automation in this repository are [BSD 3-Clause licensed](LICENSE). Each packaged application keeps its own upstream license, which its spec records in `License:`.

| Package                                   | COPR project        | Upstream                                                                                            | Packaging                                                     |
| ----------------------------------------- | ------------------- | --------------------------------------------------------------------------------------------------- | ------------------------------------------------------------- |
| `zen-browser`                             | `zen-browser`       | [Zen Browser](https://github.com/zen-browser/desktop)                                               | Official Linux binary, desktop launcher, icons, update policy |
| `zed`                                     | `zed`               | [Zed](https://github.com/zed-industries/zed)                                                        | Standard upstream source build, RPM update instructions       |
| `opencloud-desktop`                       | `opencloud-desktop` | [OpenCloud Desktop](https://github.com/opencloud-eu/desktop)                                        | Source build                                                  |
| `opencloud-desktop-dolphin`               | `opencloud-desktop` | [Dolphin integration](https://github.com/opencloud-eu/desktop-shell-integration-dolphin)            | Source build, KDE Frameworks 6                                |
| `opencloud-desktop-nautilus`              | `opencloud-desktop` | [Nautilus integration](https://github.com/opencloud-eu/desktop-shell-integration-nautilus)          | Upstream Python extension, installed with CMake               |
| `opencloud-desktop-integration-resources` | `opencloud-desktop` | [Shared integration resources](https://github.com/opencloud-eu/desktop-shell-integration-resources) | Icons and CMake discovery metadata                            |
| `libre-graph-api-cpp-qt-client`           | `opencloud-desktop` | [Libre Graph Qt client](https://github.com/opencloud-eu/libre-graph-api-cpp-qt-client)              | Shared library and development subpackage                     |

Packaging sources live at `packages/<copr-project>/<package>/`, so each directory tree is exactly what one COPR project publishes. A package directory holds:

| File          | Role                                                                                            |
| ------------- | ----------------------------------------------------------------------------------------------- |
| `main.go`     | The package definition: its upstream, its policy and its spec. Hand-written.                    |
| `<name>.spec` | Generated from `main.go`. The only place the pinned version, release number and changelog live. |
| `sources`     | Generated. SHA256 of each downloaded source, verified at build time.                            |
| `README.md`   | Packaging notes for this package: what upstream ships and what the spec does.                   |
| anything else | Local sources referenced by the spec, such as a desktop entry. Hand-written.                    |

Each project directory carries a `README.md` of its own: how to create its COPR project, and anything that spans its packages rather than belonging to one of them.

`internal/core` knows nothing about any particular package: it defines the `Releases` interface, the `Spec` struct and the generator runtime. A package decides for itself what an upstream release is by implementing `Releases`; `internal/core/github` provides the two implementations these packages happen to need.

## Getting started

### Prerequisites

- [mise](https://mise.jdx.dev/getting-started.html) installed — manages the tooling
- [Docker](https://docs.docker.com/get-docker/) installed
- [Docker Compose v2](https://docs.docker.com/compose/install/) installed

> Docker is **not** managed by mise — install it (and the Compose v2 plugin) yourself.

### First Time Setup

Install the runtimes and development tooling pinned in `mise.toml`:

```bash
mise install
```

Make sure [mise is activated in your shell](https://mise.jdx.dev/getting-started.html#activate-mise) so the managed tools land on your `PATH`. Run `mise tasks` at any time to list the available development tasks.

### Commands

Run test suite

```bash
mise run test
```

Format

```bash
mise run fmt
```

Check code

```bash
mise run check
```

Check for all new releases

```bash
mise run releases
```

Regenerate all spec files

```bash
mise run generate
```

## Package commands

Each `main.go` is the whole interface to its package, and the workflows call nothing else:

```sh
go run packages/zed/zed/main.go config        # where and how is this built?
go run packages/zed/zed/main.go check         # what does upstream offer?
go run packages/zed/zed/main.go gen 1.2.3     # write the spec and checksums for 1.2.3
go run packages/zed/zed/main.go gen           # regenerate at the packaged version
```

`config` is how the workflows learn everything package-specific, so no workflow carries a rule about any particular package:

```
project=zed
name=zed
chroots=fedora-43-x86_64 fedora-44-x86_64 fedora-45-x86_64 fedora-rawhide-x86_64
releases=43 44 45 rawhide
network=on
rebuild=false
verify=
```

`Chroots` is where the package is built, and it is the package's decision: one that reaches `aarch64`, or that only builds on newer Fedora releases, just says so and the matrices follow. `releases` is the distinct Fedora releases behind them, because building a source RPM does not depend on the architecture. `Network` is for a build that cannot run offline — Zed's Cargo and WebRTC downloads. `Verify` is a command run against what the rebuilt RPM installs; setting it asks the pull request check to rebuild the binary RPM and run it, which is cheap for Zen's repackaged binaries and prohibitive for Zed.

`check` asks the package's own `Releases` implementation, so what counts as a release is the package's decision — a GitHub release for most, the highest version tag for Libre Graph, and whatever an upstream that is not on GitHub needs. It writes nothing and reports `key=value` lines, which a workflow step appends straight to `$GITHUB_OUTPUT`:

```
current=1.20.1
version=1.20.2
update=true
upstream=https://github.com/zed-industries/zed
```

`gen` is told which version to write; it never asks upstream what to package next. The committed spec file records what _is_ packaged, so `gen` with no version regenerates that, and `check` compares against it. Output depends only on the definition and the version, so regenerating an unchanged package rewrites the same bytes.

`Release` restarts at `1` for a new version and is otherwise kept as it stands, which is what makes regeneration reproducible. A packaging change that must reach COPR as a new build names its release explicitly:

```sh
go run packages/zed/zed/main.go -release 2 gen 1.20.1
```

A downgrade is refused wherever it comes from, upstream or the command line.

The `%changelog` is generator-owned for the same reason. An entry is written when the version or release moves, dated that day, and then carried forward untouched — regenerating never redates it. That date is not cosmetic: Fedora derives `SOURCE_DATE_EPOCH` from the newest entry and clamps build mtimes to it, so a committed date is what lets two builds of one NEVR agree, and `add-determinism` normalizes the rest of the buildroot automatically. This does not make the builds reproducible — the buildroot is unpinned — but it removes the cheapest source of drift. Entries are authored by `RELEASE_BOT_EMAIL`, which has no default: writing one without an address fails rather than inventing an author, so pinning a new version locally means setting it. `RELEASE_BOT_NAME` falls back to `copr-bot`. Regenerating what is already packaged writes no entry and needs neither.

## Automation

Three workflows, each fanning out into one job per package, so a failure names the package that caused it. None of them knows what a package is: they list `packages/*/*` and run its `main.go`.

**Upstream releases** — every six hours at minute 17, one job per package. It runs `check`, and if there is an update, `gen`, then commits to the branch named after the package's path, `<project>/<package>`, and opens or updates its pull request:

```
zed/zed                                   opencloud-desktop/opencloud-desktop-nautilus
```

The bot owns those branches. A newer release force-pushes over a pending one — replacing an unmerged 1.20.1 with 1.20.2 is the point — but a branch that would not change is left untouched, so an open pull request is not rewritten every six hours. If a branch's last commit is not the bot's, the job refuses to overwrite it.

**Check** — on every pull request. One job lints and tests the repository; the rest run only for the packages the pull request touches, so a bot update to one package validates that package alone. A plan job asks each changed package for its `config` and builds the matrices from the answers: regenerate the spec and fail if the committed file differs, build the source RPM once per Fedora release that package targets, and — only for packages that set `Verify` — rebuild the binary RPM, install it and run the command they named.

**COPR** — on push to `main`. It rebuilds the packages that merge changed, building each source RPM in its own job and handing it to COPR with `copr-cli build`, using the chroots and the network setting that package reports. Manual dispatch rebuilds everything.

Source RPMs are built in GitHub Actions by `.github/actions/source-rpm`, which downloads the sources the spec lists, verifies them against the committed `sources` digests, and runs `rpmbuild -bs`. COPR never clones this repository, and its build phase runs offline unless the package says otherwise. Each package belongs to the COPR project named by its directory: `zen-browser`, `zed`, or `opencloud-desktop` for the five OpenCloud packages. Packages are submitted independently and in parallel, so a consumer's build may reach COPR before a dependency rebuilt in the same merge; it resolves against whatever that project holds at the time.

Every package decides for itself whether a discovered release should be packaged at all, by implementing `core.Policy`:

```go
type Policy interface { Decide(Update) Action }
```

`Decide` answers `Review` (open or update a pull request) or `Skip` (stay where we are); the core ships `core.ReviewAll{}`, which every package currently uses. A `Skip` is reported by `check` as `update=false`, so the workflow never opens a pull request for it. Every update is reviewed: nothing here writes to `main`, and merging a pull request is what triggers a COPR build.

## GitHub setup

### Secrets

`COPR_CONFIG` belongs to a GitHub environment named `copr`, which must be created first; the other is a repository secret. Environment approval rules are optional, and requiring approval makes publishing manual.

| Secret        | What it is                                                                    | How to provide                                                                                                                                                                                                                                                                                             |
| ------------- | ----------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `COPR_CONFIG` | The credentials `copr-cli` publishes source RPMs with.                        | Generate an API token at [COPR API credentials](https://copr.fedorainfracloud.org/api/) and paste the complete configuration, including its `[copr-cli]` section. The account must have builder access to the projects.                                                                                    |
| `GH_PAT`      | The identity the updater pushes update branches and opens pull requests with. | Create a fine-grained personal access token for this repository with **Contents: read/write** and **Pull requests: read/write**. It never needs to push `main`. A dedicated token is required because pushes made with the default `GITHUB_TOKEN` do not trigger the pull request checks. Keep it renewed. |

### Variables

| Variable            | What it is                                                                                                                                                                                                                                                                                                                          | How to provide                                                                                                                                                      |
| ------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `COPR_OWNER`        | The account or group owning the COPR projects. Required; until it is set, the COPR workflow is skipped.                                                                                                                                                                                                                             | The COPR username, or `@group` for a group, so builds are submitted to `$COPR_OWNER/<project>`.                                                                     |
| `RELEASE_BOT_NAME`  | Commit author name for update branches, and the author of generated changelog entries. Optional.                                                                                                                                                                                                                                    | Any name; defaults to `copr-bot`.                                                                                                                                   |
| `RELEASE_BOT_EMAIL` | Commit author email for update branches, the author of generated changelog entries, and the identity that decides which update branches the bot may rewrite. Required, with no default: the Upstream releases workflow fails on a dedicated first step without it, and pinning a new version fails rather than inventing an author. | Any address the bot should commit as, for example `copr-bot@fabian-haenel.dev`. Changing it later makes the updater refuse branches its previous identity authored. |

## Adding a package

Create `packages/<copr-project>/<package>/main.go`, calling `core.Run` with a `core.Definition`: its project and name (which must match the two directories), the `Chroots` it is built for, its `Releases` implementation, and a function returning the `core.Spec`. Then pin the first release:

```sh
go run packages/<copr-project>/<package>/main.go check     # what is the current version?
go run packages/<copr-project>/<package>/main.go gen 1.0.0 # write the spec and checksums
```

No core file, registry or workflow change is needed — every workflow lists `packages/*/*` and runs whatever it finds. A new COPR project also needs to be created once, with the chroots its packages name; give the project directory a `README.md` saying so.

If an upstream is not on GitHub, implement `Releases` in that package or extend the core:

```go
Releases: core.ReleasesFunc(func(ctx context.Context) (core.Release, error) {
    // Ask whatever the upstream actually is, and return its version.
}),
```

A new package directory should carry a `README.md` with its packaging notes, as the existing ones do.
