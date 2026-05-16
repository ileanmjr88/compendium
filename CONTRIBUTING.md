# Contributing to Compendium

Thanks for considering a contribution. Issues and pull requests are both
welcome.

For anything non-trivial (a new feature, a behavior change, a refactor
that touches multiple packages), please open an issue first to discuss
the approach before writing code. It saves both of us time.

## Dev environment

Requires Go 1.26.1 (see [`go.mod`](go.mod)).

```bash
git clone https://github.com/ileanmjr88/compendium.git
cd compendium
make build      # builds ./bin/compendium with version info baked in
make test       # runs the full test suite
```

The test suite is hermetic. Registry and installer tests use in-process
`httptest.NewServer`; no network required. Run a single test with:

```bash
go test -v -run TestName ./internal/config/
```

## Branch model

Git Flow lite:

- `main`: stable. Only tagged releases land here.
- `develop`: integration branch. All work branches from `develop` and
  merges back into `develop`.
- Feature branches: `feat/<name>`, `fix/<name>`, `docs/<name>`, etc.

**PRs target `develop`, not `main`.** Releases merge `develop` to `main`
and tag from `main`.

## Commit messages

Compendium uses [Conventional Commits](https://www.conventionalcommits.org/).
Subject line is lowercase, imperative, under ~70 characters.

```
feat: add support for swift in [languages]
fix: handle empty checksums.txt without panicking
test: cover arm-none-eabi-gcc activate path
docs: clarify [packages] semantics in compendium.toml docs
refactor: extract archive type detection into helper
```

Common prefixes: `feat`, `fix`, `test`, `docs`, `refactor`, `tool`, `chore`.

## Code style

Before committing:

```bash
make fmt        # gofmt -w .
make vet        # go vet ./...
make lint       # golangci-lint run
```

CI runs all three. PRs that fail any of them will be flagged.

## Testing

Add tests for new behavior. The repo uses Go's standard `testing` package
plus `httptest.NewServer` for network-adjacent code.

Patterns to mirror:

- [`internal/registry/client_test.go`](internal/registry/client_test.go): hermetic registry tests with an in-process HTTP server.
- [`internal/installer/download_test.go`](internal/installer/download_test.go): hermetic download + checksum verification.
- [`internal/config/loader_test.go`](internal/config/loader_test.go): TOML parsing and validation.

Avoid tests that hit the real public registry; they're flaky and slow.

## Filing issues

**Bugs.** Include:

- What you ran and what you expected.
- What actually happened (full error output if any).
- Output of `compendium version`.
- OS and architecture if not obvious from the version line.

**Feature requests.** Explain the use case before the implementation idea.
"I want to add support for X" is less helpful than "I'm trying to do Y
and the current behavior makes it hard because Z."

## Pull requests

- Target `develop`.
- One feature or fix per PR. Mixed PRs get harder to review.
- Include tests for new behavior or bug fixes.
- Run `make fmt && make vet && make lint && make test` locally before
  opening the PR.
- Reference the issue number if there is one.
- Update docs in the [`compendium-docs`](https://github.com/ileanmjr88/compendium-docs)
  repo if the change affects user-facing behavior. Link the docs PR from
  the code PR.

## Where else to contribute

This repo holds only the Compendium binary source code. Other contributions
go to sibling repos:

- **Adding a new tool or language version to the public registry:**
  [`compendium-registry`](https://github.com/ileanmjr88/compendium-registry).
- **Improving the docs site or written guides:**
  [`compendium-docs`](https://github.com/ileanmjr88/compendium-docs).

## License

By contributing, you agree that your contributions will be licensed under
the same GPL-3.0-or-later as the rest of the project. See [LICENSE](LICENSE).
