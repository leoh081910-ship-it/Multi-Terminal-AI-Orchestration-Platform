#!/usr/bin/env sh
set -eu

: "${TASK_PROMPT:?TASK_PROMPT is required}"

model="${AIOP_GEMINI_MODEL:-${GEMINI_MODEL:-gemini-2.5-flash}}"
if [ "$model" = "gemini-3.1-pro" ]; then
  model="gemini-2.5-flash"
fi

exec gemini --skip-trust --yolo --model "$model" -p "$TASK_PROMPT" < /dev/null
