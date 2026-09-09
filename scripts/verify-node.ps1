param([string]$ArtifactDirectory = '')
$ErrorActionPreference = 'Stop'
Push-Location (Split-Path $PSScriptRoot -Parent)
try {
    if ($ArtifactDirectory -eq '') { $ArtifactDirectory = Join-Path $PWD '.build/node-real' }
    $ArtifactDirectory = [IO.Path]::GetFullPath($ArtifactDirectory)
    New-Item -ItemType Directory -Force -Path $ArtifactDirectory | Out-Null
    $env:MYENV_TEST_ARTIFACTS = $ArtifactDirectory
    $env:MYENV_TEST_PREPARED_RECORD = Join-Path $ArtifactDirectory 'prepared.json'
    if (-not (Test-Path -LiteralPath $env:MYENV_TEST_PREPARED_RECORD)) {
        $env:MYENV_TEST_PREPARE = '1'
        go test ./internal/backend -run '^TestNodeOfficialPrepare$' -count=1 -v
        if ($LASTEXITCODE -ne 0) { throw 'Official Node preparation failed' }
        $env:MYENV_TEST_PREPARE = '0'
    }
    go test ./internal/backend ./internal/runner ./internal/core ./internal/cli -run 'TestExtractTarGZ|TestPlatformIdentification|TestVerifyRetainedNode|TestSyncRetainedNode|TestRunRetainedNode|TestRollbackRetainedNode|TestCancelRetainedNode|TestWindowsJobRetainedNode|TestWindowsJobSupervisorCrash' -count=1 -v
    if ($LASTEXITCODE -ne 0) { throw 'Node verification failed' }
} finally {
    Pop-Location
}
