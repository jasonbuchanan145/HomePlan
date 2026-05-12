param(
  [string]$Namespace = "homeplan",
  [string]$Release = "homeplan",
  [switch]$SkipBuild,
  [int]$Port = 8080
)

$ErrorActionPreference = "Stop"
if ($PSVersionTable.PSVersion.Major -ge 7) {
  $PSNativeCommandUseErrorActionPreference = $true
}

$apiImage = "localhost/homeplan-api:dev"
$webImage = "localhost/homeplan-web:dev"
$rolloutToken = [DateTimeOffset]::UtcNow.ToUnixTimeSeconds().ToString()

function Require-Command {
  param([string]$Name)
  if (-not (Get-Command $Name -ErrorAction SilentlyContinue)) {
    throw "$Name is required but was not found on PATH."
  }
}

function Invoke-Step {
  param(
    [string]$Label,
    [scriptblock]$Command
  )

  Write-Host $Label
  try {
    & $Command
  } catch {
    throw "$Label failed. $($_.Exception.Message)"
  }
  if ($LASTEXITCODE -ne 0) {
    throw "$Label failed with exit code $LASTEXITCODE."
  }
}

function Get-MinikubeHostStatus {
  $status = & minikube status --format "{{.Host}}" 2>$null
  if ($LASTEXITCODE -ne 0 -or [string]::IsNullOrWhiteSpace($status)) {
    return "Stopped"
  }
  return $status.Trim()
}

function Assert-ClusterReady {
  Invoke-Step "Checking minikube status..." { minikube status }
  Invoke-Step "Checking Kubernetes API connectivity..." { kubectl cluster-info }
}

function Ensure-Namespace {
  Write-Host "Checking namespace $Namespace..."
  $previousNativePreference = $PSNativeCommandUseErrorActionPreference
  if ($PSVersionTable.PSVersion.Major -ge 7) {
    $PSNativeCommandUseErrorActionPreference = $false
  }
  kubectl get namespace $Namespace *> $null
  $namespaceExists = $LASTEXITCODE -eq 0
  if ($PSVersionTable.PSVersion.Major -ge 7) {
    $PSNativeCommandUseErrorActionPreference = $previousNativePreference
  }

  if (-not $namespaceExists) {
    Invoke-Step "Creating namespace $Namespace..." { kubectl create namespace $Namespace }
  }
}

function Wait-Rollout {
  param([string]$Name)
  Invoke-Step "Waiting for deployment/$Name..." {
    kubectl -n $Namespace rollout status "deployment/$Name" --timeout=180s
  }
}

try {
  Require-Command minikube
  Require-Command kubectl
  Require-Command helm

  $repoRoot = Resolve-Path (Join-Path $PSScriptRoot "..")
  Set-Location $repoRoot

  $hostStatus = Get-MinikubeHostStatus
  if ($hostStatus -ne "Running") {
    Invoke-Step "Starting minikube..." { minikube start }
  }

  Assert-ClusterReady

  if (-not $SkipBuild) {
    Invoke-Step "Building API image inside minikube as $apiImage..." {
      minikube image build -t $apiImage -f api/Containerfile .
    }

    Invoke-Step "Building web image inside minikube as $webImage..." {
      minikube image build -t $webImage -f web/Containerfile .
    }
  }

  Ensure-Namespace

  Invoke-Step "Deploying HomePlan with Helm..." {
    helm upgrade --install $Release deploy/helm/homeplan --namespace $Namespace --set images.api=$apiImage --set images.web=$webImage --set rolloutToken=$rolloutToken
  }

  Write-Host "Waiting for workloads..."
  Wait-Rollout "homeplan-postgres"
  Invoke-Step "Waiting for migration job..." {
    kubectl -n $Namespace wait --for=condition=complete "job/homeplan-migrate" --timeout=180s
  }
  Wait-Rollout "homeplan-api"
  Wait-Rollout "homeplan-web"

  $existingPortForward = Get-CimInstance Win32_Process |
    Where-Object { $_.CommandLine -like "*kubectl*port-forward*svc/homeplan-web*$Port`:80*" }

  if (-not $existingPortForward) {
    Write-Host "Starting kubectl port-forward on localhost:$Port..."
    Start-Process -FilePath "kubectl" -ArgumentList @("-n", $Namespace, "port-forward", "svc/homeplan-web", "$Port`:80") -WindowStyle Hidden
  }

  Write-Host ""
  Write-Host "HomePlan is ready:"
  Write-Host "  Frontend:   http://localhost:$Port"
  Write-Host "  API health: http://localhost:$Port/healthz"
  Write-Host ""
  Write-Host "The app starts empty in dev mode. Seed user 1 with:"
  Write-Host "  .\scripts\seed-dev-house.ps1 -BaseUrl http://localhost:$Port"
} catch {
  Write-Error $_.Exception.Message
  Write-Host ""
  Write-Host "HomePlan did not start. Fix the error above and rerun scripts\run-minikube.ps1."
  exit 1
}
