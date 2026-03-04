param(
  [Parameter(Mandatory = $true)][string]$WorkingDirectory,
  [Parameter(Mandatory = $true)][string]$TabName,
  [Parameter(Mandatory = $true)][string]$Command
)

$location = $WorkingDirectory.Replace("'", "''")
$composed = "Set-Location '$location'; $Command"

if ([string]::IsNullOrWhiteSpace($TabName)) {
  wt new-tab powershell -NoExit -Command $composed | Out-Null
} else {
  wt new-tab --title $TabName powershell -NoExit -Command $composed | Out-Null
}
