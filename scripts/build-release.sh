#!/usr/bin/env bash
set -euo pipefail

APP_NAME="${APP_NAME:-lazyfirewall}"
CMD_PATH="${CMD_PATH:-./cmd/lazyfirewall}"
OUT_DIR="${OUT_DIR:-dist}"
ARCHES="${ARCHES:-amd64 arm64 arm 386}"

VERSION="${VERSION:-dev}"
COMMIT="${COMMIT:-$(git rev-parse --short HEAD 2>/dev/null || echo none)}"
DATE="${DATE:-$(date -u +%Y-%m-%dT%H:%M:%SZ)}"

LDFLAGS="-s -w -X lazyfirewall/internal/version.Version=${VERSION} -X lazyfirewall/internal/version.Commit=${COMMIT} -X lazyfirewall/internal/version.Date=${DATE}"

rm -rf "${OUT_DIR}"
mkdir -p "${OUT_DIR}"

for arch in ${ARCHES}; do
  package="${APP_NAME}_${VERSION}_linux_${arch}"
  package_dir="${OUT_DIR}/${package}"

  mkdir -p "${package_dir}"
  CGO_ENABLED=0 GOOS=linux GOARCH="${arch}" go build \
    -trimpath \
    -ldflags "${LDFLAGS}" \
    -o "${package_dir}/${APP_NAME}" \
    "${CMD_PATH}"

  tar -C "${OUT_DIR}" -czf "${OUT_DIR}/${package}.tar.gz" "${package}"
  rm -rf "${package_dir}"
done

(
  cd "${OUT_DIR}"
  sha256sum ./*.tar.gz > checksums.sha256
  sha512sum ./*.tar.gz > checksums.sha512
)

echo "Release artifacts created in ${OUT_DIR}/"
