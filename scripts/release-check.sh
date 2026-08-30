#!/bin/sh
# Verify the working tree is ready to cut a release. Invoked by `make
# release-check` with VERSION set (read from the first CHANGELOG.md heading).
set -eu

VERSION="${VERSION:?}"
fail=0

say() { printf '%s %s\n' "$1" "$2"; }

# 1. Clean working tree.
if [ -n "$(git status --porcelain 2>/dev/null)" ]; then
	say "FAIL" "working tree is dirty — commit or stash first"
	fail=1
else
	say "OK  " "working tree is clean"
fi

# 2. CHANGELOG has a section for this version (not still 'Unreleased' only).
if grep -q "^## \[${VERSION}\]" CHANGELOG.md 2>/dev/null; then
	say "OK  " "CHANGELOG.md has a [$VERSION] section"
else
	say "FAIL" "CHANGELOG.md has no '## [$VERSION]' heading"
	fail=1
fi

# 3. Tag is free.
if git rev-parse -q --verify "refs/tags/v${VERSION}" >/dev/null; then
	say "FAIL" "tag v$VERSION already exists"
	fail=1
else
	say "OK  " "tag v$VERSION is free"
fi

# 4. No blocking markers left in tracked Go source.
if git grep -nE 'TODO\(release\)|XXX\(release\)' -- '*.go' >/dev/null 2>&1; then
	say "FAIL" "release-blocking TODO/XXX markers present:"
	git grep -nE 'TODO\(release\)|XXX\(release\)' -- '*.go' | sed 's/^/       /'
	fail=1
else
	say "OK  " "no release-blocking markers"
fi

# 5. The module still cross-compiles to every release target.
for pair in linux/amd64 linux/386 linux/arm64 linux/arm; do
	os="${pair%/*}"
	arch="${pair#*/}"
	if CGO_ENABLED=0 GOOS="$os" GOARCH="$arch" go build -o /dev/null ./... 2>/dev/null; then
		say "OK  " "cross-build $pair"
	else
		say "FAIL" "cross-build $pair"
		fail=1
	fi
done

echo
if [ "$fail" -ne 0 ]; then
	echo "release-check: NOT ready"
	exit 1
fi
echo "release-check: ready to tag v$VERSION"
