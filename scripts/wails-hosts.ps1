# VibeX Workbench - Windows hosts file fix
# Ensures wails.localhost resolves to 127.0.0.1 on Windows
# Self-elevates if not running as admin

$isAdmin = ([Security.Principal.WindowsPrincipal] [Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)
if (-not $isAdmin) {
    Write-Host "[wails-hosts] Requesting admin elevation..."
    Start-Process powershell.exe -ArgumentList "-NoProfile -ExecutionPolicy Bypass -File `"$PSCommandPath`"" -Verb RunAs
    exit 0
}

$hostsPath = "C:\Windows\System32\drivers\etc\hosts"
$entry = "127.0.0.1  wails.localhost"
$marker = "# Added by VibeX Workbench"

if (Select-String -Path $hostsPath -Pattern "wails.localhost" -Quiet) {
    Write-Host "[wails-hosts] wails.localhost already in hosts."
} else {
    Write-Host "[wails-hosts] Adding wails.localhost..."
    $lines = @(Get-Content $hostsPath)
    $lines += ""
    $lines += $marker
    $lines += $entry
    Set-Content -Path $hostsPath -Value ($lines -join "`r`n")
    Write-Host "[wails-hosts] Done."
}
