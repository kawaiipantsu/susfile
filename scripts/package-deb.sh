#!/bin/sh
# Build one .deb per Linux arch from the cross-built binaries, using dpkg-deb
# directly (no fpm, no ruby). Invoked by `make deb` with VERSION, BINARY, DIST set.
set -eu

VERSION="${VERSION:?}"
BINARY="${BINARY:?}"
DIST="${DIST:?}"

command -v dpkg-deb >/dev/null 2>&1 || {
	echo "package-deb.sh: dpkg-deb not found (install dpkg-dev)" >&2
	exit 1
}

# GOARCH -> Debian architecture
deb_arch() {
	case "$1" in
	amd64) echo amd64 ;;
	386) echo i386 ;;
	arm64) echo arm64 ;;
	arm) echo armhf ;;
	*)
		echo "unknown arch $1" >&2
		exit 1
		;;
	esac
}

# Debian revision-free version: strip a leading 'v', keep the rest.
DEBVER="${VERSION#v}"

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
CONTROL_IN="$ROOT/packaging/debian/control.in"
COPYRIGHT="$ROOT/packaging/debian/copyright"
CHANGELOG_IN="$ROOT/packaging/debian/changelog.in"
MANGZ="$DIST/$BINARY.1.gz"

for arch in amd64 386 arm64 arm; do
	da="$(deb_arch "$arch")"
	src="$DIST/${BINARY}_${VERSION}_linux_${arch}/$BINARY"
	[ -x "$src" ] || {
		echo "package-deb.sh: missing $src — run 'make build-linux' first" >&2
		exit 1
	}

	pkgdir="$(mktemp -d)"
	chmod 0755 "$pkgdir"
	mkdir -p "$pkgdir/DEBIAN" \
		"$pkgdir/usr/bin" \
		"$pkgdir/usr/share/doc/$BINARY" \
		"$pkgdir/usr/share/man/man1"

	install -m 0755 "$src" "$pkgdir/usr/bin/$BINARY"

	sed -e "s/@VERSION@/$DEBVER/g" -e "s/@ARCH@/$da/g" "$CONTROL_IN" \
		> "$pkgdir/DEBIAN/control"

	install -m 0644 "$COPYRIGHT" "$pkgdir/usr/share/doc/$BINARY/copyright"
	sed -e "s/@VERSION@/$DEBVER/g" -e "s/@DATE@/$(date -u -R)/g" "$CHANGELOG_IN" \
		| gzip -9 -n > "$pkgdir/usr/share/doc/$BINARY/changelog.Debian.gz"
	[ -f "$MANGZ" ] && install -m 0644 "$MANGZ" \
		"$pkgdir/usr/share/man/man1/$BINARY.1.gz"

	out="$DIST/${BINARY}_${DEBVER}_${da}.deb"
	dpkg-deb --root-owner-group --build "$pkgdir" "$out" >/dev/null
	rm -rf "$pkgdir"

	echo "built $out"
	dpkg-deb -I "$out" | sed 's/^/    /'
done

# Fold the .deb checksums into SHA256SUMS if dist already made one.
if [ -f "$DIST/SHA256SUMS" ]; then
	(cd "$DIST" && sha256sum ./*.deb >> SHA256SUMS)
fi
