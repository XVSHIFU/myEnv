#!/usr/bin/env bash
# Run on native Linux x86_64/glibc or macOS arm64 from any working directory.
set -euo pipefail
cd "$(dirname "$0")/.."
artifact_dir="${1:-$PWD/.build/node-real}"
mkdir -p "$artifact_dir"
artifact_dir="$(cd "$artifact_dir" && pwd)"
export MYENV_TEST_ARTIFACTS="$artifact_dir"
export MYENV_TEST_PREPARED_RECORD="$artifact_dir/prepared.json"
if [[ ! -f "$MYENV_TEST_PREPARED_RECORD" ]]; then
  MYENV_TEST_PREPARE=1 go test ./internal/backend -run '^TestNodeOfficialPrepare$' -count=1 -v
fi
go test ./internal/backend ./internal/runner ./internal/core ./internal/cli \
  -run 'TestExtractTarGZ|TestPlatformIdentification|TestVerifyRetainedNode|TestSyncRetainedNode|TestRunRetainedNode|TestRollbackRetainedNode|TestCancelRetainedNode|TestCancelProcessGroup' -count=1 -v
