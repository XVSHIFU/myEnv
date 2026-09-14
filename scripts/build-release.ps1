param(
    [ValidatePattern('^[0-9]+\.[0-9]+\.[0-9]+(?:-[0-9A-Za-z.-]+)?(?:\+[0-9A-Za-z.-]+)?$')]
    [string]$Version = '0.1.0-dev',
    [ValidateSet('windows-amd64', 'linux-amd64')]
    [string[]]$Targets = @('windows-amd64', 'linux-amd64'),
    [switch]$GUI,
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
function Get-ReleaseSourceEvidence {
    # Include actual Go/frontend/build inputs, including generated embedded assets.
    # Skip dependency caches; versions and integrity records remain in the lockfiles.
    $inputs = [System.Collections.Generic.List[object]]::new()
    $pending = [System.Collections.Generic.Stack[string]]::new()
    foreach ($folder in @('cmd','internal','scripts')) { $pending.Push((Join-Path $sourceDirectory $folder)) }
    while ($pending.Count) {
        foreach ($item in Get-ChildItem -LiteralPath $pending.Pop() -Force) {
            if ($item.PSIsContainer) {
                if ($item.Name -notin @('node_modules','.git','.build','.myenv')) { $pending.Push($item.FullName) }
            } else {
                $relative = [IO.Path]::GetRelativePath($sourceDirectory, $item.FullName).Replace('\','/')
                $inputs.Add([pscustomobject]@{path=$relative;sha256=(Get-FileHash -LiteralPath $item.FullName -Algorithm SHA256).Hash.ToLowerInvariant()})
            }
        }
    }
    foreach ($name in @('go.mod','go.sum')) {
        $inputs.Add([pscustomobject]@{path=$name;sha256=(Get-FileHash -LiteralPath (Join-Path $sourceDirectory $name) -Algorithm SHA256).Hash.ToLowerInvariant()})
    }
    $files = @($inputs | Sort-Object path)
    $canonical = ($files | ForEach-Object { "$($_.sha256)  $($_.path)" }) -join "`n"
    $digest = [Convert]::ToHexString([Security.Cryptography.SHA256]::HashData([Text.Encoding]::UTF8.GetBytes($canonical))).ToLowerInvariant()
    $commit = $null; $dirty = $null
    if (Get-Command git -ErrorAction SilentlyContinue) {
        $head = & git -C $sourceDirectory rev-parse --verify HEAD 2>$null
        if ($LASTEXITCODE -eq 0) {
            $commit = [string]$head
            $status = & git -C $sourceDirectory status --porcelain --untracked-files=normal
            if ($LASTEXITCODE -ne 0) { throw 'Cannot determine release source status' }
            $dirty = [bool]$status
        }
    }
    return [ordered]@{base_commit=$commit;working_tree_modified=$dirty;input_sha256=$digest;files=$files}
}
$sourceEvidence = $null
Push-Location -LiteralPath $sourceDirectory
try {
    $env:CGO_ENABLED = '0'
    if ($GUI) {
        if (!$IsWindows -or $Targets -notcontains 'windows-amd64') { throw 'GUI packaging requires Windows and the Windows CLI target' }
        Push-Location cmd/myenv-gui/frontend
        try {
            & npm ci --ignore-scripts
            if ($LASTEXITCODE -ne 0) { throw 'GUI dependency install failed' }
            & npm run build
            if ($LASTEXITCODE -ne 0) { throw 'GUI frontend build failed' }
        } finally { Pop-Location }
        $sourceEvidence = Get-ReleaseSourceEvidence
        $env:GOOS='windows'; $env:GOARCH='amd64'
        $name='myenv-gui.exe'; $path=Join-Path $stagingDirectory $name
        & go build -trimpath -buildvcs=false -tags 'gui,desktop,production' "-ldflags=-H windowsgui $linkFlags" -o $path ./cmd/myenv-gui
        if ($LASTEXITCODE -ne 0) { throw 'GUI build failed' }
        & (Join-Path $PSScriptRoot 'set-gui-icon.ps1') -Executable $path
        $artifacts.Add([pscustomobject]@{target='windows-amd64-gui'; file=$name; bytes=(Get-Item $path).Length; sha256=(Get-FileHash $path -Algorithm SHA256).Hash.ToLowerInvariant()})
    }
    if (!$sourceEvidence) { $sourceEvidence = Get-ReleaseSourceEvidence }
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
    if ((Get-ReleaseSourceEvidence).input_sha256 -ne $sourceEvidence.input_sha256) { throw 'Release inputs changed during build; prior artifacts are unchanged' }
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
    source = $sourceEvidence
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
