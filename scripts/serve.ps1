$ErrorActionPreference = "Stop"

$imageName = "homeplan-local"
$containerName = "homeplan-local"
$hostPort = "8080"

podman build -t $imageName -f Containerfile .

$existing = podman ps -a --format "{{.Names}}" | Where-Object { $_ -eq $containerName }
if ($existing) {
  podman rm -f $containerName | Out-Null
}

podman run --rm --name $containerName -p "${hostPort}:80" $imageName
