<#
.SYNOPSIS
  Builds the public copy of Atalaya in a new folder, ready to become its own repository.

.DESCRIPTION
  Windows PowerShell 5.1 or PowerShell 7. The working repository keeps its history and its process documents
  (docs/trazabilidad, the plan, the audits); the public copy takes only what serves other people:
  - every tracked file of backend/, frontend/, kit/, scripts/ and .github/ (the CI);
  - README.md, README.es.md, CHANGELOG.md, LICENSE, proyectos.ejemplo.json, .gitignore;
  - from docs/, only INSTALL.md and INSTALAR.md;
  - a generic .mcp.json (the local one names this machine's project).
  Then it checks the copy:
  - the version in backend/internal/servidor/servidor.go is the first one in CHANGELOG.md;
  - kit/atalaya-proyecto/nivel-2/scripts/codex.ps1 is the same file as scripts/codex.ps1;
  - no file contains a term from .publicar-excluir.txt (one term per line, case-insensitive). That file stays out
    of git on purpose: it lists the private names that must never reach the public copy.
  Two modes, by what -Destino is:
  - A new or empty folder: the first copy. With -Commit it runs git init and makes one commit, tagged
    v<version>, with -Autor and -Correo (use the noreply address GitHub gives your account).
  - An existing clone of the public repository (it has .git): an update, the way dev reaches main. The clone
    must be clean; its tracked files are replaced by the new copy (what dev deleted disappears there too), and
    with -Commit it commits with -Mensaje, using the clone's own git identity, and tags v<version> if that tag
    does not exist yet.
  It pushes only with -Subir (main and its tags, to the clone's origin). It never creates a remote.

.EXAMPLE
  .\scripts\publicar.ps1 -Destino C:\Publicar\atalaya
  .\scripts\publicar.ps1 -Destino C:\Publicar\atalaya -Commit -Autor 'usuario' -Correo '123+usuario@users.noreply.github.com'
  .\scripts\publicar.ps1 -Destino ..\atalaya -Commit -Mensaje 'Arregla el selector de fecha' -Subir
#>
[CmdletBinding()]
param(
    [Parameter(Mandatory = $true)][string]$Destino,
    [switch]$Commit,
    [string]$Autor,
    [string]$Correo,
    [string]$Mensaje,
    [switch]$Subir
)

$ErrorActionPreference = 'Stop'
$raiz = Split-Path -Parent $PSScriptRoot
$utf8 = New-Object System.Text.UTF8Encoding($false)

function Fallar([string]$mensaje) { Write-Host "NO: $mensaje"; exit 1 }

$destinoAbs = [System.IO.Path]::GetFullPath($Destino)
if ($destinoAbs.StartsWith($raiz, [System.StringComparison]::OrdinalIgnoreCase)) { Fallar "the copy must be outside the repository ($raiz)" }
$actualizar = Test-Path -LiteralPath (Join-Path $destinoAbs '.git')
if (-not $actualizar -and (Test-Path -LiteralPath $destinoAbs) -and (Get-ChildItem -LiteralPath $destinoAbs -Force | Select-Object -First 1)) {
    Fallar "$destinoAbs already exists, is not empty and is not a git clone"
}
if ($Commit -and -not $actualizar -and (-not $Autor -or -not $Correo)) { Fallar '-Commit on a new copy needs -Autor and -Correo' }
if ($Commit -and $actualizar -and -not $Mensaje) { Fallar '-Commit on an existing clone needs -Mensaje' }
if ($Subir -and -not ($Commit -and $actualizar)) { Fallar '-Subir goes with -Commit on an existing clone' }
$pendiente = & git -C $raiz status --porcelain
if ($pendiente) { Fallar "the working tree has uncommitted changes; commit them first:`n$($pendiente -join "`n")" }
if ($actualizar) {
    $sucio = & git -C $destinoAbs status --porcelain
    if ($sucio) { Fallar "the public clone has uncommitted changes:`n$($sucio -join "`n")" }
    # What dev deleted must disappear from main too: start from no tracked files.
    foreach ($viejo in (& git -C $destinoAbs ls-files)) { Remove-Item -LiteralPath (Join-Path $destinoAbs $viejo) -Force }
}

# 1. The public files.
$sueltos = @('README.md', 'README.es.md', 'CHANGELOG.md', 'LICENSE', 'proyectos.ejemplo.json', '.gitignore',
    'docs/INSTALL.md', 'docs/INSTALAR.md')
$archivos = & git -C $raiz ls-files | Where-Object {
    $_ -match '^(backend|frontend|kit|scripts|\.github)/' -or $sueltos -contains $_
}
foreach ($a in $sueltos) { if ($archivos -notcontains $a) { Fallar "$a is not tracked" } }
foreach ($rel in $archivos) {
    $origen = Join-Path $raiz $rel
    $copia = Join-Path $destinoAbs $rel
    New-Item -ItemType Directory -Force -Path (Split-Path -Parent $copia) | Out-Null
    Copy-Item -LiteralPath $origen -Destination $copia
}
$mcp = "{`n  `"mcpServers`": {`n    `"atalaya`": {`n      `"type`": `"http`",`n      `"url`": `"http://127.0.0.1:7420/mcp?proyecto=atalaya`"`n    }`n  }`n}`n"
[System.IO.File]::WriteAllText((Join-Path $destinoAbs '.mcp.json'), $mcp, $utf8)
Write-Host "== $($archivos.Count + 1) files copied to $destinoAbs"

# 2. The version.
$servidor = [System.IO.File]::ReadAllText((Join-Path $raiz 'backend\internal\servidor\servidor.go'), $utf8)
$version = [regex]::Match($servidor, 'const Version = "([^"]+)"').Groups[1].Value
$cambios = [System.IO.File]::ReadAllText((Join-Path $raiz 'CHANGELOG.md'), $utf8)
$ultima = [regex]::Match($cambios, '(?m)^## (\S+)').Groups[1].Value
if (-not $version -or $version -ne $ultima) { Fallar "servidor.go says $version and CHANGELOG.md starts with $ultima" }
Write-Host "== version $version"

# 3. The kit carries the same codex.ps1.
$h1 = (Get-FileHash -LiteralPath (Join-Path $raiz 'scripts\codex.ps1')).Hash
$h2 = (Get-FileHash -LiteralPath (Join-Path $raiz 'kit\atalaya-proyecto\nivel-2\scripts\codex.ps1')).Hash
if ($h1 -ne $h2) { Fallar 'kit/atalaya-proyecto/nivel-2/scripts/codex.ps1 differs from scripts/codex.ps1: copy it again' }

# 4. Nothing private.
$excluir = Join-Path $raiz '.publicar-excluir.txt'
if (Test-Path -LiteralPath $excluir) {
    $terminos = [System.IO.File]::ReadAllLines($excluir, $utf8) | ForEach-Object { $_.Trim() } | Where-Object { $_ -and -not $_.StartsWith('#') }
    $hallados = @()
    # Only what this run copied: a clone also has .git (with its identity) and whatever was built there.
    foreach ($rel in @($archivos) + '.mcp.json') {
        $texto = [System.IO.File]::ReadAllText((Join-Path $destinoAbs $rel), $utf8)
        foreach ($t in $terminos) {
            if ($texto.IndexOf($t, [System.StringComparison]::OrdinalIgnoreCase) -ge 0) { $hallados += "${rel}: $t" }
        }
    }
    if ($hallados) { Fallar "private terms in the copy:`n$($hallados -join "`n")" }
    Write-Host "== no private terms ($($terminos.Count) checked)"
} else {
    Write-Host '== .publicar-excluir.txt not found: the private-terms check was skipped'
}

# 5. The commit: a new repository with one commit, or one more commit on the public clone.
if ($Commit -and -not $actualizar) {
    & git -C $destinoAbs init -q -b main
    & git -C $destinoAbs add -A
    & git -C $destinoAbs -c "user.name=$Autor" -c "user.email=$Correo" commit -q -m "Atalaya $version"
    & git -C $destinoAbs -c "user.name=$Autor" -c "user.email=$Correo" tag -a "v$version" -m "Atalaya $version"
    Write-Host "== committed and tagged v$version in $destinoAbs (nothing pushed)"
}
if ($Commit -and $actualizar) {
    & git -C $destinoAbs add -A
    if (-not (& git -C $destinoAbs status --porcelain)) { Write-Host '== nothing changed: no commit'; exit 0 }
    & git -C $destinoAbs commit -q -m $Mensaje
    if ($LASTEXITCODE -ne 0) { Fallar 'git commit failed in the public clone' }
    $etiqueta = ''
    if (-not (& git -C $destinoAbs tag -l "v$version")) {
        & git -C $destinoAbs tag -a "v$version" -m "Atalaya $version"
        $etiqueta = " and tagged v$version"
    }
    Write-Host "== committed$etiqueta in $destinoAbs"
    if ($Subir) {
        & git -C $destinoAbs push -q origin main --follow-tags
        if ($LASTEXITCODE -ne 0) { Fallar 'git push failed' }
        Write-Host '== pushed to origin main'
    } else {
        Write-Host '== nothing pushed (use -Subir)'
    }
}
