$Root = (Resolve-Path (Join-Path (Split-Path -Parent $MyInvocation.MyCommand.Path) "..\..")).Path
& (Join-Path $Root "extension\setup\uninstall.ps1")
