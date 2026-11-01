<#
.SYNOPSIS
    Reinicia Atalaya en 127.0.0.1:7420: lo apaga, compila si se pide y lo vuelve a encender.
.DESCRIPTION
    1. Apaga el atalaya.exe de este repositorio que escucha en el puerto (solo ese: si otro programa ocupa el puerto, se
       detiene y lo dice, sin tocarlo).
    2. Con -Compilar corre scripts\compilar.ps1; con -Verificar, scripts\compilar.ps1 -Verificar. atalaya.exe bloquea su
       archivo mientras corre: por eso se apaga antes.
    3. Lo enciende en segundo plano, sin ventana, y espera a que /api/salud responda.
    Con -Apagar solo lo apaga.
.EXAMPLE
    .\scripts\reiniciar.ps1 -Verificar
#>
param(
    [switch]$Compilar,
    [switch]$Verificar,
    [switch]$Apagar,
    [int]$Puerto = 7420
)
$ErrorActionPreference = 'Stop'
$raiz = Split-Path -Parent $PSScriptRoot
$exe = Join-Path $raiz 'atalaya.exe'

# 1. Apagar el que escucha en el puerto, si es este atalaya.exe.
$conexiones = @(Get-NetTCPConnection -LocalPort $Puerto -State Listen -ErrorAction SilentlyContinue)
foreach ($c in $conexiones) {
    $proc = Get-Process -Id $c.OwningProcess -ErrorAction SilentlyContinue
    if (-not $proc) { continue }
    if ($proc.Path -and ([IO.Path]::GetFullPath($proc.Path) -ieq [IO.Path]::GetFullPath($exe))) {
        Write-Host "== apagar atalaya.exe (pid $($proc.Id)) en el $Puerto"
        Stop-Process -Id $proc.Id -Force
        $proc.WaitForExit(5000) | Out-Null
    } else {
        throw "El puerto $Puerto lo ocupa otro programa ($($proc.ProcessName), pid $($proc.Id), $($proc.Path)). No se toca."
    }
}
if ($Apagar) { Write-Host 'apagado'; exit 0 }

# 2. Compilar, si se pide.
if ($Verificar) { & (Join-Path $PSScriptRoot 'compilar.ps1') -Verificar }
elseif ($Compilar) { & (Join-Path $PSScriptRoot 'compilar.ps1') }

# 3. Encender y esperar a que responda.
if (-not (Test-Path -LiteralPath $exe)) { throw "No existe ${exe}: corre con -Compilar." }
$p = Start-Process -FilePath $exe -ArgumentList '-escucha', "127.0.0.1:$Puerto" -WorkingDirectory $raiz -WindowStyle Hidden -PassThru
for ($i = 0; $i -lt 30; $i++) {
    Start-Sleep -Milliseconds 500
    try {
        $s = Invoke-RestMethod "http://127.0.0.1:$Puerto/api/salud" -TimeoutSec 2
        Write-Host "== encendido: atalaya.exe $($s.version) (pid $($p.Id)) en http://127.0.0.1:$Puerto"
        exit 0
    } catch { }
    if ($p.HasExited) { throw "atalaya.exe terminó al arrancar (salida $($p.ExitCode))." }
}
throw "atalaya.exe no respondió en el $Puerto a los 15 s."
