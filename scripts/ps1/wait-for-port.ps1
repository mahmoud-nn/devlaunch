param(
  [Parameter(Mandatory = $true)][int]$Port,
  [Parameter(Mandatory = $true)][int]$TimeoutMs
)

try {
  $client = New-Object System.Net.Sockets.TcpClient
  $async = $client.BeginConnect("127.0.0.1", $Port, $null, $null)
  $ok = $async.AsyncWaitHandle.WaitOne($TimeoutMs, $false)
  if (-not $ok) {
    $client.Close()
    exit 1
  }

  $client.EndConnect($async)
  $client.Close()
  "ready"
  exit 0
} catch {
  exit 1
}
