$ErrorActionPreference = "Stop"

$prompt = $env:TASK_PROMPT
if ([string]::IsNullOrWhiteSpace($prompt)) {
    throw "TASK_PROMPT is required"
}

$env:CLAUDE_CONFIG_DIR = Join-Path $HOME ".claude"
Remove-Item Env:ANTHROPIC_BASE_URL -ErrorAction SilentlyContinue
Remove-Item Env:ANTHROPIC_AUTH_TOKEN -ErrorAction SilentlyContinue

claude -p --dangerously-skip-permissions $prompt
exit $LASTEXITCODE
