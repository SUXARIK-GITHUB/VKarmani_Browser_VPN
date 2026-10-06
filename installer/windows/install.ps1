$ErrorActionPreference = "Stop"
$Root = (Resolve-Path (Join-Path (Split-Path -Parent $MyInvocation.MyCommand.Path) "..\..")).Path
& (Join-Path $Root "extension\setup\install.ps1")
exit $LASTEXITCODE
