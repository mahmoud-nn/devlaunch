param(
  [Parameter(Mandatory = $true)][string]$WorkingDirectory,
  [Parameter(Mandatory = $true)][string]$TabName,
  [Parameter(Mandatory = $true)][string]$Command,
  [Parameter(Mandatory = $true)][string]$PIDFilePath
)

$arguments = @("new-tab", "-d", $WorkingDirectory)
$escapedPidFilePath = $PIDFilePath.Replace("'", "''")
$bootstrap = "& { `$PID | Set-Content -Path '$escapedPidFilePath'; $Command }"

if ([string]::IsNullOrWhiteSpace($TabName)) {
  $arguments += @("powershell", "-NoExit", "-Command", $bootstrap)
}
else {
  $arguments += @("--title", $TabName, "powershell", "-NoExit", "-Command", $bootstrap)
}

& wt @arguments | Out-Null

for ($i = 0; $i -lt 50; $i++) {
  if (Test-Path $PIDFilePath) {
    $pidValue = (Get-Content -Path $PIDFilePath -ErrorAction SilentlyContinue | Select-Object -First 1).Trim()
    if (-not [string]::IsNullOrWhiteSpace($pidValue)) {
      $pidValue
      exit 0
    }
  }
  Start-Sleep -Milliseconds 100
}

throw "Unable to capture interactive shell PID for tab '$TabName'."
