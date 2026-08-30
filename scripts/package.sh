#!/bin/sh
# Package the cross-built Linux binaries into per-arch tar.gz archives and a
# single SHA256SUMS file. Invoked by `make dist` with VERSION, BINARY, DIST set.
set -eu

VERSION="${VERSION:?}"
BINARY="${BINARY:?}"
DIST="${DIST:?}"

ARCHES="amd64 386 arm64 arm"
MANGZ="$DIST/$BINARY.1.gz"

cd "$DIST"
: > SHA256SUMS

for arch in $ARCHES; do
	dir="${BINARY}_${VERSION}_linux_${arch}"
	if [ ! -x "$dir/$BINARY" ]; then
		echo "package.sh: missing $dir/$BINARY — run 'make build-linux' first" >&2
		exit 1
	fi

	# Assemble a staging tree so the archive carries docs alongside the binary.
	stage="$(mktemp -d)"
	mkdir -p "$stage/$dir"
	cp "$dir/$BINARY" "$stage/$dir/"
	cp ../LICENSE ../README.md ../CHANGELOG.md "$stage/$dir/" 2>/dev/null || true
	[ -f "$BINARY.1.gz" ] && cp "$BINARY.1.gz" "$stage/$dir/"

	tar -C "$stage" -czf "$dir.tar.gz" "$dir"
	rm -rf "$stage"

	sha256sum "$dir.tar.gz" >> SHA256SUMS
	echo "packaged $dir.tar.gz"
done

[ -f "$(basename "$MANGZ")" ] && sha256sum "$(basename "$MANGZ")" >> SHA256SUMS || true

echo
echo "SHA256SUMS:"
cat SHA256SUMS
