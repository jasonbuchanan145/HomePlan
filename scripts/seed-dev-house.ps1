param(
  [string]$BaseUrl = "http://localhost:8080",
  [string]$SeedFile = "seed/homeplan-demo-house.json",
  [switch]$Reset
)

$ErrorActionPreference = "Stop"
if ($PSVersionTable.PSVersion.Major -ge 7) {
  $PSNativeCommandUseErrorActionPreference = $true
}

function Invoke-JsonRequest {
  param(
    [string]$Method,
    [string]$Uri,
    [string]$Body = $null
  )

  try {
    if ($null -eq $Body) {
      return Invoke-WebRequest -Method $Method -Uri $Uri -UseBasicParsing
    }
    return Invoke-WebRequest -Method $Method -Uri $Uri -Body $Body -ContentType "application/json" -UseBasicParsing
  } catch {
    $response = $_.Exception.Response
    if ($response) {
      $bodyText = ""
      if ($response.Content) {
        $bodyText = $response.Content.ReadAsStringAsync().GetAwaiter().GetResult()
      } elseif ($response.GetResponseStream()) {
        $reader = [System.IO.StreamReader]::new($response.GetResponseStream())
        $bodyText = $reader.ReadToEnd()
      }
      throw "$Method $Uri failed with status $([int]$response.StatusCode): $bodyText"
    }
    throw "$Method $Uri failed: $($_.Exception.Message)"
  }
}

$repoRoot = Resolve-Path (Join-Path $PSScriptRoot "..")
$seedPath = Resolve-Path (Join-Path $repoRoot $SeedFile)
$normalizedBaseUrl = $BaseUrl.TrimEnd("/")
$devHouseUrl = "$normalizedBaseUrl/api/dev/users/user-1/house/current"

if ($Reset) {
  Write-Host "Resetting dev user 1 house..."
  Invoke-JsonRequest -Method "DELETE" -Uri $devHouseUrl | Out-Null
}

Write-Host "Seeding dev user 1 house from $seedPath..."
$seedJson = Get-Content -Raw -LiteralPath $seedPath
Invoke-JsonRequest -Method "POST" -Uri $devHouseUrl -Body $seedJson | Out-Null

Write-Host "Dev house seeded."
Write-Host "Refresh: $normalizedBaseUrl"
