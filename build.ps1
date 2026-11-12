# Gera o executavel de distribuicao em dist\ConversorStone1503.exe
# Uso:  .\build.ps1            (roda os testes e compila)
#       .\build.ps1 -SemTestes
param([switch]$SemTestes)

$ErrorActionPreference = 'Stop'
Set-Location $PSScriptRoot

if (-not (Get-Command go -ErrorAction SilentlyContinue)) {
    Write-Error 'Go nao encontrado. Instale em https://go.dev/dl/ (versao 1.24 ou superior).'
}

$env:CGO_ENABLED = '0'

if (-not $SemTestes) {
    go vet ./...
    if ($LASTEXITCODE) { exit $LASTEXITCODE }
    go test ./...
    if ($LASTEXITCODE) { exit $LASTEXITCODE }
}

$env:GOOS = 'windows'
$env:GOARCH = 'amd64'
New-Item -ItemType Directory -Force dist | Out-Null
go build -trimpath -ldflags='-s -w -H windowsgui' -o dist\ConversorStone1503.exe .
if ($LASTEXITCODE) { exit $LASTEXITCODE }

Get-Item dist\ConversorStone1503.exe | Select-Object FullName, Length, LastWriteTime
