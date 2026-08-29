#!/usr/bin/env bash
set -euo pipefail

mkdir -p build

VERSION=$(cat VERSION)

targets=(
  "linux amd64 odota_cli_linux_amd64"
  "linux arm64 odota_cli_linux_arm64"
  "windows amd64 odota_cli_win_amd64.exe"
  "windows arm64 odota_cli_win_arm64.exe"
  "darwin amd64 odota_cli_macos_amd64"
  "darwin arm64 odota_cli_macos_arm64"
)

for t in "${targets[@]}"; do
  read -r goos goarch out <<< "$t"
  GOOS="$goos" GOARCH="$goarch" go build -trimpath -ldflags="-s -w -X main.version=$VERSION" -o "build/$out" .
  echo "✓ build/$out"
done

echo "All builds complete (v$VERSION)."