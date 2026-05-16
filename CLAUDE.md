# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project

Compendium is a per-project developer environment manager written in Go. It reads a `compendium.toml` config file and sets up languages, tools, and hooks under `~/.local/compendium/`. Think nvm + pyenv + rustup unified under one declarative config.

User-facing docs live in the separate `compendium-docs` repo (Astro/Starlight, deploys to `compendium.ilean.me`). This repo holds only the binary's source.

Module path: `github.com/ileanmjr88/compendium`

## Build Commands

```bash
make build              # build binary to bin/compendium (ldflags inject version)
make test               # run all tests (go test -v ./...)
make fmt                # format code (gofmt -w .)
make lint               # lint (golangci-lint run)
make run                # go run ./cmd/compendium
make clean              # remove bin/ and go clean
go test -v -run TestName ./internal/config/   # run a single test
```

## Architecture

Entry: `cmd/compendium/main.go` → `internal/cli/root.go` (cobra CLI)

Layers (each in its own `internal/` package):

- **config** — `Load(path)` reads `compendium.toml`, unmarshals into structs, validates. Validation enforces required fields (`compendium.version`, `compendium.min_compendium`, `registry.source` which must equal `"public"` in v0.1) as errors. Warnings only fire for `target.platform`/`target.arch` outside the known set, or `fpu`/`linker_script` on non-embedded platforms. Unknown language/tool/package-manager names are not validated; the registry is the source of truth.
- **cli** — Cobra commands (`install`, `activate`, `deactivate`, `env`, `status`, `versions`, `version`). Version info injected via ldflags into `internal/buildinfo`. Note: `activate`/`deactivate` print shell scripts to stdout (for `source <(compendium activate)`), so they use `fmt.Fprintf(os.Stderr, ...)` for messages instead of `ui.Print`.
- **env** — `Paths` struct defining storage layout under `$HOME/.local/compendium/`. Subdirs: `languages/`, `tools/`, `envs/`. Methods: `LanguageDir()`, `ToolDir()`, `CheckDir()`, `EnsureDirs()`. Constructor `NewPathsWithRoot(root)` for testing.
- **registry** — HTTP client that fetches a JSON manifest and resolves `(kind, tool, version, platform, arch)` → `Artifact` (URL, checksum, size, strip). Manifest has separate `Languages` and `Tools` maps. `ResolveSource()` maps registry source to URL.
- **installer** — `Resolve()` builds `[]InstallItem` from config (parses `@` syntax for compiler refs). `FilterInstalled()` skips items already on disk. `DownloadVerify()` fetches manifest, downloads tarballs, verifies sha256 checksums, extracts to correct paths.
- **shell** — `ResolveEnv(cfg, paths)` walks installed items and returns `(dirs, envVars, warnings)`: bin directories for PATH, per-language env vars sourced from `manager.ForLanguage(...)`, and "not installed" warnings. `ActivateScript(dirs, envVars)` and `DeactivateScript()` generate POSIX shell scripts that manage PATH plus language env. Used via `source <(compendium activate)`.
- **manager** — Per-language env-var providers. `LanguageManager` interface (`Name`, `EnvVars(version, platform)`, `ProjectEnvVars(projectName, packages)`) returning `EnvVar{Name, Value, Action: set|unset|prepend}`. `ForLanguage(name, paths)` dispatches to per-language impls (Go, GCC, Clang, Python, Node, arm-none-eabi-gcc, riscv-none-elf-gcc) that produce vars like `GOROOT`, `CC`/`CXX`, `VIRTUAL_ENV`. Consumed by `shell.ResolveEnv` to fold language env into the activate script.
- **extract** — `Extract(filePath, destDir, strip)` handles tar.gz, tar.xz, tar.zst, tar.bz2 with strip-components support and file permission preservation.
- **ui** — `Print(symbol, message, detail)` and `PrintDetail()` for consistent colored terminal output. `ProgressBar()` for downloads. Symbols: `→` (action), `✓` (success), `✗` (fail), `↓` (download), `!` (warning).

### Key design decisions
- `Languages`, `Tools`, and `Packages` are all `map[string]string`
- Language values can include compiler refs: `c = "gcc@12.2.1"` (parsed via `@` split in `Resolve()`)
- If both `c` and `cpp` are set in `[languages]`, only `c` is installed (one compiler ships both drivers); `cpp` is skipped with a warning — see `installer.Resolve()`
- `[compendium].name` defaults to `filepath.Base(cwd)` when omitted; applied in `cli/install.go`, `cli/activate.go`, `cli/env.go`
- `compendium.version` and `compendium.min_compendium` are required by validation but their semantics (staleness bump, binary compatibility check) are not yet enforced
- Storage path formula: `filepath.Join(home, ".local", "compendium", section, tool, version)`
- Platform uses `darwin` (matches `runtime.GOOS`), not `macos`
- Checksum format in manifest: `sha256:<hex>`
- Strip-components per artifact in manifest, not auto-detected
- All user-facing output goes through the `ui` package
- Targets: macOS and Linux only

### Dependencies
- `github.com/pelletier/go-toml/v2` — TOML parsing
- `github.com/spf13/cobra` — CLI framework
- `github.com/schollz/progressbar/v3` — download progress bars
- `github.com/ulikunitz/xz` — xz decompression
- `github.com/klauspost/compress/zstd` — zstd decompression

### Testing
- Unit tests use `httptest.NewServer` in-process (see `internal/registry/client_test.go` and `internal/installer/download_test.go`) — hermetic, no network required.
- Manual end-to-end testing runs against the real public registry at `PublicIndexURL` (see `internal/registry/client.go`).
- `testdata/compendium.toml` — example config for manual end-to-end testing.

## Working Preferences

The developer prefers to write code by hand with guidance. Give patterns and explanations rather than auto-generating code. Fill in code only when explicitly asked. The developer is new to Go — explain Go concepts when they come up.
