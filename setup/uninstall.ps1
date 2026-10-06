$ErrorActionPreference = "Stop"
$HostName = "com.vkarmani.browser"
$InstallRoot = Join-Path $env:LOCALAPPDATA "VKarmaniBrowserVPN"
$Keys = @(
    "HKCU:\Software\Google\Chrome\NativeMessagingHosts\$HostName",
    "HKCU:\Software\Chromium\NativeMessagingHosts\$HostName",
    "HKCU:\Software\Microsoft\Edge\NativeMessagingHosts\$HostName",
    "HKCU:\Software\BraveSoftware\Brave-Browser\NativeMessagingHosts\$HostName",
    "HKCU:\Software\Vivaldi\NativeMessagingHosts\$HostName",
    "HKCU:\Software\Opera Software\Opera Stable\NativeMessagingHosts\$HostName"
)
foreach ($Key in $Keys) {
    if (Test-Path $Key) { Remove-Item -Recurse -Force $Key }
}
if (Test-Path -LiteralPath $InstallRoot) { Remove-Item -LiteralPath $InstallRoot -Recurse -Force }
Write-Host "VKarmani native runtime removed. Remove the unpacked browser extension separately." -ForegroundColor Green
