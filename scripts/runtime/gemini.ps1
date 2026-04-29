$ErrorActionPreference = "Stop"

$prompt = $env:TASK_PROMPT
if ([string]::IsNullOrWhiteSpace($prompt)) {
    throw "TASK_PROMPT is required"
}

& (Join-Path $PSScriptRoot "claude.ps1")
exit $LASTEXITCODE
