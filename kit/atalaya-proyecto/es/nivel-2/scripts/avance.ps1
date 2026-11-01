# Muestra el avance de los agentes que trabajan en una fase.
# Cada agente lleva una bitacora en docs\trazabilidad\<fase>\avance\<rol>.md:
#   primera linea:  Avance: NN % - que hace ahora
#   debajo:         - HH:MM . NN % . hito
#
# Uso:
#   .\scripts\avance.ps1                 todas las fases que tengan bitacoras
#   .\scripts\avance.ps1 F2-Ejemplo      una fase
#   .\scripts\avance.ps1 -Detalle        con las ultimas lineas de cada bitacora

param(
    [string]$Fase = '*',
    [switch]$Detalle
)

$ErrorActionPreference = 'Stop'
$raiz = Split-Path -Parent $PSScriptRoot
$patron = Join-Path $raiz "docs\trazabilidad\$Fase\avance\*.md"
$bitacoras = Get-ChildItem -Path $patron -ErrorAction SilentlyContinue | Sort-Object FullName

if (-not $bitacoras) {
    Write-Host 'No hay bitacoras de avance.'
    exit 0
}

foreach ($b in $bitacoras) {
    $lineas = Get-Content -LiteralPath $b.FullName -Encoding UTF8
    $cabecera = ($lineas | Where-Object { $_ -match '^Avance:' } | Select-Object -First 1)
    $porcentaje = 0
    if ($cabecera -match '(\d{1,3})\s*%') { $porcentaje = [Math]::Min(100, [int]$Matches[1]) }
    $llenos = [int][Math]::Round($porcentaje / 5)
    $barra = ('#' * $llenos) + ('.' * (20 - $llenos))
    $fase = Split-Path -Leaf (Split-Path -Parent (Split-Path -Parent $b.FullName))
    $hace = ''
    if ($cabecera -match '%\s*.\s*(.+)$') { $hace = $Matches[1] }
    $edad = [int]((Get-Date) - $b.LastWriteTime).TotalMinutes

    Write-Host ('{0,-16} {1,-22} [{2}] {3,3} %  {4}  (hace {5} min)' -f $fase, $b.BaseName, $barra, $porcentaje, $hace, $edad)
    if ($Detalle) {
        $lineas | Where-Object { $_ -match '^- ' } | Select-Object -Last 4 | ForEach-Object { Write-Host "    $_" }
    }
}
