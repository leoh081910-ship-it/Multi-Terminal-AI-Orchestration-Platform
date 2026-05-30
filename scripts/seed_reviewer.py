"""Idempotent Reviewer Runner seed — inserts and verifies the Reviewer agent.

Usage:
    python seed_reviewer.py

Safe to run multiple times. Verifies the agent is registered after insert.
"""

import json
import sqlite3
import sys
from pathlib import Path

DB_PATH = Path(r"E:\vibe coding\Projects\多终端 AI 编排平台\ai-orchestration.db")
SQL_PATH = Path(__file__).parent / "seed_reviewer.sql"

EXPECTED_CONFIG = {
    "endpoint": "http://127.0.0.1:8765/api/runner/review",
    "method": "POST",
    "timeout_ms": 600000,
    "output_path": "output",
}


def main() -> int:
    if not DB_PATH.exists():
        print(f"[FAIL] Database not found: {DB_PATH}")
        return 1

    sql = SQL_PATH.read_text(encoding="utf-8")
    con = sqlite3.connect(str(DB_PATH))

    try:
        con.executescript(sql)
        con.commit()
    except Exception as e:
        print(f"[FAIL] SQL execution failed: {e}")
        con.close()
        return 1

    # Verify
    cur = con.execute(
        "SELECT id, runner_type, runner_config FROM agents WHERE id = ?",
        ("Reviewer",),
    )
    row = cur.fetchone()
    con.close()

    if not row:
        print("[FAIL] Reviewer not found after seed")
        return 1

    agent_id, runner_type, runner_config_raw = row
    config = json.loads(runner_config_raw) if runner_config_raw else {}

    print(f"[OK] Reviewer agent registered:")
    print(f"  id:           {agent_id}")
    print(f"  runner_type:  {runner_type}")
    print(f"  endpoint:     {config.get('endpoint', '?')}")
    print(f"  method:       {config.get('method', '?')}")
    print(f"  timeout_ms:   {config.get('timeout_ms', '?')}")
    print(f"  output_path:  {config.get('output_path', '?')}")

    # Validate config matches expected
    for key, expected in EXPECTED_CONFIG.items():
        actual = config.get(key)
        if actual != expected:
            print(f"  [WARN] {key}: expected={expected}, actual={actual}")

    print()
    print("NOTE: Restart Go server to load Reviewer into runnerRegistry.")
    return 0


if __name__ == "__main__":
    sys.exit(main())
