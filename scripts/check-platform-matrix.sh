#!/usr/bin/env sh
set -eu

mkdir -p dist/check

for target in \
  linux/amd64 \
  linux/arm64 \
  darwin/amd64 \
  darwin/arm64 \
  windows/amd64 \
  windows/arm64
do
  GOOS_VALUE="${target%/*}"
  GOARCH_VALUE="${target#*/}"
  EXT=""
  if [ "$GOOS_VALUE" = "windows" ]; then
    EXT=".exe"
  fi
  echo "building ${GOOS_VALUE}/${GOARCH_VALUE}"
  GOOS="$GOOS_VALUE" GOARCH="$GOARCH_VALUE" CGO_ENABLED=0 go build -o "dist/check/dbbackup233-${GOOS_VALUE}-${GOARCH_VALUE}${EXT}" .
done

echo "platform matrix build passed"
