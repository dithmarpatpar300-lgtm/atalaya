<#
.SYNOPSIS
    Compila Atalaya: el frontend exportado, embebido en un solo atalaya.exe.
.DESCRIPTION
    0. Si falta frontend/node_modules (un repositorio recién clonado), npm ci.
    1. frontend: next build (exportación estática a frontend/out).
    2. Copia frontend/out a backend/internal/web/dist (solo LEEME.txt va al repositorio).
    3. backend: go build -> atalaya.exe en la raíz del proyecto (atalaya en Linux y macOS, con PowerShell 7).
    Con -Verificar corre antes go vet, go test, la comprobación de tipos y el lint.
#>
param([switch]$Verificar)
$ErrorActionPreference = 'Stop'
$raiz = Split-Path -Parent $PSScriptRoot
$front = Join-Path $raiz 'frontend'
$back = Join-Path $raiz 'backend'

function Paso([string]$nombre, [scriptblock]$orden) {
    Write-Host "== $nombre"
    & $orden
    if ($LASTEXITCODE -ne 0) { throw "$nombre fallo (salida $LASTEXITCODE)" }
}

# En un repositorio recién clonado, las dependencias del panel todavía no están.
if (-not (Test-Path -LiteralPath (Join-Path $front 'node_modules'))) {
    Push-Location $front
    try { Paso 'npm ci' { npm ci } } finally { Pop-Location }
}

if ($Verificar) {
    Push-Location $back
    try {
        Paso 'gofmt' { $f = gofmt -l .; if ($f) { Write-Host $f; $global:LASTEXITCODE = 1 } else { $global:LASTEXITCODE = 0 } }
        Paso 'go vet' { go vet ./... }
        Paso 'go test' { go test ./... }
    } finally { Pop-Location }
    Push-Location $front
    try {
        Paso 'tsc' { npx tsc --noEmit }
        Paso 'eslint' { npm run lint }
        Paso 'node --test' { npm test }
    } finally { Pop-Location }
}

Push-Location $front
try { Paso 'next build' { npm run build } } finally { Pop-Location }

$dist = Join-Path (Join-Path (Join-Path $back 'internal') 'web') 'dist'
Get-ChildItem -LiteralPath $dist -Force | Where-Object { $_.Name -ne 'LEEME.txt' } | Remove-Item -Recurse -Force
Copy-Item -Path (Join-Path (Join-Path $front 'out') '*') -Destination $dist -Recurse -Force

# atalaya.exe en Windows; atalaya en Linux y macOS (PowerShell 7). En Windows PowerShell 5.1 $IsLinux no existe. No se
# llama $nombre: dentro de Paso, ese nombre es el del paso y el binario salía como «go build».
$binario = if ($IsLinux -or $IsMacOS) { 'atalaya' } else { 'atalaya.exe' }
Push-Location $back
try { Paso 'go build' { go build -trimpath -ldflags '-s -w' -o (Join-Path $raiz $binario) ./cmd/atalaya } } finally { Pop-Location }
$exe = Get-Item (Join-Path $raiz $binario)
Write-Host ("listo: {0} ({1:N0} B)" -f $exe.FullName, $exe.Length)
