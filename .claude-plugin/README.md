# Loom Claude Code Plugin

This directory contains the configuration for using Loom as a Claude Code plugin.

## Installation

The Loom plugin uses `go install` with version pinning for reliable, reproducible installations.

### Quick Start

1. Install the Loom binary:
   ```bash
   go install github.com/jake-mok-nelson/loom@v1.0.0
   ```

2. Add the plugin to Claude Code:
   ```bash
   claude plugin add /path/to/loom
   ```

### Using the Install Script

Alternatively, run the provided installation script:

```bash
cd /path/to/loom/.claude-plugin
./install.sh
```

## Configuration

### plugin.json

The main plugin manifest that defines metadata, commands, and references the MCP server configuration.

### .mcp.json

Configures the MCP server connection. The plugin expects the Loom binary to be installed at:
- `$GOBIN/loom` (if GOBIN is set)
- `$GOPATH/bin/loom` (if GOPATH is set)
- `$HOME/go/bin/loom` (default Go installation path)

To customize the database path, add `LOOM_DB_PATH` to the `env` block:

```json
{
  "loom": {
    "command": "${GOBIN:-${GOPATH:-${HOME}/go}/bin}/loom",
    "args": [],
    "env": {
      "LOOM_DB_PATH": "/custom/path/to/loom.db"
    }
  }
}
```

## Version Pinning

The plugin uses `go install` with semantic version tags to ensure consistent, reproducible installations. The version is specified in:
- `install.sh` script (VERSION variable)
- Installation documentation

To update the version, ensure the repository has a corresponding git tag (e.g., `v1.0.0`) and update the references in:
- `.claude-plugin/install.sh`
- `README.md` installation instructions

## Development

For local development without installing to $GOPATH/bin, you can temporarily use `go run`:

1. Edit `.mcp.json` to use:
   ```json
   {
     "loom": {
       "command": "go",
       "args": ["run", "${CLAUDE_PLUGIN_ROOT}"],
       "env": {}
     }
   }
   ```

2. Revert to the installed binary configuration before committing.

## Commands

The `commands/` directory contains markdown files that define custom commands available through the plugin:
- `plan.md` - Interactive project planning
- `status.md` - Show project status
- `review.md` - Review tasks and progress
- `blocked.md` - Show blocked tasks

## Skills

The `skills/manage/` directory contains the Loom project management skill that enables proactive task tracking and management throughout conversations.
