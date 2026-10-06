[CmdletBinding()]
param()

$ErrorActionPreference = "Stop"
Set-StrictMode -Version Latest
$ProgressPreference = "SilentlyContinue"

$ProductVersion = "0.4.0"
$HostName = "com.vkarmani.browser"
$ExtensionId = "bfcnangfjclakpliejpaamnalnjbmlgh"
$SingBoxVersion = "1.14.1"
$SingBoxArchiveSha256 = "5197f16d492d93202dc623622149a6ed040f8eca263128f91d603f2b901baa89"
$HelperSha256 = "75bc3f9ee22a53f260f8877a9160efb3687e0d8014c799c35a4cb1d4ccefda9a"
$BootstrapSha256 = "d5c5b075bed904a4604a8097811881d3fc86078bd24efe20c6065497af450ab2"

$ScriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$ExtensionDir = (Resolve-Path (Join-Path $ScriptDir "..")).Path
$RuntimeDir = Join-Path $ExtensionDir "runtime\windows-amd64"
$HelperSource = Join-Path $RuntimeDir "vkarmani-browser-helper.exe"
$BootstrapSource = Join-Path $RuntimeDir "bootstrap.json"
$BundledSingBoxZip = Join-Path $RuntimeDir "sing-box-1.14.1-windows-amd64.zip"

$InstallRoot = Join-Path $env:LOCALAPPDATA "VKarmaniBrowserVPN"
$VersionDir = Join-Path $InstallRoot ("versions\" + $ProductVersion)
$ManifestPath = Join-Path $InstallRoot ($HostName + ".json")
$HelperDest = Join-Path $VersionDir "vkarmani-browser-helper.exe"
$SingBoxDest = Join-Path $VersionDir "sing-box.exe"
$BootstrapDest = Join-Path $VersionDir "bootstrap.json"
$InstallInfoPath = Join-Path $InstallRoot "install.json"

function Write-Step([string]$Text) {
    Write-Host "[VKarmani] $Text" -ForegroundColor Cyan
}

function Get-Sha256([string]$Path) {
    return (Get-FileHash -Algorithm SHA256 -LiteralPath $Path).Hash.ToLowerInvariant()
}

function Assert-FileHash([string]$Path, [string]$Expected, [string]$Label) {
    if (-not (Test-Path -LiteralPath $Path -PathType Leaf)) { throw "$Label not found: $Path" }
    $Actual = Get-Sha256 $Path
    if ($Actual -ne $Expected) { throw "$Label SHA256 mismatch. Expected $Expected, got $Actual" }
}

if (-not [Environment]::Is64BitOperatingSystem) {
    throw "This VKarmani package currently supports Windows x64 only."
}

Assert-FileHash $HelperSource $HelperSha256 "VKarmani helper"
Assert-FileHash $BootstrapSource $BootstrapSha256 "bootstrap.json"

$TempDir = Join-Path $env:TEMP ("vkarmani-browser-vpn-" + [guid]::NewGuid().ToString("N"))
New-Item -ItemType Directory -Force -Path $TempDir | Out-Null

try {
    $VerifiedZip = Join-Path $TempDir "sing-box.zip"
    $HaveVerifiedZip = $false

    if (Test-Path -LiteralPath $BundledSingBoxZip -PathType Leaf) {
        Write-Step "Checking bundled sing-box $SingBoxVersion core..."
        if ((Get-Sha256 $BundledSingBoxZip) -eq $SingBoxArchiveSha256) {
            Copy-Item -LiteralPath $BundledSingBoxZip -Destination $VerifiedZip -Force
            $HaveVerifiedZip = $true
        } else {
            throw "Bundled sing-box archive failed SHA256 verification. Installation stopped."
        }
    }

    if (-not $HaveVerifiedZip) {
        [Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12
        $Official = "https://github.com/SagerNet/sing-box/releases/download/v1.14.1/sing-box-1.14.1-windows-amd64.zip"
        $Sources = @(
            @{ Name = "GitHub official"; Url = $Official },
            @{ Name = "SourceForge exact mirror"; Url = "https://sourceforge.net/projects/sing-box.mirror/files/v1.14.1/sing-box-1.14.1-windows-amd64.zip/download" },
            @{ Name = "ghproxy.net mirror"; Url = "https://ghproxy.net/$Official" },
            @{ Name = "ghfast.top mirror"; Url = "https://ghfast.top/$Official" },
            @{ Name = "gh-proxy.com mirror"; Url = "https://gh-proxy.com/$Official" }
        )

        Write-Step "VPN core is not installed separately. Downloading the pinned official sing-box archive automatically..."
        $Errors = New-Object System.Collections.Generic.List[string]
        foreach ($Source in $Sources) {
            try {
                if (Test-Path -LiteralPath $VerifiedZip) { Remove-Item -LiteralPath $VerifiedZip -Force }
                Write-Host ("  -> " + $Source.Name)
                Invoke-WebRequest -UseBasicParsing -Uri $Source.Url -OutFile $VerifiedZip -TimeoutSec 120 -MaximumRedirection 10
                $Actual = Get-Sha256 $VerifiedZip
                if ($Actual -ne $SingBoxArchiveSha256) {
                    throw "downloaded bytes have unexpected SHA256"
                }
                $HaveVerifiedZip = $true
                Write-Host ("     verified SHA256 " + $Actual) -ForegroundColor Green
                break
            } catch {
                $Errors.Add(($Source.Name + ": " + $_.Exception.Message))
                Write-Host ("     failed, trying next source") -ForegroundColor Yellow
            }
        }
        if (-not $HaveVerifiedZip) {
            throw ("Unable to obtain verified sing-box core automatically. Attempts: " + ($Errors -join " | "))
        }
    }

    $ExtractDir = Join-Path $TempDir "singbox"
    Expand-Archive -LiteralPath $VerifiedZip -DestinationPath $ExtractDir -Force
    $SingBoxFound = Get-ChildItem -LiteralPath $ExtractDir -Recurse -File -Filter "sing-box.exe" | Select-Object -First 1
    if (-not $SingBoxFound) { throw "sing-box.exe not found in verified archive" }

    Write-Step "Preparing versioned local runtime..."
    New-Item -ItemType Directory -Force -Path $VersionDir | Out-Null
    Copy-Item -LiteralPath $HelperSource -Destination $HelperDest -Force
    Copy-Item -LiteralPath $BootstrapSource -Destination $BootstrapDest -Force
    Copy-Item -LiteralPath $SingBoxFound.FullName -Destination $SingBoxDest -Force

    Assert-FileHash $HelperDest $HelperSha256 "installed helper"
    Assert-FileHash $BootstrapDest $BootstrapSha256 "installed bootstrap.json"

    $VersionOutput = @(& $SingBoxDest version 2>&1)
    if ($LASTEXITCODE -ne 0) { throw "sing-box version check failed" }
    $VersionLine = ($VersionOutput | Select-Object -First 1 | Out-String).Trim()
    if ($VersionLine -notmatch "1\.14\.1") { throw "Unexpected sing-box version: $VersionLine" }

    $Manifest = [ordered]@{
        name = $HostName
        description = "VKarmani Browser VPN native helper"
        path = $HelperDest
        type = "stdio"
        allowed_origins = @("chrome-extension://$ExtensionId/")
    } | ConvertTo-Json -Depth 4
    New-Item -ItemType Directory -Force -Path $InstallRoot | Out-Null
    [System.IO.File]::WriteAllText($ManifestPath, $Manifest, (New-Object System.Text.UTF8Encoding($false)))

    # Chrome key is also used by Opera. Additional browser-specific keys cover Edge/Brave/Chromium/Vivaldi.
    $RegistryRoots = @(
        "HKCU:\Software\Google\Chrome\NativeMessagingHosts\$HostName",
        "HKCU:\Software\Chromium\NativeMessagingHosts\$HostName",
        "HKCU:\Software\Microsoft\Edge\NativeMessagingHosts\$HostName",
        "HKCU:\Software\BraveSoftware\Brave-Browser\NativeMessagingHosts\$HostName",
        "HKCU:\Software\Vivaldi\NativeMessagingHosts\$HostName",
        "HKCU:\Software\Opera Software\Opera Stable\NativeMessagingHosts\$HostName"
    )
    foreach ($Key in $RegistryRoots) {
        New-Item -Force -Path $Key | Out-Null
        Set-Item -Path $Key -Value $ManifestPath
    }

    $InstallInfo = [ordered]@{
        product_version = $ProductVersion
        extension_id = $ExtensionId
        installed_at = (Get-Date).ToUniversalTime().ToString("o")
        runtime_dir = $VersionDir
        helper_sha256 = $HelperSha256
        sing_box_version = $SingBoxVersion
        sing_box_archive_sha256 = $SingBoxArchiveSha256
        bootstrap_sha256 = $BootstrapSha256
    } | ConvertTo-Json -Depth 4
    [System.IO.File]::WriteAllText($InstallInfoPath, $InstallInfo, (New-Object System.Text.UTF8Encoding($false)))

    Write-Host ""
    Write-Host "VKarmani Browser VPN runtime installed successfully." -ForegroundColor Green
    Write-Host "No separate VPN application is required." -ForegroundColor Green
    Write-Host "Runtime: $VersionDir"
    Write-Host "Extension folder: $ExtensionDir"
    Write-Host "Extension ID: $ExtensionId"

    # Put the exact folder path on clipboard and open it for the final browser-security step.
    try { Set-Clipboard -Value $ExtensionDir } catch {}
    try { Start-Process -FilePath "explorer.exe" -ArgumentList @($ExtensionDir) } catch {}

    $BrowserCandidates = @(
        @{ Path = (Join-Path $env:ProgramFiles "Google\Chrome\Application\chrome.exe"); Url = "chrome://extensions/" },
        @{ Path = (Join-Path ${env:ProgramFiles(x86)} "Google\Chrome\Application\chrome.exe"); Url = "chrome://extensions/" },
        @{ Path = (Join-Path $env:LOCALAPPDATA "Google\Chrome\Application\chrome.exe"); Url = "chrome://extensions/" },
        @{ Path = (Join-Path $env:ProgramFiles "Microsoft\Edge\Application\msedge.exe"); Url = "edge://extensions/" },
        @{ Path = (Join-Path ${env:ProgramFiles(x86)} "Microsoft\Edge\Application\msedge.exe"); Url = "edge://extensions/" },
        @{ Path = (Join-Path $env:ProgramFiles "BraveSoftware\Brave-Browser\Application\brave.exe"); Url = "brave://extensions/" },
        @{ Path = (Join-Path $env:LOCALAPPDATA "BraveSoftware\Brave-Browser\Application\brave.exe"); Url = "brave://extensions/" },
        @{ Path = (Join-Path $env:LOCALAPPDATA "Programs\Opera\launcher.exe"); Url = "opera://extensions/" },
        @{ Path = (Join-Path $env:LOCALAPPDATA "Vivaldi\Application\vivaldi.exe"); Url = "vivaldi://extensions/" }
    )
    foreach ($Candidate in $BrowserCandidates) {
        if ($Candidate.Path -and (Test-Path -LiteralPath $Candidate.Path -PathType Leaf)) {
            try { Start-Process -FilePath $Candidate.Path -ArgumentList $Candidate.Url } catch {}
            break
        }
    }

    Write-Host ""
    Write-Host "FINAL BROWSER STEP:" -ForegroundColor Yellow
    Write-Host "Enable Developer mode -> Load unpacked -> choose the folder that just opened." -ForegroundColor Yellow
    Write-Host "The folder path has also been copied to the clipboard." -ForegroundColor Yellow
}
finally {
    if (Test-Path -LiteralPath $TempDir) { Remove-Item -LiteralPath $TempDir -Recurse -Force -ErrorAction SilentlyContinue }
}
