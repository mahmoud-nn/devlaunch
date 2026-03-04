param(
  [Parameter(Mandatory = $true)][string]$Path
)

$proc = Start-Process -FilePath $Path -PassThru
$proc.Id
