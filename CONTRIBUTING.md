# Contributing to susfile

Thanks for looking. susfile is a single-binary CLI file-forensics visualiser
written in Go.

Read [`PROJECT.md`](PROJECT.md) first — it is the authoritative specification and
the engineering contract. When the spec and the code disagree, the spec wins;
changing it is a deliberate act, not a side effect of an implementation.

## Getting set up

You need Go 1.27 or newer and Git. That is the whole list — `CGO_ENABLED=0`
throughout, no C toolchain.

```bash
git clone https://github.com/kawaiipantsu/susfile.git
cd susfile
make build
./susfile version
```

`make help` lists every target.

## The pre-commit loop

```bash
make fmt vet test build
```

Run all four, every time, before you commit. CI runs the same plus the race
detector, the linter and a vulnerability scan.

Narrower runs while you work:

```bash
go test ./internal/analyze/ -run TestClassify -v
make race
make coverage        # writes coverage.html
make lint            # golangci-lint if installed, else go vet
```

Tests never need the network and never load a committed binary fixture — inputs
are `[]byte` literals or generated in-test. Cross-compilation is a hard
requirement; if you touch a dependency, verify it still holds:

```bash
CGO_ENABLED=0 GOOS=linux GOARCH=arm GOARM=7 go build ./...
make build-linux
```

## Branches

Git Flow, from day one.

| Branch | From | Merges to |
|---|---|---|
| `feature/<short-name>` | `develop` | `develop` |
| `fix/<short-name>` | `develop` | `develop` |
| `release/<version>` | `develop` | `main` **and** `develop` |
| `hotfix/<version-or-desc>` | `main` | `main` **and** `develop` |

- `main` is release history only. The **Branch flow** CI check fails any PR to
  `main` that is not from `release/*` or `hotfix/*`.
- `develop` is the integration branch and the base for feature branches.
- Merges use `--no-ff`.
- Releases are annotated `v`-prefixed tags on `main`.

```bash
git switch develop
git switch -c feature/block-classifier
```

## Commit messages

Conventional-commit prefixes:

```
feat:     a new capability
fix:      a bug fix
test:     tests only
docs:     documentation only
refactor: behaviour-preserving change
build:    build system, Makefile, cross-compilation, packaging
chore:    dependencies, housekeeping
ci:       CI workflows
```

Keep commits small and cohesive — one logical change each. Write the body when
the *why* is not obvious from the diff.

## Definition of done

- [ ] Implementation is complete — no reachable stubs.
- [ ] Tests exist where appropriate and pass.
- [ ] `make fmt vet test build` passes; cross-build passes if you touched deps.
- [ ] User-facing behaviour documented in `docs/` and `CHANGELOG.md` under
      `[Unreleased]`.
- [ ] Errors are handled — a malformed file yields a populated `Result`, not a
      panic and not a bare `return err`.
- [ ] No network code introduced. No writes to the analysed file.
- [ ] No committed binary fixtures.
- [ ] Code is formatted (`make fmt`).

## House rules

**A recoverable problem is a populated `Result`, not an `error`.** Truncated
input, an unparseable executable, an unknown magic — fill what you can and set
the verdict. A Go `error` is for "the file could not be read at all".

**`internal/analyze` is the only place measurement happens**, and it imports
nothing from `internal/tui` or `internal/report`. It never writes and never
opens a socket.

**The micro-block scan is resolution-independent.** `analyze` produces a fixed
set of blocks; renderers downsample. Nothing in `analyze` may depend on terminal
size.

**Dependency minimalism.** Standard library first. New third-party dependencies
need justification in the PR and must be pure Go.

**Do not claim it works without running it.** Applies to humans and agents alike.

## What a good pull request looks like

- **One thing.** A PR that adds a classifier *and* refactors the reporter is two
  PRs.
- **A title in conventional-commit form.**
- **A description that says why**, links the issue (`Closes #NN`), and names
  alternatives you rejected.
- **Tests that would have caught the bug.**
- **A `CHANGELOG.md` entry** under `[Unreleased]`.
- **Green CI** — test, race, cross-build, lint, vulnerability scan, Branch flow.

## Reporting bugs

Use the issue templates. For a rendering problem, say whether `--no-tui` shows it
too. For an analysis problem, attach the smallest file that reproduces it.

## Security

Do not open a public issue for a vulnerability. See [SECURITY.md](SECURITY.md).

## Code of conduct

By participating you agree to the [Code of Conduct](CODE_OF_CONDUCT.md).

## Licence

Contributions are accepted under the MIT Licence ([LICENSE](LICENSE)).
