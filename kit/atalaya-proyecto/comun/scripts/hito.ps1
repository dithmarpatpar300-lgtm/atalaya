<#
.SYNOPSIS
  Writes a milestone into the phase's AVANCE.md using the system clock time.
.DESCRIPTION
  A milestone's time comes from the clock, never from memory: an agent has no clock and, if it writes the time by hand,
  it estimates it. (Atalaya warns when the latest milestone is later than the time the file was saved.)
  The milestone goes above the others (newest first) or, if it is the first one, below the package table. It keeps the
  file's line endings. Windows PowerShell 5.1 or PowerShell 7.

  Usage:
    .\scripts\hito.ps1 'W3 done: global limiter with its tests'
    .\scripts\hito.ps1 'Round 2 delivered' -Fase F6
#>
param(
    [Parameter(Mandatory = $true, Position = 0)][string]$Texto,
    [string]$Fase = ''
)

$ErrorActionPreference = 'Stop'
$raiz = Split-Path -Parent $PSScriptRoot
$utf8 = New-Object System.Text.UTF8Encoding($false)
# ASCII only in this file: PowerShell 5.1 reads a .ps1 without BOM as ANSI.
$dot = [char]0x00B7
$base = Join-Path (Join-Path $raiz 'docs') 'trazabilidad'

# Without -Fase, the phase in progress: the highest F<n> that has an AVANCE.md.
if (-not $Fase) {
    $fases = Get-ChildItem -LiteralPath $base -Directory -ErrorAction SilentlyContinue |
        Where-Object { $_.Name -match '^F\d+$' -and (Test-Path -LiteralPath (Join-Path $_.FullName 'AVANCE.md')) } |
        Sort-Object { [int]$_.Name.Substring(1) }
    if (-not $fases) { throw 'there is no phase with an AVANCE.md in docs/trazabilidad' }
    $Fase = @($fases)[-1].Name
}
$archivo = Join-Path (Join-Path $base $Fase) 'AVANCE.md'
if (-not (Test-Path -LiteralPath $archivo)) { throw "$archivo does not exist" }

$Texto = ($Texto -replace '\s+', ' ').Trim()
if (-not $Texto) { throw 'the milestone has no text' }
$linea = "- $(Get-Date -Format 'HH:mm') $dot $Texto"

$original = [System.IO.File]::ReadAllText($archivo, $utf8)
$fin = if ($original.Contains("`r`n")) { "`r`n" } else { "`n" }
$lineas = New-Object System.Collections.Generic.List[string]
$lineas.AddRange([string[]]($original -split "`r?`n"))
if ($lineas.Count -gt 0 -and $lineas[$lineas.Count - 1] -eq '') { $lineas.RemoveAt($lineas.Count - 1) }

# The newest milestone already there (a line that starts with "- HH:MM" and the middle dot): the new one goes right above.
$primero = -1
for ($k = 0; $k -lt $lineas.Count; $k++) {
    if ($lineas[$k] -match "^-\s*\d{1,2}:\d{2}\s*$dot") { $primero = $k; break }
}
if ($primero -ge 0) {
    $lineas.Insert($primero, $linea)
} else {
    # None yet: after the last row of the table, with a blank line; with no table, at the end.
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
