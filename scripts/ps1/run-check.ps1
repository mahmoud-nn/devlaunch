param(
  [Parameter(Mandatory = $true)][string]$WorkingDirectory,
  [Parameter(Mandatory = $true)][string]$Command
)

Set-Location $WorkingDirectory
powershell -NoProfile -Command $Command | Out-Null
exit $LASTEXITCODE
