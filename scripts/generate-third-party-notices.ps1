param(
    [string]$OutputDirectory = (Join-Path $PSScriptRoot '../cmd/myenv-gui/build'),
    [switch]$CLIOnly
)
$ErrorActionPreference = 'Stop'
$sourceRoot = (Resolve-Path -LiteralPath (Join-Path $PSScriptRoot '..')).Path
$outputRoot = [IO.Path]::GetFullPath($OutputDirectory)
$utf8 = [Text.UTF8Encoding]::new($false)
$records = [Collections.Generic.List[object]]::new()
$seenFiles = [Collections.Generic.HashSet[string]]::new([StringComparer]::OrdinalIgnoreCase)
$seenDirectories = [Collections.Generic.HashSet[string]]::new([StringComparer]::OrdinalIgnoreCase)
$modules = @{}
$noticePattern = '^(LICENSE|LICENCE|COPYING|COPYRIGHT|NOTICE|PATENTS)([._-].*)?$|^THIRD[-_]PARTY[-_]LICENSE.*$'

function Add-Notice([string]$Path, [string]$Component, [string]$Version, [string]$Source, [string]$Scope) {
    $resolved = (Resolve-Path -LiteralPath $Path).Path
    $info = Get-Item -LiteralPath $resolved
    if ($info.PSIsContainer -or $info.Length -gt 2MB) { throw "Invalid or oversized upstream notice: $resolved" }
    $identity = $Component + '|' + $Source
    if (!$seenFiles.Add($identity)) { return }
    $text = [IO.File]::ReadAllText($resolved)
    if ([string]::IsNullOrWhiteSpace($text)) { throw "Empty upstream notice: $resolved" }
    $records.Add([pscustomobject][ordered]@{
        component=$Component; version=$Version; source=$Source; scope=$Scope
        sha256=(Get-FileHash -LiteralPath $resolved -Algorithm SHA256).Hash.ToLowerInvariant()
        text=$text
    })
}

function Add-DirectoryNotices([string]$Directory, [string]$Root, [string]$Component, [string]$Version, [string]$Scope) {
    if (!$seenDirectories.Add($Component+'|'+$Directory)) { return }
    foreach ($file in @(Get-ChildItem -LiteralPath $Directory -File | Where-Object Name -match $noticePattern | Sort-Object Name)) {
        $relative = [IO.Path]::GetRelativePath($Root, $file.FullName).Replace('\','/')
        Add-Notice $file.FullName $Component $Version $relative $Scope
    }
}

$savedEnvironment = @{}
foreach ($name in @('GOOS','GOARCH','CGO_ENABLED','GOPROXY','GOSUMDB','GOTOOLCHAIN')) {
    $savedEnvironment[$name] = [Environment]::GetEnvironmentVariable($name,'Process')
}
Push-Location -LiteralPath $sourceRoot
try {
    # All license material must already be installed. This command never
    # provisions a Go toolchain or downloads dependency source to fill a gap.
    $env:GOPROXY='off'; $env:GOSUMDB='off'; $env:GOTOOLCHAIN='local'
    $env:GOARCH='amd64'; $env:CGO_ENABLED='0'
    $goRoot = (& go env GOROOT).Trim()
    if ($LASTEXITCODE -ne 0) { throw 'Cannot identify installed Go source root' }
    $goVersion = (& go env GOVERSION).Trim()
    Add-DirectoryNotices $goRoot $goRoot 'Go runtime and standard library' $goVersion 'Windows CLI/GUI and Linux CLI/TUI'
    $targets = @(
        @{Name='windows-amd64-cli'; GOOS='windows'; Main='./cmd/myenv'; Tags=''},
        @{Name='linux-amd64-cli'; GOOS='linux'; Main='./cmd/myenv'; Tags=''}
    )
    if (!$CLIOnly) { $targets += @{Name='windows-amd64-gui'; GOOS='windows'; Main='./cmd/myenv-gui'; Tags='gui,desktop,production'} }
    foreach ($target in $targets) {
        $env:GOOS=$target.GOOS
        $template='{{if .Module}}{{.Module.Path}}|{{.Module.Version}}|{{.Module.Dir}}|{{.Dir}}{{else}}{{if .Standard}}std|' + $goVersion + '|' + $goRoot + '|{{.Dir}}{{end}}{{end}}'
        $arguments = @('list','-deps','-f',$template)
        if ($target.Tags) { $arguments += @('-tags',$target.Tags) }
        $arguments += $target.Main
        $rows = @(& go @arguments)
        if ($LASTEXITCODE -ne 0) { throw "Cannot enumerate cached production dependencies for $($target.Name)" }
        foreach ($row in $rows) {
            if (!$row -or $row.StartsWith('myenv|')) { continue }
            $parts=$row.Split('|')
            if ($parts.Count -ne 4) { throw "Unexpected Go dependency row: $row" }
            $component,$version,$moduleRoot,$packageDirectory=$parts
            if (!$moduleRoot -or !(Test-Path -LiteralPath $moduleRoot -PathType Container)) { throw "Missing cached source for $component $version" }
            if (!$modules.ContainsKey($component)) {
                $modules[$component]=[ordered]@{component=$component; version=$version; targets=[Collections.Generic.HashSet[string]]::new(); notices=[Collections.Generic.HashSet[string]]::new()}
            }
            $null=$modules[$component].targets.Add($target.Name)
            $directory=$packageDirectory
            while ($true) {
                $relative=[IO.Path]::GetRelativePath($moduleRoot,$directory)
                if ($relative -eq '..' -or $relative.StartsWith('..'+[IO.Path]::DirectorySeparatorChar)) { throw "Dependency source escaped module root: $directory" }
                Add-DirectoryNotices $directory $moduleRoot $component $version 'Production Go dependency closure'
                if ($relative -eq '.') { break }
                $directory=[IO.Path]::GetDirectoryName($directory)
            }
        }
    }
    foreach ($component in @($modules.Keys | Sort-Object)) {
        if ($component -eq 'std') { continue }
        if (@($records | Where-Object component -eq $component).Count -eq 0) { throw "No actual upstream license found for $component; cannot silently omit its notice" }
    }
    if (!$CLIOnly) {
        $frontend=Join-Path $sourceRoot 'cmd/myenv-gui/frontend'
        $package=Get-Content -LiteralPath (Join-Path $frontend 'package.json') -Raw | ConvertFrom-Json
        $lock=Get-Content -LiteralPath (Join-Path $frontend 'package-lock.json') -Raw | ConvertFrom-Json -AsHashtable
        $pending=[Collections.Generic.Queue[string]]::new()
        foreach ($property in $package.dependencies.PSObject.Properties) { $pending.Enqueue($property.Name) }
        # Vite's module-preload polyfill and Rolldown's interop helpers appear
        # in the shipped bundle. Their upstream additional notices are retained.
        $pending.Enqueue('vite'); $pending.Enqueue('rolldown')
        $seenPackages=[Collections.Generic.HashSet[string]]::new()
        while ($pending.Count -gt 0) {
            $name=$pending.Dequeue()
            if (!$seenPackages.Add($name)) { continue }
            $packageRoot=Join-Path $frontend ('node_modules/'+$name)
            $metadata=Get-Content -LiteralPath (Join-Path $packageRoot 'package.json') -Raw | ConvertFrom-Json
            $locked=$lock.packages['node_modules/'+$name]
            if (!$locked -or $locked.version -ne $metadata.version) { throw "Installed frontend package does not match lock: $name" }
            $before=$records.Count
            Add-DirectoryNotices $packageRoot $packageRoot ('npm:'+ $name) $metadata.version 'Bundled GUI JavaScript or generated runtime helper'
            if ($records.Count -eq $before) { throw "No installed upstream license found for frontend package $name" }
            if ($name -notin @('vite','rolldown')) {
                foreach ($property in $metadata.dependencies.PSObject.Properties) { $pending.Enqueue($property.Name) }
            }
        }
        Add-Notice (Join-Path $sourceRoot 'cmd/myenv-gui/THIRD_PARTY_NOTICES.txt') 'GUI SVG artwork (Devicon and Lucide)' 'Pinned source commits recorded below' 'cmd/myenv-gui/THIRD_PARTY_NOTICES.txt' 'Bundled GUI icons; original notice retained verbatim'
    }
} finally {
    Pop-Location
    foreach ($name in $savedEnvironment.Keys) { [Environment]::SetEnvironmentVariable($name,$savedEnvironment[$name],'Process') }
}

$builder=[Text.StringBuilder]::new()
$null=$builder.AppendLine('myEnv — third-party copyright and license notices')
$null=$builder.AppendLine()
$null=$builder.AppendLine('This file reproduces notices from installed upstream source used by the selected production builds.')
$null=$builder.AppendLine('Each component remains under its own license. The original myEnv project is distributed under MIT; see the accompanying LICENSE file.')
$null=$builder.AppendLine('The Windows GUI uses the system-installed WebView2 Runtime; that runtime is not redistributed in these packages.')
$null=$builder.AppendLine('Additional notices distributed by an upstream component are retained even when they cover code outside the selected build.')
$null=$builder.AppendLine('Generated offline by scripts/generate-third-party-notices.ps1; source file hashes are listed in third-party-manifest.json.')
foreach ($record in @($records | Sort-Object component,source)) {
    $null=$builder.AppendLine(); $null=$builder.AppendLine(('=' * 78))
    $null=$builder.AppendLine($record.component+' '+$record.version)
    $null=$builder.AppendLine('Upstream notice: '+$record.source)
    $null=$builder.AppendLine('Scope: '+$record.scope)
    $null=$builder.AppendLine(); $null=$builder.Append($record.text)
    if (!$record.text.EndsWith("`n")) { $null=$builder.AppendLine() }
}
$null=[IO.Directory]::CreateDirectory($outputRoot)
$noticePath=Join-Path $outputRoot 'THIRD_PARTY_NOTICES.txt'
[IO.File]::WriteAllText($noticePath,$builder.ToString(),$utf8)
$manifest=[ordered]@{
    source='Installed npm packages and Go production dependency closure, no network retrieval'
    project_license='MIT; see the accompanying LICENSE file'
    project_license_sha256=(Get-FileHash -LiteralPath (Join-Path $sourceRoot 'LICENSE') -Algorithm SHA256).Hash.ToLowerInvariant()
    cli_only=[bool]$CLIOnly
    targets=@($targets | ForEach-Object { $_.Name })
    modules=@($modules.Keys | Where-Object { $_ -ne 'std' } | Sort-Object | ForEach-Object { [ordered]@{component=$_;version=$modules[$_].version;targets=@($modules[$_].targets | Sort-Object)} })
    notices=@($records | Sort-Object component,source | Select-Object component,version,source,scope,sha256)
    notice_sha256=(Get-FileHash -LiteralPath $noticePath -Algorithm SHA256).Hash.ToLowerInvariant()
}
[IO.File]::WriteAllText((Join-Path $outputRoot 'third-party-manifest.json'),(($manifest | ConvertTo-Json -Depth 7)+"`n"),$utf8)
[pscustomobject]@{Notice=$noticePath; Modules=$manifest.modules.Count; Notices=$records.Count; Bytes=(Get-Item -LiteralPath $noticePath).Length; SHA256=$manifest.notice_sha256}
