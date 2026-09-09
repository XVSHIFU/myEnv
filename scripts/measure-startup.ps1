param(
    [string]$Executable = (Join-Path $PSScriptRoot '../dist/myenv.exe'),
    [ValidateRange(20, 1000)][int]$Samples = 50,
    [string]$Output = (Join-Path $PSScriptRoot '../.build/perf/startup-direct-windows.json')
)
$ErrorActionPreference = 'Stop'
$reportPath = [System.IO.Path]::GetFullPath($Output)
if (Test-Path -LiteralPath $reportPath) { throw 'Report already exists; choose a new output path' }
$resolvedExecutable = (Resolve-Path -LiteralPath $Executable).Path
$binaryHash = (Get-FileHash -LiteralPath $resolvedExecutable -Algorithm SHA256).Hash
$measurements = foreach ($argument in @('--version', '--help')) {
    $elapsedSamples = [System.Collections.Generic.List[double]]::new()
    for ($iteration = 0; $iteration -le $Samples; $iteration++) {
        $process = [System.Diagnostics.Process]::new()
        $process.StartInfo.FileName = $resolvedExecutable
        $process.StartInfo.ArgumentList.Add($argument)
        $process.StartInfo.UseShellExecute = $false
        $process.StartInfo.CreateNoWindow = $true
        $process.StartInfo.RedirectStandardOutput = $true
        $process.StartInfo.RedirectStandardError = $true
        try {
            $timer = [System.Diagnostics.Stopwatch]::StartNew()
            if (!$process.Start()) { throw 'Process start failed' }
            $stdout = $process.StandardOutput.ReadToEndAsync()
            $stderr = $process.StandardError.ReadToEndAsync()
            if (!$process.WaitForExit(10000)) {
                $process.Kill($true)
                $process.WaitForExit()
                throw 'Startup measurement timed out'
            }
            $outputText = $stdout.GetAwaiter().GetResult()
            $errorText = $stderr.GetAwaiter().GetResult()
            $timer.Stop()
            if ($process.ExitCode -ne 0) { throw "Command failed: $errorText" }
            if ($errorText.Length -ne 0) { throw "Unexpected diagnostic output: $errorText" }
            if ([string]::IsNullOrWhiteSpace($outputText) -or
                ($argument -eq '--help' -and !$outputText.Contains('Usage:')) -or
                ($argument -eq '--version' -and !$outputText.StartsWith('myenv version '))) {
                throw "Unexpected $argument output"
            }
            if ($iteration -eq 0) { $firstObserved = $timer.Elapsed.TotalMilliseconds }
            else { $elapsedSamples.Add($timer.Elapsed.TotalMilliseconds) }
        } finally { $process.Dispose() }
    }
    $ordered = @($elapsedSamples | Sort-Object)
    [pscustomobject]@{
        argument = $argument
        binary_sha256 = $binaryHash
        samples_ms = $elapsedSamples.ToArray()
        first_observed_ms = $firstObserved
        median_ms = $ordered[[int][Math]::Ceiling($Samples * 0.5) - 1]
        p95_ms = $ordered[[int][Math]::Ceiling($Samples * 0.95) - 1]
        method = 'Process.Start, no shell, async stdout/stderr drain, wait for exit; first sample excluded; no OS cache eviction'
        os = [System.Runtime.InteropServices.RuntimeInformation]::OSDescription
    }
}
if ((Get-FileHash -LiteralPath $resolvedExecutable -Algorithm SHA256).Hash -ne $binaryHash) {
    throw 'Executable changed during measurement'
}
$null = New-Item -ItemType Directory -Path ([System.IO.Path]::GetDirectoryName($reportPath)) -Force
$reportBytes = [System.Text.UTF8Encoding]::new($false).GetBytes(($measurements | ConvertTo-Json -Depth 4) + "`n")
$reportFile = [System.IO.File]::Open($reportPath, [System.IO.FileMode]::CreateNew, [System.IO.FileAccess]::Write)
try { $reportFile.Write($reportBytes, 0, $reportBytes.Length) }
finally { $reportFile.Dispose() }
$measurements | Select-Object argument, first_observed_ms, median_ms, p95_ms, binary_sha256
