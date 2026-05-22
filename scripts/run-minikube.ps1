param(
  [string]$Namespace = "homeplan",
  [string]$Release = "homeplan",
  [string]$KubeContext = "minikube",
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
  Invoke-Step "Checking Kubernetes API connectivity for context $KubeContext..." { kubectl --context $KubeContext cluster-info }
}

function Assert-MinikubeContext {
  $contexts = & kubectl config get-contexts -o name
  if ($LASTEXITCODE -ne 0) {
    throw "Could not read kube contexts."
  }
  if ($contexts -notcontains $KubeContext) {
    throw "Kubernetes context '$KubeContext' was not found. Refusing to deploy locally."
  }

  $currentContext = (& kubectl config current-context).Trim()
  Write-Host "Using Kubernetes context $KubeContext (current context: $currentContext)."
  if ($KubeContext -ne "minikube") {
    throw "Local deploys must target the minikube context. Refusing to use '$KubeContext'."
  }
}

function Ensure-Namespace {
  Write-Host "Checking namespace $Namespace..."
  $previousNativePreference = $PSNativeCommandUseErrorActionPreference
  if ($PSVersionTable.PSVersion.Major -ge 7) {
    $PSNativeCommandUseErrorActionPreference = $false
  }
  kubectl --context $KubeContext get namespace $Namespace *> $null
  $namespaceExists = $LASTEXITCODE -eq 0
  if ($PSVersionTable.PSVersion.Major -ge 7) {
    $PSNativeCommandUseErrorActionPreference = $previousNativePreference
  }

  if (-not $namespaceExists) {
    Invoke-Step "Creating namespace $Namespace..." { kubectl --context $KubeContext create namespace $Namespace }
  }
}

function Wait-Rollout {
  param([string]$Name)
  Invoke-Step "Waiting for deployment/$Name..." {
    kubectl --context $KubeContext -n $Namespace rollout status "deployment/$Name" --timeout=180s
  }
}

function Get-PortListener {
  param([int]$LocalPort)
  Get-NetTCPConnection -LocalPort $LocalPort -State Listen -ErrorAction SilentlyContinue |
    Select-Object -First 1
}

function Ensure-PortForward {
  $listener = Get-PortListener -LocalPort $Port
  if ($listener) {
    $process = Get-Process -Id $listener.OwningProcess -ErrorAction SilentlyContinue
    if ($process -and $process.ProcessName -eq "kubectl") {
      Write-Host "Restarting existing kubectl port-forward on localhost:$Port..."
      Stop-Process -Id $listener.OwningProcess -Force
      Start-Sleep -Milliseconds 500
    } else {
      $processName = if ($process) { $process.ProcessName } else { "process $($listener.OwningProcess)" }
      throw "Port $Port is already in use by $processName. Stop that process or choose another -Port."
    }
  }

  Get-CimInstance Win32_Process |
    Where-Object { $_.CommandLine -like "*kubectl*port-forward*svc/homeplan-web*$Port`:80*" } |
    ForEach-Object { Stop-Process -Id $_.ProcessId -Force -ErrorAction SilentlyContinue }

  $logDir = Join-Path $env:TEMP "homeplan-port-forward"
  New-Item -ItemType Directory -Force -Path $logDir | Out-Null
  $stdoutLog = Join-Path $logDir "stdout.log"
  $stderrLog = Join-Path $logDir "stderr.log"
  Remove-Item -LiteralPath $stdoutLog, $stderrLog -ErrorAction SilentlyContinue

  Write-Host "Starting kubectl port-forward on localhost:$Port..."
  Start-Process -FilePath "kubectl" `
    -ArgumentList @("--context", $KubeContext, "-n", $Namespace, "port-forward", "svc/homeplan-web", "$Port`:80") `
    -WindowStyle Hidden `
    -RedirectStandardOutput $stdoutLog `
    -RedirectStandardError $stderrLog

  $deadline = (Get-Date).AddSeconds(15)
  do {
    Start-Sleep -Milliseconds 500
    if (Get-PortListener -LocalPort $Port) {
      try {
        $response = Invoke-WebRequest -UseBasicParsing -Uri "http://localhost:$Port/healthz" -TimeoutSec 3
        if ($response.StatusCode -eq 200) {
          return
        }
      } catch {
        # The listener can appear before the pod connection is ready.
      }
    }
  } while ((Get-Date) -lt $deadline)

  $stdout = if (Test-Path $stdoutLog) { Get-Content -Raw -LiteralPath $stdoutLog } else { "" }
  $stderr = if (Test-Path $stderrLog) { Get-Content -Raw -LiteralPath $stderrLog } else { "" }
  throw "Port-forward did not start listening on localhost:$Port. Logs: $logDir`n$stdout`n$stderr"
}

try {
  Require-Command minikube
  Require-Command kubectl
  Require-Command helm
  Assert-MinikubeContext

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
    helm --kube-context $KubeContext upgrade --install $Release deploy/helm/homeplan --namespace $Namespace -f deploy/helm/homeplan/values-local.yaml --set images.api=$apiImage --set images.web=$webImage --set rolloutToken=$rolloutToken
  }

  Write-Host "Waiting for workloads..."
  Wait-Rollout "homeplan-postgres"
  Invoke-Step "Waiting for migration job..." {
    kubectl --context $KubeContext -n $Namespace wait --for=condition=complete "job/homeplan-migrate" --timeout=180s
  }
  Wait-Rollout "homeplan-api"
  Wait-Rollout "homeplan-web"

  Ensure-PortForward

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
