<!--
Thanks for contributing to susfile.

Read CONTRIBUTING.md. PROJECT.md is the authoritative spec — if this change
contradicts it, say so explicitly below. Changing the spec is allowed; doing it
by accident is not.
-->

## What this changes

<!-- One or two sentences. What is different after this is merged? -->

## Why

<!-- The reasoning, not the diff. What was missing or broken, and which
alternatives you rejected. -->

Closes #

## How to verify

```bash

```

<!-- The commands a reviewer runs, and what they should see. -->

## Definition of done

- [ ] Implementation is complete — no reachable stubs
- [ ] Tests exist where appropriate
- [ ] `make fmt vet test build` passes
- [ ] `CHANGELOG.md` updated under `[Unreleased]`
- [ ] User-facing behaviour documented in `docs/`
- [ ] A malformed file yields a populated `Result`, not a panic
- [ ] Commits are small, cohesive and conventionally prefixed

## Checks that apply

<!-- Tick what is relevant; delete the rest. -->

- [ ] **Cross-compilation** still works — `make build-linux`, or at least
      `CGO_ENABLED=0 GOOS=linux GOARCH=arm GOARM=7 go build ./...`
- [ ] **No cgo dependency** added
- [ ] **No network code** added — susfile is offline by design
- [ ] **No writes** to the analysed file
- [ ] **No committed binary fixtures** — inputs are `[]byte` literals or built in-test
- [ ] **Measurement stays in `internal/analyze`** — renderers don't compute
- [ ] **New dependency** justified here, and pure Go
- [ ] **Branch** is `feature/…` / `fix/…` off `develop` (or `hotfix/…` off `main`)

## Anything a reviewer should know

<!-- Known gaps, follow-up work, decisions you want a second opinion on. -->
