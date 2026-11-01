<#
.SYNOPSIS
  Anota un hito en el AVANCE.md de la fase con la hora del reloj del sistema.
.DESCRIPTION
  La hora de un hito sale del reloj, nunca de memoria: un agente no tiene reloj y, si la escribe a mano, la estima.
  (Atalaya avisa cuando el ultimo hito es posterior a la hora en que se guardo el archivo.)
  El hito va encima de los demas (el mas nuevo primero) o, si es el primero, debajo de la tabla de paquetes. Respeta
  los finales de linea del archivo. Windows PowerShell 5.1 o PowerShell 7.

  Uso:
    .\scripts\hito.ps1 'W3 hecho: limitador global con sus pruebas'
    .\scripts\hito.ps1 'Ronda 2 entregada' -Fase F6
#>
param(
    [Parameter(Mandatory = $true, Position = 0)][string]$Texto,
    [string]$Fase = ''
)

$ErrorActionPreference = 'Stop'
$raiz = Split-Path -Parent $PSScriptRoot
$utf8 = New-Object System.Text.UTF8Encoding($false)
# Solo ASCII en este archivo: PowerShell 5.1 lee un .ps1 sin BOM como ANSI.
$dot = [char]0x00B7
$base = Join-Path (Join-Path $raiz 'docs') 'trazabilidad'

# Sin -Fase, la fase en curso: la F<n> mas alta que tiene AVANCE.md.
if (-not $Fase) {
    $fases = Get-ChildItem -LiteralPath $base -Directory -ErrorAction SilentlyContinue |
        Where-Object { $_.Name -match '^F\d+$' -and (Test-Path -LiteralPath (Join-Path $_.FullName 'AVANCE.md')) } |
        Sort-Object { [int]$_.Name.Substring(1) }
    if (-not $fases) { throw 'no hay ninguna fase con AVANCE.md en docs/trazabilidad' }
    $Fase = @($fases)[-1].Name
}
$archivo = Join-Path (Join-Path $base $Fase) 'AVANCE.md'
if (-not (Test-Path -LiteralPath $archivo)) { throw "no existe $archivo" }

$Texto = ($Texto -replace '\s+', ' ').Trim()
if (-not $Texto) { throw 'el hito no tiene texto' }
$linea = "- $(Get-Date -Format 'HH:mm') $dot $Texto"

$original = [System.IO.File]::ReadAllText($archivo, $utf8)
$fin = if ($original.Contains("`r`n")) { "`r`n" } else { "`n" }
$lineas = New-Object System.Collections.Generic.List[string]
$lineas.AddRange([string[]]($original -split "`r?`n"))
if ($lineas.Count -gt 0 -and $lineas[$lineas.Count - 1] -eq '') { $lineas.RemoveAt($lineas.Count - 1) }

# El hito mas nuevo que ya hay (una linea que empieza en "- HH:MM" y el punto medio): el nuevo va justo encima.
$primero = -1
for ($k = 0; $k -lt $lineas.Count; $k++) {
    if ($lineas[$k] -match "^-\s*\d{1,2}:\d{2}\s*$dot") { $primero = $k; break }
}
if ($primero -ge 0) {
    $lineas.Insert($primero, $linea)
} else {
    # Ninguno todavia: despues de la ultima fila de la tabla, con una linea en blanco; sin tabla, al final.
    $ultimaFila = -1
    for ($k = 0; $k -lt $lineas.Count; $k++) { if ($lineas[$k].TrimStart().StartsWith('|')) { $ultimaFila = $k } }
    if ($ultimaFila -ge 0) {
        $lineas.Insert($ultimaFila + 1, '')
        $lineas.Insert($ultimaFila + 2, $linea)
    } else {
        $lineas.Add('')
        $lineas.Add($linea)
    }
}
[System.IO.File]::WriteAllText($archivo, ($lineas -join $fin) + $fin, $utf8)
"$Fase/AVANCE.md: $linea"
