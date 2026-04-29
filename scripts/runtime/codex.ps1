$ErrorActionPreference = "Stop"

$prompt = $env:TASK_PROMPT
if ([string]::IsNullOrWhiteSpace($prompt)) {
    throw "TASK_PROMPT is required"
}

if ([string]::IsNullOrWhiteSpace($env:CODEX_HOME)) {
    $env:CODEX_HOME = "E:\04-Claude\Runtime\.codex"
}

codex exec --dangerously-bypass-approvals-and-sandbox --color never $prompt
exit $LASTEXITCODE
