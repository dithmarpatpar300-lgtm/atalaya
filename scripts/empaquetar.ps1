<#
.SYNOPSIS
  Builds the release zips of Atalaya: one per system, ready to download and run.

.DESCRIPTION
  Windows PowerShell 5.1 or PowerShell 7. Run scripts\compilar.ps1 first: the panel must already be built and
  copied into backend/internal/web/dist, because every binary embeds it. Then, for each target, it cross-compiles
  with GOOS/GOARCH (pure Go, no cgo: modernc.org/sqlite needs no C compiler) and zips the binary with README.md,
  README.es.md, LICENSE, CHANGELOG.md, proyectos.ejemplo.json, the install guides and the atalaya-proyecto kit.
  The zips go to dist/ (outside git) as atalaya-<version>-<os>-<arch>.zip.

.EXAMPLE
  .\scripts\compilar.ps1
  .\scripts\empaquetar.ps1
#>
[CmdletBinding()]
param(
    [string[]]$Destinos = @('windows/amd64', 'linux/amd64', 'darwin/arm64')
)

$ErrorActionPreference = 'Stop'
$raiz = Split-Path -Parent $PSScriptRoot
$back = Join-Path $raiz 'backend'
$utf8 = New-Object System.Text.UTF8Encoding($false)

$servidor = [System.IO.File]::ReadAllText((Join-Path $back 'internal\servidor\servidor.go'), $utf8)
$version = [regex]::Match($servidor, 'const Version = "([^"]+)"').Groups[1].Value
if (-not $version) { throw 'no version found in backend/internal/servidor/servidor.go' }
if (-not (Test-Path -LiteralPath (Join-Path $back 'internal\web\dist\index.html'))) {
    throw 'the panel is not built: run .\scripts\compilar.ps1 first'
}

$dist = Join-Path $raiz 'dist'
New-Item -ItemType Directory -Force -Path $dist | Out-Null
$acompanan = @('README.md', 'README.es.md', 'LICENSE', 'CHANGELOG.md', 'proyectos.ejemplo.json', 'docs\INSTALL.md', 'docs\INSTALAR.md')

$antes = @{ GOOS = $env:GOOS; GOARCH = $env:GOARCH; CGO_ENABLED = $env:CGO_ENABLED }
try {
    foreach ($d in $Destinos) {
        $os, $arch = $d -split '/'
        $nombre = "atalaya-$version-$os-$arch"
        $carpeta = Join-Path $dist $nombre
        if (Test-Path -LiteralPath $carpeta) { Remove-Item -LiteralPath $carpeta -Recurse -Force }
        New-Item -ItemType Directory -Force -Path $carpeta | Out-Null
        $exe = if ($os -eq 'windows') { 'atalaya.exe' } else { 'atalaya' }

        $env:GOOS = $os; $env:GOARCH = $arch; $env:CGO_ENABLED = '0'
        Push-Location $back
        try {
            & go build -trimpath -ldflags '-s -w' -o (Join-Path $carpeta $exe) ./cmd/atalaya
            if ($LASTEXITCODE -ne 0) { throw "go build failed for $d" }
        } finally { Pop-Location }

        foreach ($a in $acompanan) {
            $copia = Join-Path $carpeta $a
            New-Item -ItemType Directory -Force -Path (Split-Path -Parent $copia) | Out-Null
            Copy-Item -LiteralPath (Join-Path $raiz $a) -Destination $copia
        }
        Copy-Item -Recurse -LiteralPath (Join-Path $raiz 'kit\atalaya-proyecto') -Destination (Join-Path $carpeta 'kit\atalaya-proyecto')

        $zip = Join-Path $dist "$nombre.zip"
        if (Test-Path -LiteralPath $zip) { Remove-Item -LiteralPath $zip -Force }
        Compress-Archive -Path (Join-Path $carpeta '*') -DestinationPath $zip
        Remove-Item -LiteralPath $carpeta -Recurse -Force
        $mb = [math]::Round((Get-Item -LiteralPath $zip).Length / 1MB, 1)
        Write-Host "== $nombre.zip ($mb MB)"
    }
} finally {
    $env:GOOS = $antes.GOOS; $env:GOARCH = $antes.GOARCH; $env:CGO_ENABLED = $antes.CGO_ENABLED
}
