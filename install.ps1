# Installs only LiuTangLei/tailcat-quic for the current Windows user.
# No elevation, execution-policy changes, services or network settings.
[CmdletBinding()]
param(
    [string]$Version = 'v0.7.0-quic.2',
    [string]$InstallDir = (Join-Path $env:LOCALAPPDATA 'Programs\tailcat-quic'),
    [switch]$NoPath,
    [switch]$DryRun
)
$ErrorActionPreference = 'Stop'
if ($Version -notmatch '^v[0-9]+\.[0-9]+\.[0-9]+(-quic\.[0-9]+)?$') { throw 'Invalid QUIC release version.' }
if (-not [IO.Path]::IsPathRooted($InstallDir)) { throw 'InstallDir must be absolute.' }
$nativeArch = if ($env:PROCESSOR_ARCHITEW6432) { $env:PROCESSOR_ARCHITEW6432 } else { $env:PROCESSOR_ARCHITECTURE }
switch ($nativeArch.ToUpperInvariant()) {
    'ARM64' { $arch = 'arm64' }
    'AMD64' { $arch = 'amd64' }
    default { throw "No published executable for architecture: $nativeArch" }
}
$asset = "tailcat_$($Version.Substring(1))_windows_$arch.zip"
$base = "https://github.com/LiuTangLei/tailcat-quic/releases/download/$Version"
if ($DryRun) {
    [PSCustomObject]@{ Version=$Version; Architecture=$arch; URL="$base/$asset"; Destination=(Join-Path $InstallDir 'tailcat.exe') }
    return
}
[Net.ServicePointManager]::SecurityProtocol = [Net.ServicePointManager]::SecurityProtocol -bor [Net.SecurityProtocolType]::Tls12
$temp = Join-Path ([IO.Path]::GetTempPath()) ('tailcat-install-' + [Guid]::NewGuid().ToString('N'))
$stage = $null
New-Item -ItemType Directory -Path $temp | Out-Null
try {
    $checksums = (Invoke-WebRequest -UseBasicParsing -Uri "$base/checksums.txt").Content
    $matches = @($checksums -split "`n" | Where-Object { $_ -match ('^([0-9a-f]{64})\s+\*?' + [regex]::Escape($asset) + '\s*$') })
    if ($matches.Count -ne 1) { throw 'Missing or ambiguous release checksum.' }
    $expected = ($matches[0] -split '\s+')[0]
    $archive = Join-Path $temp 'archive.zip'
    Invoke-WebRequest -UseBasicParsing -Uri "$base/$asset" -OutFile $archive
    if ((Get-FileHash -Algorithm SHA256 $archive).Hash.ToLowerInvariant() -ne $expected) { throw 'SHA-256 mismatch; nothing installed.' }
    Add-Type -AssemblyName System.IO.Compression.FileSystem
    $zip = [IO.Compression.ZipFile]::OpenRead($archive)
    try {
        $entries = @($zip.Entries | Where-Object { $_.FullName -ceq 'tailcat.exe' })
        if ($entries.Count -ne 1) { throw 'Invalid archive executable layout.' }
        $exe = Join-Path $temp 'tailcat.exe'
        [IO.Compression.ZipFileExtensions]::ExtractToFile($entries[0], $exe, $false)
    } finally { $zip.Dispose() }
    $actualVersion = (& $exe version | Out-String).Trim()
    if ($LASTEXITCODE -ne 0 -or $actualVersion -ne $Version) { throw 'Executable version mismatch.' }
    New-Item -ItemType Directory -Force -Path $InstallDir | Out-Null
    $stage = Join-Path $InstallDir ('.tailcat-' + [Guid]::NewGuid().ToString('N') + '.exe')
    Copy-Item -LiteralPath $exe -Destination $stage
    $destination = Join-Path $InstallDir 'tailcat.exe'
    if (Test-Path -LiteralPath $destination) {
        [IO.File]::Replace($stage, $destination, $null)
    } else { [IO.File]::Move($stage, $destination) }
    $stage = $null
    if (-not $NoPath) {
        $oldPath = [Environment]::GetEnvironmentVariable('Path','User')
        $items = @($oldPath -split ';' | Where-Object { $_ })
        if ($items -notcontains $InstallDir) {
            [Environment]::SetEnvironmentVariable('Path', (($items + $InstallDir) -join ';'), 'User')
        }
        if (($env:Path -split ';') -notcontains $InstallDir) { $env:Path = "$InstallDir;$env:Path" }
    }
    Write-Output "Installed $Version at $destination (SHA-256 verified)."
    Write-Output 'No service or network setting was changed; open a new terminal for the updated user PATH.'
} finally {
    if ($stage -and (Test-Path -LiteralPath $stage)) { Remove-Item -LiteralPath $stage -Force }
    Remove-Item -LiteralPath $temp -Force -Recurse
}
