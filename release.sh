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

# darwin builds lack GPU, fan, PSS and CPU temp (those read /sys, /proc or need CGO)
for target in linux/amd64 linux/arm64 darwin/amd64 darwin/arm64; do
  os="${target%/*}" arch="${target#*/}"
  echo "→ Building $target…"
  GOOS="$os" GOARCH="$arch" CGO_ENABLED=0 go build \
    -ldflags="-s -w -X main.version=${VERSION}" \
    -o "$DIST/cscom_${os}_${arch}" ./cmd/cscom/
done

( cd "$DIST" && sha256sum cscom_* > checksums.txt )

echo
echo "✓ Artifacts in $DIST:"
ls -lh "$DIST" | tail -n +2 | awk '{printf "    %-24s %s\n", $9, $5}'
echo
echo "Next steps:"
echo "  1. git tag -a $VERSION -m \"$VERSION\" && git push origin $VERSION"
echo "  2. open https://github.com/ogak-github/cscom_go/releases/new?tag=$VERSION"
echo "  3. drag the files above into the release, then publish"
