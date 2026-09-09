param(
    [string]$ReleaseDirectory = (Join-Path $PSScriptRoot '../dist/release'),
    [string]$OutputDirectory = (Join-Path $PSScriptRoot '../dist/packages')
)
$ErrorActionPreference = 'Stop'
$release = (Resolve-Path -LiteralPath $ReleaseDirectory).Path
$manifest = Get-Content -LiteralPath (Join-Path $release 'build-manifest.json') -Raw | ConvertFrom-Json
$entry = @($manifest.artifacts | Where-Object target -eq 'windows-amd64')
if ($entry.Count -ne 1) { throw 'Expected one Windows amd64 artifact' }
$binary = Join-Path $release 'myenv-windows-amd64.exe'
if ((Get-FileHash -LiteralPath $binary -Algorithm SHA256).Hash.ToLowerInvariant() -ne $entry[0].sha256) { throw 'Release checksum mismatch' }
$output = [IO.Path]::GetFullPath($OutputDirectory)
$null = [IO.Directory]::CreateDirectory($output)
$stage = Join-Path $output ('stage-' + [Guid]::NewGuid().ToString('N'))
$null = [IO.Directory]::CreateDirectory($stage)
$portable = Join-Path $stage 'myenv'
$null = [IO.Directory]::CreateDirectory($portable)
Copy-Item -LiteralPath $binary -Destination (Join-Path $portable 'myenv.exe')
Copy-Item -LiteralPath (Join-Path $PSScriptRoot '../docs/windows-install.md') -Destination (Join-Path $portable 'README.md')
[IO.File]::WriteAllText((Join-Path $stage 'version.txt'), $manifest.version, [Text.UTF8Encoding]::new($false))
$compiler = Join-Path $env:WINDIR 'Microsoft.NET/Framework64/v4.0.30319/csc.exe'
if (!(Test-Path -LiteralPath $compiler)) { throw 'Windows .NET Framework compiler is required to package the wizard' }
$setup = Join-Path $output "myenv-$($manifest.version)-windows-amd64-setup.exe"
& $compiler /nologo /target:winexe /platform:x64 /optimize+ "/out:$setup" /reference:System.Windows.Forms.dll /reference:System.Drawing.dll "/win32manifest:$(Join-Path $PSScriptRoot 'windows/Setup.manifest')" "/resource:$binary,myenv.exe" "/resource:$(Join-Path $stage 'version.txt'),version.txt" (Join-Path $PSScriptRoot 'windows/Setup.cs')
if ($LASTEXITCODE -ne 0) { throw 'Installer compilation failed' }
$zip = Join-Path $output "myenv-$($manifest.version)-windows-amd64-portable.zip"
Compress-Archive -LiteralPath $portable -DestinationPath $zip -Force
$hashes = @($setup, $zip) | ForEach-Object {
    '{0}  {1}' -f (Get-FileHash -LiteralPath $_ -Algorithm SHA256).Hash.ToLowerInvariant(), [IO.Path]::GetFileName($_)
}
[IO.File]::WriteAllLines((Join-Path $output 'SHA256SUMS'), $hashes, [Text.UTF8Encoding]::new($false))
Get-Item -LiteralPath $setup, $zip | Select-Object Name, Length
