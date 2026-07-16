param(
    [Parameter(ValueFromRemainingArguments = $true)]
    [string[]]$GoArgs
)

$ErrorActionPreference = 'Stop'

$tempRoot = Join-Path (Split-Path -Parent $PSScriptRoot) '.test-tmp'
if (Test-Path -LiteralPath $tempRoot) {
    Remove-Item -LiteralPath $tempRoot -Recurse -Force
}
New-Item -ItemType Directory -Path $tempRoot | Out-Null

$env:TMP = $tempRoot
$env:TEMP = $tempRoot
& go test @GoArgs
exit $LASTEXITCODE
