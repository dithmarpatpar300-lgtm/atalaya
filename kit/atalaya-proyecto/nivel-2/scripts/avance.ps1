# Shows the progress of the agents working on a phase.
# Every agent keeps a log in docs\trazabilidad\<phase>\avance\<role>.md:
#   first line:  Progress: NN % - what it is doing now   (the Spanish mark "Avance:" is accepted too)
#   below:       - HH:MM . NN % . milestone
#
# Usage:
#   .\scripts\avance.ps1                 every phase that has logs
#   .\scripts\avance.ps1 F2-Example      one phase
#   .\scripts\avance.ps1 -Detalle        with the last lines of each log

param(
    [string]$Fase = '*',
    [switch]$Detalle
)

$ErrorActionPreference = 'Stop'
$raiz = Split-Path -Parent $PSScriptRoot
$patron = Join-Path $raiz "docs\trazabilidad\$Fase\avance\*.md"
$bitacoras = Get-ChildItem -Path $patron -ErrorAction SilentlyContinue | Sort-Object FullName

if (-not $bitacoras) {
    Write-Host 'There are no progress logs.'
    exit 0
}

foreach ($b in $bitacoras) {
    $lineas = Get-Content -LiteralPath $b.FullName -Encoding UTF8
    $cabecera = ($lineas | Where-Object { $_ -match '^(Progress|Avance):' } | Select-Object -First 1)
    $porcentaje = 0
    if ($cabecera -match '(\d{1,3})\s*%') { $porcentaje = [Math]::Min(100, [int]$Matches[1]) }
    $llenos = [int][Math]::Round($porcentaje / 5)
    $barra = ('#' * $llenos) + ('.' * (20 - $llenos))
    $fase = Split-Path -Leaf (Split-Path -Parent (Split-Path -Parent $b.FullName))
    $hace = ''
    if ($cabecera -match '%\s*.\s*(.+)$') { $hace = $Matches[1] }
    $edad = [int]((Get-Date) - $b.LastWriteTime).TotalMinutes

    Write-Host ('{0,-16} {1,-22} [{2}] {3,3} %  {4}  ({5} min ago)' -f $fase, $b.BaseName, $barra, $porcentaje, $hace, $edad)
    if ($Detalle) {
        $lineas | Where-Object { $_ -match '^- ' } | Select-Object -Last 4 | ForEach-Object { Write-Host "    $_" }
    }
}
