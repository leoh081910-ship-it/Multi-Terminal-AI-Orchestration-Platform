#!/usr/bin/env sh
set -eu

: "${TASK_PROMPT:?TASK_PROMPT is required}"

source_claude_config_dir="${CLAUDE_CONFIG_DIR:-/root/.claude}"
runtime_claude_config_dir="${AIOP_CLAUDE_CONFIG_DIR:-/root/.aiop-claude-home}"
runtime_claude_base_url="${AIOP_CLAUDE_BASE_URL:-http://host.docker.internal:15721}"
claude_allowed_tools="${AIOP_CLAUDE_ALLOWED_TOOLS:-Read,Write,Edit,MultiEdit,Bash,Grep,Glob,LS}"

if [ -z "$runtime_claude_config_dir" ] || [ "$runtime_claude_config_dir" = "/" ] || [ "$runtime_claude_config_dir" = "$source_claude_config_dir" ]; then
  echo "refusing unsafe AIOP_CLAUDE_CONFIG_DIR: $runtime_claude_config_dir" >&2
  exit 1
fi

mkdir -p "$runtime_claude_config_dir"
find "$runtime_claude_config_dir" -mindepth 1 -maxdepth 1 -exec rm -rf {} +

config_file="$source_claude_config_dir/.claude.json"
if [ -f "$config_file" ]; then
  cp "$config_file" "$runtime_claude_config_dir/.claude.json"
fi

runtime_config_file="$runtime_claude_config_dir/.claude.json"
config_size=0
if [ -f "$runtime_config_file" ]; then
  config_size="$(wc -c < "$runtime_config_file" | tr -d ' ')"
fi
if [ "$config_size" -lt 1000 ] && [ -d "$source_claude_config_dir/backups" ]; then
  for backup in $(ls -t "$source_claude_config_dir"/backups/.claude.json.backup.* 2>/dev/null || true); do
    if [ "$(wc -c < "$backup" | tr -d ' ')" -gt 1000 ]; then
      cp "$backup" "$runtime_config_file"
      break
    fi
  done
fi

if [ -f "$source_claude_config_dir/settings.json" ] && command -v jq >/dev/null 2>&1; then
  AIOP_CLAUDE_BASE_URL="$runtime_claude_base_url" jq '
    .env = (.env // {}) |
    if (.env.ANTHROPIC_BASE_URL == null or .env.ANTHROPIC_BASE_URL == "" or .env.ANTHROPIC_BASE_URL == "http://127.0.0.1:15721" or .env.ANTHROPIC_BASE_URL == "http://localhost:15721") then
      .env.ANTHROPIC_BASE_URL = env.AIOP_CLAUDE_BASE_URL
    else
      .
    end |
    .env.CLAUDE_CODE_ENABLE_TELEMETRY = "0"
  ' "$source_claude_config_dir/settings.json" > "$runtime_claude_config_dir/settings.json"
else
  cat > "$runtime_claude_config_dir/settings.json" <<EOF
{"env":{"ANTHROPIC_BASE_URL":"$runtime_claude_base_url","CLAUDE_CODE_ENABLE_TELEMETRY":"0"}}
EOF
fi

if [ -f "$source_claude_config_dir/settings.local.json" ]; then
  cp "$source_claude_config_dir/settings.local.json" "$runtime_claude_config_dir/settings.local.json"
fi

export CLAUDE_CONFIG_DIR="$runtime_claude_config_dir"

if [ "$(id -u)" = "0" ]; then
  exec claude --bare -p --output-format text --allowedTools "$claude_allowed_tools" -- "$TASK_PROMPT" < /dev/null
fi

exec claude --bare -p --output-format text --dangerously-skip-permissions --allowedTools "$claude_allowed_tools" -- "$TASK_PROMPT" < /dev/null
