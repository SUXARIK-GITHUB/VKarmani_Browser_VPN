$ErrorActionPreference = "Continue"
$HostName = "com.vkarmani.browser"
$InstallRoot = Join-Path $env:LOCALAPPDATA "VKarmaniBrowserVPN"
$Manifest = Join-Path $InstallRoot "$HostName.json"
$Info = Join-Path $InstallRoot "install.json"

Write-Host "===== VKarmani Browser VPN diagnostics ====="
Write-Host "InstallRoot=$InstallRoot"
Write-Host "ManifestExists=$(Test-Path -LiteralPath $Manifest)"
Write-Host "InstallInfoExists=$(Test-Path -LiteralPath $Info)"
if (Test-Path -LiteralPath $Info) {
    try {
        $i = Get-Content -Raw -LiteralPath $Info | ConvertFrom-Json
        Write-Host "ProductVersion=$($i.product_version)"
        Write-Host "RuntimeDir=$($i.runtime_dir)"
        Write-Host "ExpectedExtensionID=$($i.extension_id)"
        $Helper = Join-Path $i.runtime_dir "vkarmani-browser-helper.exe"
        $Sing = Join-Path $i.runtime_dir "sing-box.exe"
        $Boot = Join-Path $i.runtime_dir "bootstrap.json"
        Write-Host "HelperExists=$(Test-Path -LiteralPath $Helper)"
        Write-Host "SingBoxExists=$(Test-Path -LiteralPath $Sing)"
        Write-Host "BootstrapExists=$(Test-Path -LiteralPath $Boot)"
        if (Test-Path -LiteralPath $Helper) { Write-Host "HelperSHA256=$((Get-FileHash -Algorithm SHA256 -LiteralPath $Helper).Hash.ToLowerInvariant())" }
        if (Test-Path -LiteralPath $Sing) { & $Sing version | Select-Object -First 2 }
    } catch { Write-Host "InstallInfoReadError=$($_.Exception.Message)" }
}
if (Test-Path -LiteralPath $Manifest) {
    try {
        $m = Get-Content -Raw -LiteralPath $Manifest | ConvertFrom-Json
        Write-Host "NativeHostName=$($m.name)"
        Write-Host "NativeHostPath=$($m.path)"
        Write-Host "AllowedOrigin=$($m.allowed_origins -join ',')"
    } catch { Write-Host "ManifestReadError=$($_.Exception.Message)" }
}
$Keys = @(
    "HKCU:\Software\Google\Chrome\NativeMessagingHosts\$HostName",
    "HKCU:\Software\Chromium\NativeMessagingHosts\$HostName",
    "HKCU:\Software\Microsoft\Edge\NativeMessagingHosts\$HostName",
    "HKCU:\Software\BraveSoftware\Brave-Browser\NativeMessagingHosts\$HostName",
    "HKCU:\Software\Vivaldi\NativeMessagingHosts\$HostName",
    "HKCU:\Software\Opera Software\Opera Stable\NativeMessagingHosts\$HostName"
)
foreach ($Key in $Keys) {
    $Value = if (Test-Path $Key) { (Get-Item $Key).GetValue('') } else { 'ABSENT' }
    Write-Host "$Key=$Value"
}
Write-Host "NOTE: subscription URL and VLESS credentials are intentionally not printed."
