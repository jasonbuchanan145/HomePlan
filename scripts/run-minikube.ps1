param(
  [string]$Namespace = "homeplan",
  [string]$Release = "homeplan",
  [switch]$SkipBuild,
  [int]$Port = 8080
)

$ErrorActionPreference = "Stop"

$apiImage = "localhost/homeplan-api:dev"
$webImage = "localhost/homeplan-web:dev"

function Require-Command {
  param([string]$Name)
  if (-not (Get-Command $Name -ErrorAction SilentlyContinue)) {
    throw "$Name is required but was not found on PATH."
  }
}

function Wait-Rollout {
  param([string]$Name)
  kubectl -n $Namespace rollout status "deployment/$Name" --timeout=180s
}

Require-Command minikube
Require-Command kubectl
Require-Command helm

$repoRoot = Resolve-Path (Join-Path $PSScriptRoot "..")
Set-Location $repoRoot

$hostStatus = ""
try {
  $hostStatus = minikube status --format "{{.Host}}"
} catch {
  $hostStatus = "Stopped"
}

if ($hostStatus -ne "Running") {
  Write-Host "Starting minikube..."
  minikube start
}

if (-not $SkipBuild) {
  Write-Host "Building API image inside minikube as $apiImage..."
  minikube image build -t $apiImage -f api/Containerfile .

  Write-Host "Building web image inside minikube as $webImage..."
  minikube image build -t $webImage -f web/Containerfile .
}

kubectl create namespace $Namespace --dry-run=client -o yaml | kubectl apply -f -

Write-Host "Deploying HomePlan with Helm..."
helm upgrade --install $Release deploy/helm/homeplan --namespace $Namespace --set images.api=$apiImage --set images.web=$webImage

Write-Host "Waiting for workloads..."
Wait-Rollout "homeplan-postgres"
kubectl -n $Namespace wait --for=condition=complete "job/homeplan-migrate" --timeout=180s
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
