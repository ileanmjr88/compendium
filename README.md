# Compendium

Reproducible developer environments, declared in one config. Pin the
toolchains and tools your project needs in `compendium.toml`, and every
developer gets the same versions on every machine.

Think nvm + pyenv + rustup unified under one declarative config.

## Install

```bash
curl -fsSL https://compendium.ilean.me/install.sh | sh
```

Installs to `~/.local/bin/compendium`. No `sudo` required.

For pinning a specific version, choosing a custom install location, or
building from source, see the
[installation guide](https://compendium.ilean.me/getting-started/installation/).

## Quick example

Add a `compendium.toml` to your project:

```toml
[compendium]
name = "my-project"
version = "0.1.0"
min_compendium = "0.1.0"

[registry]
source = "public"

[languages]
go = "1.26.1"

[tools]
golangci-lint = "2.11.4"
```

Then:

```bash
compendium install               # download and extract everything
source <(compendium activate)    # add tools to PATH for this shell
go build ./...                   # uses Compendium's pinned Go
golangci-lint run                # uses Compendium's pinned linter
source <(compendium deactivate)  # restore original environment
```

Same `compendium.toml`, same versions on every machine.

## Build from source

Requires Go 1.26.1 (see [`go.mod`](go.mod)).

```bash
git clone https://github.com/ileanmjr88/compendium.git
cd compendium
make build      # produces ./bin/compendium
make test       # runs the test suite
make lint       # runs golangci-lint
```

## Links

- **Docs:** [compendium.ilean.me](https://compendium.ilean.me)
- **Roadmap** [compendium.ilean.me/roadmap](https://compendium.ilean.me/roadmap/)
- **Issues:** [github.com/ileanmjr88/compendium/issues](https://github.com/ileanmjr88/compendium/issues)
- **Contributing:** [CONTRIBUTING.md](CONTRIBUTING.md)
- **Registry (separate repo):** [github.com/ileanmjr88/compendium-registry](https://github.com/ileanmjr88/compendium-registry)

## License

[GPL-3.0-or-later](LICENSE). The source stays open: anyone can use, modify,
or redistribute Compendium, but modifications and derivative works must be
released under the same license. Running the tool carries no obligations.

Built by [Ilean Monterrubio Jr](https://ilean.me).
