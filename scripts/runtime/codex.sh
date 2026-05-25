#!/usr/bin/env sh
set -eu

: "${TASK_PROMPT:?TASK_PROMPT is required}"

source_codex_home="${CODEX_HOME:-/root/.codex}"
runtime_codex_home="${AIOP_CODEX_HOME:-/root/.aiop-codex-home}"

if [ -z "$runtime_codex_home" ] || [ "$runtime_codex_home" = "/" ] || [ "$runtime_codex_home" = "$source_codex_home" ]; then
  echo "refusing unsafe AIOP_CODEX_HOME: $runtime_codex_home" >&2
  exit 1
fi

mkdir -p "$runtime_codex_home"

# Keep the scheduler runtime small. Linking the whole user CODEX_HOME pulls in
# malformed local skills/plugins and can make non-interactive tasks spend minutes
# loading unrelated tool metadata before doing any work.
find "$runtime_codex_home" -mindepth 1 -maxdepth 1 \
  ! -name auth.json \
  ! -name installation_id \
  ! -name version.json \
  -exec rm -rf {} +

for name in auth.json installation_id version.json; do
  if [ -e "$source_codex_home/$name" ]; then
    ln -sfn "$source_codex_home/$name" "$runtime_codex_home/$name"
  fi
done

cat > "$runtime_codex_home/config.toml" <<'EOF'
model_provider = "custom"
model = "gpt-5.5"
model_reasoning_effort = "xhigh"
disable_response_storage = true

[model_providers.custom]
name = "custom"
wire_api = "responses"
requires_openai_auth = true
base_url = "http://host.docker.internal:15721/v1"

[projects."/workspace/default"]
trust_level = "trusted"
EOF

if [ -f "$source_codex_home/config.toml" ]; then
  source_model="$(grep -E '^model = ' "$source_codex_home/config.toml" | head -n 1 || true)"
  if [ -n "$source_model" ]; then
    sed -i "s#^model = .*#$source_model#" "$runtime_codex_home/config.toml"
  fi
fi

export CODEX_HOME="$runtime_codex_home"

exec codex exec \
  --dangerously-bypass-approvals-and-sandbox \
  --color never \
  -C "$(pwd)" \
  "$TASK_PROMPT" \
  < /dev/null
