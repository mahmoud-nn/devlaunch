param(
  [Parameter(Mandatory = $true)][string]$Name
)

$proc = Get-Process -Name $Name -ErrorAction SilentlyContinue | Select-Object -First 1
if ($null -ne $proc) {
  "true"
} else {
  "false"
}
