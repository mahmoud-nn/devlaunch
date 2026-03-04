param(
  [Parameter(Mandatory = $true)][string]$Path
)

Start-Process explorer $Path | Out-Null
