param(
    [Parameter(Mandatory)][string]$Executable,
    [string]$Icon = (Join-Path $PSScriptRoot '../cmd/myenv-gui/build/windows/icon.ico')
)
$ErrorActionPreference = 'Stop'
if (!$IsWindows) { throw 'Windows GUI resources require Windows' }
$executablePath = (Resolve-Path -LiteralPath $Executable).Path
$iconPath = (Resolve-Path -LiteralPath $Icon).Path
# Group 3 is Wails AppIconID; group 32512 is the icon its native window class loads.
# Supplying both gives the title bar, taskbar/Alt+Tab and Explorer the same identity.
# Update only icon resources in the newly built, unsigned GUI; preserve its manifest.
if (!('MyEnvBuild.IconResource' -as [type])) {
    Add-Type @'
using System;
using System.ComponentModel;
using System.IO;
using System.Runtime.InteropServices;
namespace MyEnvBuild {
    public static class IconResource {
        [DllImport("kernel32.dll", CharSet=CharSet.Unicode, SetLastError=true)]
        static extern IntPtr BeginUpdateResource(string file, bool deleteExistingResources);
        [DllImport("kernel32.dll", SetLastError=true)]
        static extern bool UpdateResource(IntPtr update, IntPtr type, IntPtr name, ushort language, byte[] data, uint size);
        [DllImport("kernel32.dll", SetLastError=true)]
        static extern bool EndUpdateResource(IntPtr update, bool discard);
        public static void Apply(string executable, string icon) {
            byte[] bytes = File.ReadAllBytes(icon);
            using var reader = new BinaryReader(new MemoryStream(bytes));
            if (reader.ReadUInt16() != 0 || reader.ReadUInt16() != 1) throw new InvalidDataException("Expected ICO file");
            ushort count = reader.ReadUInt16();
            if (count == 0 || count > 32 || bytes.Length < 6 + count * 16) throw new InvalidDataException("Invalid icon directory");
            using var group = new MemoryStream();
            using var writer = new BinaryWriter(group);
            writer.Write((ushort)0); writer.Write((ushort)1); writer.Write(count);
            var images = new byte[count][];
            for (ushort i = 0; i < count; i++) {
                byte[] entry = reader.ReadBytes(8);
                uint length = reader.ReadUInt32(), offset = reader.ReadUInt32();
                if (length == 0 || (ulong)offset + length > (ulong)bytes.Length) throw new InvalidDataException("Icon frame outside file");
                images[i] = new byte[length];
                Array.Copy(bytes, offset, images[i], 0, length);
                writer.Write(entry); writer.Write(length); writer.Write((ushort)(i + 1));
            }
            IntPtr update = BeginUpdateResource(executable, false);
            if (update == IntPtr.Zero) throw new Win32Exception(Marshal.GetLastWin32Error());
            bool committed = false;
            try {
                for (int i = 0; i < count; i++) {
                    if (!UpdateResource(update, (IntPtr)3, (IntPtr)(i + 1), 0, images[i], (uint)images[i].Length))
                        throw new Win32Exception(Marshal.GetLastWin32Error());
                }
                byte[] data = group.ToArray();
                if (!UpdateResource(update, (IntPtr)14, (IntPtr)3, 0, data, (uint)data.Length))
                    throw new Win32Exception(Marshal.GetLastWin32Error());
                if (!UpdateResource(update, (IntPtr)14, (IntPtr)32512, 0, data, (uint)data.Length))
                    throw new Win32Exception(Marshal.GetLastWin32Error());
                if (!EndUpdateResource(update, false)) throw new Win32Exception(Marshal.GetLastWin32Error());
                committed = true;
            } finally { if (!committed) EndUpdateResource(update, true); }
        }
    }
}
'@
}
[MyEnvBuild.IconResource]::Apply($executablePath, $iconPath)
Write-Output "Embedded myEnv Windows application icon: $executablePath"
