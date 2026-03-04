param(
  [Parameter(Mandatory = $true)][string]$WorkingDirectory,
  [Parameter(Mandatory = $true)][string]$Command
)

$encoded = [Convert]::ToBase64String([Text.Encoding]::Unicode.GetBytes($Command))
$proc = Start-Process -FilePath "powershell" `
  -ArgumentList @("-NoProfile", "-WindowStyle", "Hidden", "-EncodedCommand", $encoded) `
  -WorkingDirectory $WorkingDirectory `
  -PassThru

$proc.Id
