param(
    [string]$Executable = (Join-Path $PSScriptRoot '../dist/release/myenv-windows-amd64.exe'),
    [string[]]$CommandArguments = @('--version'),
    [ValidateRange(1, 1000)][int]$Samples = 20,
    [string]$ExpectedGeneration = '',
    [switch]$StatusResponse,
    [Parameter(Mandatory)][string]$Output
)
$ErrorActionPreference = 'Stop'
if ($StatusResponse -and !$ExpectedGeneration) { throw 'StatusResponse requires ExpectedGeneration.' }
if (!$IsWindows) { throw 'This measurement requires Windows PowerShell 7.' }
$binary = (Resolve-Path -LiteralPath $Executable).Path
$reportPath = [IO.Path]::GetFullPath($Output)
if (Test-Path -LiteralPath $reportPath) { throw 'Report already exists; choose a new output path.' }
if (!('MyEnvPeakMemory' -as [type])) {
    Add-Type -TypeDefinition @'
using System;
using System.ComponentModel;
using System.Runtime.InteropServices;
using System.IO;
using System.Text;
using System.Threading.Tasks;
public static class MyEnvPeakMemory {
    public static async Task<string> ReadBoundedJSON(TextReader reader) {
        var text = new StringBuilder();
        var buffer = new char[4096];
        bool overflow = false;
        int count;
        while ((count = await reader.ReadAsync(buffer, 0, buffer.Length)) != 0) {
            if (text.Length + count > 65536) overflow = true;
            if (!overflow) text.Append(buffer, 0, count);
        }
        if (overflow) throw new InvalidOperationException("Sync JSON exceeded 64 Ki characters.");
        return text.ToString();
    }
    [StructLayout(LayoutKind.Sequential)]
    public struct Counters {
        public uint Size, PageFaultCount;
        public UIntPtr PeakWorkingSet, WorkingSet, QuotaPeakPagedPool,
            QuotaPagedPool, QuotaPeakNonPagedPool, QuotaNonPagedPool,
            Pagefile, PeakPagefile;
    }
    [DllImport("psapi.dll", SetLastError=true)]
    [return: MarshalAs(UnmanagedType.Bool)]
    private static extern bool GetProcessMemoryInfo(IntPtr process, out Counters counters, uint size);
    public static ulong ReadPeak(IntPtr process) {
        Counters counters;
        if (!GetProcessMemoryInfo(process, out counters, (uint)Marshal.SizeOf<Counters>()))
            throw new Win32Exception(Marshal.GetLastWin32Error());
        if (counters.PeakWorkingSet == UIntPtr.Zero)
            throw new InvalidOperationException("Peak working set is unavailable.");
        return counters.PeakWorkingSet.ToUInt64();
    }
}
'@
}
$hash = (Get-FileHash -LiteralPath $binary -Algorithm SHA256).Hash.ToLowerInvariant()
$peaks = [Collections.Generic.List[ulong]]::new()
for ($iteration = 0; $iteration -lt $Samples; $iteration++) {
    $process = [Diagnostics.Process]::new()
    $process.StartInfo.FileName = $binary
    foreach ($argument in $CommandArguments) { $process.StartInfo.ArgumentList.Add($argument) }
    $process.StartInfo.UseShellExecute = $false
    $process.StartInfo.CreateNoWindow = $true
    $process.StartInfo.RedirectStandardOutput = $true
    $process.StartInfo.RedirectStandardError = $true
    $process.StartInfo.RedirectStandardInput = $true
    try {
        if (!$process.Start()) { throw 'Process start failed.' }
        # Retain the actual process handle across exit; never reopen by PID.
        $handle = $process.Handle
        $process.StandardInput.Close()
        if ($ExpectedGeneration) {
            $stdout = [MyEnvPeakMemory]::ReadBoundedJSON($process.StandardOutput)
        } else {
            $stdout = $process.StandardOutput.BaseStream.CopyToAsync([IO.Stream]::Null)
        }
        $stderr = $process.StandardError.BaseStream.CopyToAsync([IO.Stream]::Null)
        if (!$process.WaitForExit(10000)) {
            $process.Kill($true)
            $process.WaitForExit()
            throw 'Measured command timed out.'
        }
        $peak = [MyEnvPeakMemory]::ReadPeak($handle)
        $commandOutput = $stdout.GetAwaiter().GetResult()
        $null = $stderr.GetAwaiter().GetResult()
        if ($process.ExitCode -ne 0) { throw "Measured command exited with $($process.ExitCode)." }
        if ($ExpectedGeneration) {
            $sync = $commandOutput | ConvertFrom-Json
            if ($StatusResponse) {
                if ($sync.ok -ne $true -or $sync.changed -ne $false -or $sync.data.environment -cne 'ready' -or
                    $sync.data.generation.id -cne $ExpectedGeneration) { throw 'Measured status was not ready at the expected generation.' }
            } elseif ($sync.ok -ne $true -or $sync.changed -ne $false -or $sync.data.Changed -ne $false -or
                $sync.data.lock_changed -ne $false -or $sync.data.native_lock_changed -ne $false -or
                $sync.data.Generation.id -cne $ExpectedGeneration) { throw 'Measured sync was not unchanged.' }
        }
        $peaks.Add($peak)
    } finally { $process.Dispose() }
}
$report = [ordered]@{
    executable = $binary
    sha256 = $hash
    arguments = $CommandArguments
    expected_unchanged_generation = $ExpectedGeneration
    samples_peak_working_set_bytes = $peaks.ToArray()
    max_peak_working_set_bytes = ($peaks | Measure-Object -Maximum).Maximum
    scope = 'Single CLI process only; excludes child processes and measurement host.'
    method = 'GetProcessMemoryInfo PeakWorkingSetSize using retained process handle after exit; all samples included; no cache eviction; not a latency measurement.'
    os = [Runtime.InteropServices.RuntimeInformation]::OSDescription
}
$null = [IO.Directory]::CreateDirectory([IO.Path]::GetDirectoryName($reportPath))
$bytes = [Text.UTF8Encoding]::new($false).GetBytes(($report | ConvertTo-Json -Depth 4) + "`n")
$file = [IO.File]::Open($reportPath, [IO.FileMode]::CreateNew, [IO.FileAccess]::Write)
try { $file.Write($bytes, 0, $bytes.Length) } finally { $file.Dispose() }
[pscustomobject]@{ Samples=$Samples; PeakMiB=($report.max_peak_working_set_bytes / 1MB); SHA256=$hash }
