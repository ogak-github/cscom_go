#!/usr/bin/env bash
# Build the TUI binary for release and print the upload instructions.
# Usage: ./release.sh v0.1.0
set -euo pipefail

VERSION="${1:-}"
if [[ -z "$VERSION" ]]; then
  echo "Usage: $0 <version>   e.g. $0 v0.1.0" >&2
  exit 1
fi
[[ "$VERSION" == v* ]] || { echo "Error: version must start with 'v'" >&2; exit 1; }
[[ -f go.mod ]] || { echo "Error: run from project root" >&2; exit 1; }

DIST="dist/$VERSION"
rm -rf "$DIST" && mkdir -p "$DIST"

# metrics read /sys/class/{hwmon,drm}, so linux-only targets
for arch in amd64 arm64; do
  echo "→ Building linux/$arch…"
  GOOS=linux GOARCH="$arch" CGO_ENABLED=0 go build \
    -ldflags="-s -w -X main.version=${VERSION}" \
    -o "$DIST/cscom_linux_${arch}" ./cmd/cscom/
done

( cd "$DIST" && sha256sum cscom_linux_* > checksums.txt )

echo
echo "✓ Artifacts in $DIST:"
ls -lh "$DIST" | tail -n +2 | awk '{printf "    %-24s %s\n", $9, $5}'
echo
echo "Next steps:"
echo "  1. git tag -a $VERSION -m \"$VERSION\" && git push origin $VERSION"
echo "  2. open https://github.com/ogak-github/cscom_go/releases/new?tag=$VERSION"
echo "  3. drag the files above into the release, then publish"
