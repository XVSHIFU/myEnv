# Windows-only build utility. Reuses the frontend's myEnv leaf on a forest-green tile.
# WPF renders the SVG geometry without a downloaded image tool or runtime dependency.
param([string]$OutputPath = (Join-Path $PSScriptRoot 'icon.ico'))
$ErrorActionPreference = 'Stop'
Add-Type -AssemblyName PresentationCore, WindowsBase
[xml]$svg = Get-Content -LiteralPath (Join-Path $PSScriptRoot 'icon.svg') -Raw
$brushes = [Windows.Media.BrushConverter]::new()
$tile = $brushes.ConvertFromString($svg.svg.rect.fill)
$ink = $brushes.ConvertFromString($svg.svg.g.stroke)
$pen = [Windows.Media.Pen]::new($ink, [double]$svg.svg.g.'stroke-width')
$pen.StartLineCap = $pen.EndLineCap = [Windows.Media.PenLineCap]::Round
$pen.LineJoin = [Windows.Media.PenLineJoin]::Round
$frames = @()
foreach ($size in @(16, 20, 24, 32, 40, 48, 64, 128, 256)) {
    $visual = [Windows.Media.DrawingVisual]::new()
    $drawing = $visual.RenderOpen()
    $drawing.PushTransform([Windows.Media.ScaleTransform]::new($size / 32.0, $size / 32.0))
    $drawing.DrawRoundedRectangle($tile, $null, [Windows.Rect]::new(1, 1, 30, 30), 7, 7)
    $drawing.PushTransform([Windows.Media.TranslateTransform]::new(4, 4))
    foreach ($path in $svg.svg.g.path) {
        $drawing.DrawGeometry($null, $pen, [Windows.Media.Geometry]::Parse($path.d))
    }
    $drawing.Pop(); $drawing.Pop(); $drawing.Close()
    $bitmap = [Windows.Media.Imaging.RenderTargetBitmap]::new($size, $size, 96, 96, [Windows.Media.PixelFormats]::Pbgra32)
    $bitmap.Render($visual)
    $encoder = [Windows.Media.Imaging.PngBitmapEncoder]::new()
    $encoder.Frames.Add([Windows.Media.Imaging.BitmapFrame]::Create($bitmap))
    $stream = [IO.MemoryStream]::new()
    try {
        $encoder.Save($stream)
        $frames += [pscustomobject]@{Size = $size; Bytes = $stream.ToArray()}
    } finally { $stream.Dispose() }
}
$file = [IO.File]::Create([IO.Path]::GetFullPath($OutputPath))
$writer = [IO.BinaryWriter]::new($file)
try {
    $writer.Write([uint16]0); $writer.Write([uint16]1); $writer.Write([uint16]$frames.Count)
    $offset = 6 + 16 * $frames.Count
    foreach ($frame in $frames) {
        $dimension = if ($frame.Size -eq 256) { 0 } else { $frame.Size }
        $writer.Write([byte]$dimension); $writer.Write([byte]$dimension)
        $writer.Write([byte]0); $writer.Write([byte]0)
        $writer.Write([uint16]1); $writer.Write([uint16]32)
        $writer.Write([uint32]$frame.Bytes.Length); $writer.Write([uint32]$offset)
        $offset += $frame.Bytes.Length
    }
    foreach ($frame in $frames) { $writer.Write([byte[]]$frame.Bytes) }
} finally { $writer.Dispose(); $file.Dispose() }
Write-Output "Generated $($frames.Count) icon sizes: $OutputPath"
