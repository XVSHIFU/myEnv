param(
    [string]$ReleaseDirectory = (Join-Path $PSScriptRoot '../dist/release'),
    [string]$OutputDirectory = (Join-Path $PSScriptRoot '../dist/packages')
)
$ErrorActionPreference = 'Stop'
$release = (Resolve-Path -LiteralPath $ReleaseDirectory).Path
$manifest = Get-Content -LiteralPath (Join-Path $release 'build-manifest.json') -Raw | ConvertFrom-Json
$license = Join-Path $PSScriptRoot '../LICENSE'
$licenseEntry = @($manifest.source.files | Where-Object path -eq 'LICENSE')
if ($licenseEntry.Count -ne 1 -or !(Test-Path -LiteralPath $license -PathType Leaf)) { throw 'Build source evidence must include the project LICENSE; rebuild the release first' }
if ((Get-FileHash -LiteralPath $license -Algorithm SHA256).Hash.ToLowerInvariant() -ne $licenseEntry[0].sha256) { throw 'Project LICENSE changed since the release build; rebuild before packaging' }
$entry = @($manifest.artifacts | Where-Object target -eq 'windows-amd64')
if ($entry.Count -ne 1) { throw 'Expected one Windows amd64 artifact' }
$binary = Join-Path $release 'myenv-windows-amd64.exe'
if ((Get-FileHash -LiteralPath $binary -Algorithm SHA256).Hash.ToLowerInvariant() -ne $entry[0].sha256) { throw 'Release checksum mismatch' }
$gui = @($manifest.artifacts | Where-Object target -eq 'windows-amd64-gui')
$noticeDirectory = Join-Path $PSScriptRoot '../cmd/myenv-gui/build'
& (Join-Path $PSScriptRoot 'generate-third-party-notices.ps1') -OutputDirectory $noticeDirectory -CLIOnly:($gui.Count -eq 0)
$notice = Join-Path $noticeDirectory 'THIRD_PARTY_NOTICES.txt'
$noticeManifest = Join-Path $noticeDirectory 'third-party-manifest.json'
$output = [IO.Path]::GetFullPath($OutputDirectory)
$null = [IO.Directory]::CreateDirectory($output)
$stage = Join-Path $output ('stage-' + [Guid]::NewGuid().ToString('N'))
$null = [IO.Directory]::CreateDirectory($stage)
$portable = Join-Path $stage 'myenv'
$null = [IO.Directory]::CreateDirectory($portable)
Copy-Item -LiteralPath $binary -Destination (Join-Path $portable 'myenv.exe')
if ($gui.Count -gt 0) {
    if ($gui.Count -ne 1) { throw 'Expected one GUI artifact' }
    $guiBinary = Join-Path $release $gui[0].file
    if ((Get-FileHash $guiBinary -Algorithm SHA256).Hash.ToLowerInvariant() -ne $gui[0].sha256) { throw 'GUI checksum mismatch' }
    Copy-Item -LiteralPath $guiBinary -Destination (Join-Path $portable 'myenv-gui.exe')
}
Copy-Item -LiteralPath $notice -Destination (Join-Path $portable 'THIRD_PARTY_NOTICES.txt')
Copy-Item -LiteralPath $noticeManifest -Destination (Join-Path $portable 'third-party-manifest.json')
Copy-Item -LiteralPath $license -Destination (Join-Path $portable 'LICENSE')
Copy-Item -LiteralPath $license -Destination (Join-Path $output 'LICENSE')
Copy-Item -LiteralPath $notice -Destination (Join-Path $output 'THIRD_PARTY_NOTICES.txt')
Copy-Item -LiteralPath $noticeManifest -Destination (Join-Path $output 'third-party-manifest.json')
$readme = [IO.File]::ReadAllText((Join-Path $PSScriptRoot '../docs/windows-install.md'))
$readme = $readme.Replace('](../LICENSE)', '](LICENSE)').Replace('](../website/public/images/', '](https://xvshifu.github.io/myEnv/images/')
$readme = $readme.Replace('](build.md)', '](https://github.com/XVSHIFU/myEnv/blob/main/docs/build.md)').Replace('](implementation.md)', '](https://github.com/XVSHIFU/myEnv/blob/main/docs/implementation.md)')
[IO.File]::WriteAllText((Join-Path $portable 'README.md'), $readme, [Text.UTF8Encoding]::new($false))
[IO.File]::WriteAllText((Join-Path $stage 'version.txt'), $manifest.version, [Text.UTF8Encoding]::new($false))
$compiler = Join-Path $env:WINDIR 'Microsoft.NET/Framework64/v4.0.30319/csc.exe'
if (!(Test-Path -LiteralPath $compiler)) { throw 'Windows .NET Framework compiler is required to package the wizard' }
$setup = Join-Path $output "myenv-$($manifest.version)-windows-amd64-setup.exe"
& $compiler /nologo /target:winexe /platform:x64 /optimize+ "/out:$setup" /reference:System.Windows.Forms.dll /reference:System.Drawing.dll "/win32manifest:$(Join-Path $PSScriptRoot 'windows/Setup.manifest')" "/resource:$binary,myenv.exe" "/resource:$(Join-Path $stage 'version.txt'),version.txt" "/resource:$license,LICENSE" "/resource:$notice,THIRD_PARTY_NOTICES.txt" (Join-Path $PSScriptRoot 'windows/Setup.cs')
if ($LASTEXITCODE -ne 0) { throw 'Installer compilation failed' }
$zip = Join-Path $output "myenv-$($manifest.version)-windows-amd64-portable.zip"
Compress-Archive -LiteralPath $portable -DestinationPath $zip -Force
$hashes = @($setup, $zip, (Join-Path $output 'LICENSE'), (Join-Path $output 'THIRD_PARTY_NOTICES.txt'), (Join-Path $output 'third-party-manifest.json')) | ForEach-Object {
    '{0}  {1}' -f (Get-FileHash -LiteralPath $_ -Algorithm SHA256).Hash.ToLowerInvariant(), [IO.Path]::GetFileName($_)
}
[IO.File]::WriteAllLines((Join-Path $output 'SHA256SUMS'), $hashes, [Text.UTF8Encoding]::new($false))
Get-Item -LiteralPath $setup, $zip | Select-Object Name, Length
