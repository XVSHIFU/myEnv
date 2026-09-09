#!/usr/bin/env bash
# Native macOS prompt verification only; this is not the full release gate.
set -euo pipefail
if [[ "$(uname -s)" != Darwin || "$(uname -m)" != arm64 ]]; then
  echo 'Requires a native macOS arm64 host.' >&2
  exit 2
fi
command -v go >/dev/null || { echo 'Go is required.' >&2; exit 2; }
command -v python3 >/dev/null || { echo 'python3 is required for the private PTY fixture.' >&2; exit 2; }
project_root="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)"
cd -- "$project_root"
mkdir -p .build
result_dir="$(mktemp -d "$project_root/.build/darwin-prompt-native.XXXXXX")"
mkdir -p "$result_dir/tmp"
export GOPATH="$project_root/.build/gopath"
export GOCACHE="$project_root/.build/gocache"
export TMPDIR="$result_dir/tmp"
export GOOS=darwin GOARCH=arm64 CGO_ENABLED=0
{
  uname -a
  go version
  python3 --version
} > "$result_dir/host.txt"
printf 'Evidence directory: %s\n' "$result_dir"
go test -c -o "$result_dir/cli.test" ./internal/cli 2>&1 | tee "$result_dir/build.log"
shasum -a 256 "$result_dir/cli.test" > "$result_dir/SHA256SUMS"
"$result_dir/cli.test" -test.run='^TestDarwinPromptCancellationAndInput$' -test.count=1 -test.v -test.timeout=30s 2>&1 | tee "$result_dir/test.log"
if grep -q -- '--- SKIP:' "$result_dir/test.log"; then
  echo 'The native prompt test skipped; verification is incomplete.' >&2
  exit 1
fi
if ! grep -q -- '^--- PASS: TestDarwinPromptCancellationAndInput ' "$result_dir/test.log"; then
  echo 'The required native prompt test did not report a pass.' >&2
  exit 1
fi
printf 'Native prompt test passed; full runtime and release validation remain separate.\n'
