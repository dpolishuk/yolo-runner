# Serena MCP (Project-Local)

Purpose: project-local Serena MCP for OpenCode.

Requirements:
- `uvx` available on PATH

Steps:
1) Create `.opencode/opencode.jsonc` with the following MCP block:

```jsonc
{
  "$schema": "https://opencode.ai/config.json",
  "mcp": {
    "serena": {
      "type": "local",
      "command": [
        "uvx",
        "--from",
        "git+https://github.com/oraios/serena",
        "serena",
        "start-mcp-server",
        "--context",
        "ide",
        "--project",
        "."
      ]
    }
  }
}
```

2) Run `opencode mcp list` to verify the server is registered (status is unavailable).

Note: Keep `--project .` and `--context ide` for project-local setup.
