param(
    [ValidatePattern('^[0-9]+\.[0-9]+\.[0-9]+(?:-[0-9A-Za-z.-]+)?(?:\+[0-9A-Za-z.-]+)?$')]
    [string]$Version = '0.1.0-dev',
    [ValidateSet('windows-amd64', 'linux-amd64', 'darwin-arm64')]
    [string[]]$Targets = @('windows-amd64', 'linux-amd64', 'darwin-arm64'),
    [string]$OutputDirectory = (Join-Path $PSScriptRoot '../dist/release')
)
$ErrorActionPreference = 'Stop'
$Targets = @($Targets | Select-Object -Unique)
$sourceDirectory = (Resolve-Path -LiteralPath (Join-Path $PSScriptRoot '..')).Path
$artifactDirectory = [IO.Path]::GetFullPath($OutputDirectory)
$null = [IO.Directory]::CreateDirectory($artifactDirectory)
$previousEnvironment = @{}
foreach ($name in @('GOOS', 'GOARCH', 'CGO_ENABLED')) {
    $previousEnvironment[$name] = [Environment]::GetEnvironmentVariable($name, 'Process')
}
$compiler = (& go version) -join "`n"
if ($LASTEXITCODE -ne 0) { throw 'Could not identify Go compiler' }
$linkFlags = "-s -w -X main.version=$Version"
$stagingDirectory = Join-Path $artifactDirectory ('.build-' + [Guid]::NewGuid().ToString('N'))
$null = [IO.Directory]::CreateDirectory($stagingDirectory)
$artifacts = [System.Collections.Generic.List[object]]::new()
Push-Location -LiteralPath $sourceDirectory
try {
    $env:CGO_ENABLED = '0'
    foreach ($target in $Targets) {
        $parts = $target.Split('-')
        $env:GOOS = $parts[0]
        $env:GOARCH = $parts[1]
        $name = "myenv-$target"
        if ($env:GOOS -eq 'windows') { $name += '.exe' }
        $path = Join-Path $stagingDirectory $name
        & go build -trimpath -buildvcs=false "-ldflags=$linkFlags" -o $path ./cmd/myenv
        if ($LASTEXITCODE -ne 0) { throw "Build failed for $target" }
        $artifacts.Add([pscustomobject]@{
            target = $target
            file = $name
            bytes = (Get-Item -LiteralPath $path).Length
            sha256 = (Get-FileHash -LiteralPath $path -Algorithm SHA256).Hash.ToLowerInvariant()
        })
    }
} catch {
    Write-Warning "Build did not complete; prior artifacts are unchanged. Partial files remain in $stagingDirectory"
    throw
} finally {
    Pop-Location
    foreach ($name in $previousEnvironment.Keys) {
        [Environment]::SetEnvironmentVariable($name, $previousEnvironment[$name], 'Process')
    }
}
$manifest = [ordered]@{
    version = $Version
    compiler = $compiler
    flags = @('-trimpath', '-buildvcs=false', "-ldflags=$linkFlags")
    cgo_enabled = '0'
    artifacts = $artifacts.ToArray()
}
$utf8 = [Text.UTF8Encoding]::new($false)
[IO.File]::WriteAllText((Join-Path $stagingDirectory 'build-manifest.json'),
    (($manifest | ConvertTo-Json -Depth 5) + "`n"), $utf8)
$checksums = ($artifacts | ForEach-Object { "$($_.sha256)  $($_.file)" }) -join "`n"
[IO.File]::WriteAllText((Join-Path $stagingDirectory 'SHA256SUMS'), $checksums + "`n", $utf8)
# Compilation and manifest creation have succeeded before replacing any output.
# Individual replacements are not a multi-file atomic transaction; readers must
# verify the manifest hashes after this script has completed successfully.
foreach ($artifact in $artifacts) {
    Move-Item -LiteralPath (Join-Path $stagingDirectory $artifact.file) -Destination (Join-Path $artifactDirectory $artifact.file) -Force
}
foreach ($name in @('SHA256SUMS', 'build-manifest.json')) {
    Move-Item -LiteralPath (Join-Path $stagingDirectory $name) -Destination (Join-Path $artifactDirectory $name) -Force
}
[IO.Directory]::Delete($stagingDirectory)
$artifacts | Format-Table target, bytes, sha256
