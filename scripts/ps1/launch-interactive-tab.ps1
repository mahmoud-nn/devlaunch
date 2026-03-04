param(
  [Parameter(Mandatory = $true)][string]$WorkingDirectory,
  [Parameter(Mandatory = $true)][string]$TabName,
  [Parameter(Mandatory = $true)][string]$Command
)

$arguments = @("new-tab", "-d", $WorkingDirectory)

if ([string]::IsNullOrWhiteSpace($TabName)) {
  $arguments += @("powershell", "-NoExit", "-Command", $Command)
}
else {
  $arguments += @("--title", $TabName, "powershell", "-NoExit", "-Command", $Command)
}

& wt @arguments | Out-Null
