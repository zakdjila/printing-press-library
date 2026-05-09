# Vapi CLI

**Every Vapi feature, plus a local SQLite store that makes cost analytics, transcript search, and orphan cleanup possible — none of which the official CLI offers.**

Vapi is the developer platform for voice AI agents. vapi-pp-cli matches the official CLI on CRUD across all 15 resources, then adds offline analytics, transcript FTS, and a webhook replay buffer that the official tool can't touch. Every command supports --json/--select/--csv/--dry-run, and the full surface auto-mirrors to MCP for agents.

Printed by [@zakdjila](https://github.com/zakdjila) (Zakariadiarra).

## Install

The recommended path installs both the `vapi-pp-cli` binary and the `pp-vapi` agent skill in one shot:

```bash
npx -y @mvanhorn/printing-press install vapi
```

For CLI only (no skill):

```bash
npx -y @mvanhorn/printing-press install vapi --cli-only
```


### Without Node

The generated install path is category-agnostic until this CLI is published. If `npx` is not available before publish, install Node or use the category-specific Go fallback from the public-library entry after publish.

### Pre-built binary

Download a pre-built binary for your platform from the [latest release](https://github.com/mvanhorn/printing-press-library/releases/tag/vapi-current). On macOS, clear the Gatekeeper quarantine: `xattr -d com.apple.quarantine <binary>`. On Unix, mark it executable: `chmod +x <binary>`.

<!-- pp-hermes-install-anchor -->
## Install for Hermes

From the Hermes CLI:

```bash
hermes skills install mvanhorn/printing-press-library/cli-skills/pp-vapi --force
```

Inside a Hermes chat session:

```bash
/skills install mvanhorn/printing-press-library/cli-skills/pp-vapi --force
```

## Install for OpenClaw

Tell your OpenClaw agent (copy this):

```
Install the pp-vapi skill from https://github.com/mvanhorn/printing-press-library/tree/main/cli-skills/pp-vapi. The skill defines how its required CLI can be installed.
```

## Authentication

Vapi uses bearer-token auth. Set VAPI_API_KEY in your environment, or run `vapi-pp-cli auth set-token <key>`. Get a key from https://dashboard.vapi.ai. The official CLI's browser-OAuth login is intentionally not duplicated; if you want it, install the upstream `vapi` for `vapi login` and let this CLI read the same token.

## Quick Start

```bash
# Bearer auth — get a key from dashboard.vapi.ai
vapi-pp-cli auth set-token $VAPI_API_KEY


# Pull every assistant, call, tool, phone-number, workflow, and campaign into the local store
vapi-pp-cli sync


# Field-projected agent-native listing
vapi-pp-cli assistant list --json --select id,name,model.provider,voice.provider


# Local roll-up that the API has no preset for
vapi-pp-cli cost summary --since 7d --by assistant --json


# FTS5 over every synced transcript
vapi-pp-cli transcripts search 'cancel my appointment' --since 30d --json


# Plan an outbound call before committing (use --stdin for richer customer object)
vapi-pp-cli call create --assistant-id a1b2 --customer-id c3d4 --dry-run

```

## Unique Features

These capabilities aren't available in any other tool for this API.

### Local state that compounds

- **`cost summary`** — Aggregate call cost across the synced store by assistant, day, or phone number.

  _Agents can answer 'how much did each assistant cost last week' in one call instead of paginating /call._

  ```bash
  vapi-pp-cli cost summary --since 7d --by assistant --json
  ```
- **`transcripts search`** — Full-text search across every synced call transcript with assistant/date filters.

  _Agents can find calls matching natural-language queries without scanning every transcript over the wire._

  ```bash
  vapi-pp-cli transcripts search 'cancel my appointment' --since 30d --json --select id,assistantId,startedAt
  ```
- **`calls why`** — Group recent calls by endedReason and print counts to surface drop-offs and silence-timeouts.

  _One call answers 'why are my calls ending today' for triage._

  ```bash
  vapi-pp-cli calls why --since 24h --json
  ```

### Cleanup and hygiene

- **`orphans`** — List tools, files, phone numbers, and workflows that no assistant references.

  _Agents can clean up unused billable resources before they accrue cost._

  ```bash
  vapi-pp-cli orphans --json
  ```
- **`stale assistants`** — Assistants with no calls in the last N days based on the synced store.

  _Cleanup target list before deletion._

  ```bash
  vapi-pp-cli stale assistants --days 30 --json
  ```

### Quality signals

- **`assistants compare`** — Side-by-side cost, average duration, and ended-reason distribution for two assistants over a window.

  _Quick read on whether a new assistant variant is winning before fully rolling it out._

  ```bash
  vapi-pp-cli assistants compare a1b2 c3d4 --since 7d --json
  ```
- **`drift assistants`** — Diff every assistant against a baseline assistant to see system-prompt, model, voice, and tool drift.

  _Catch unintentional config drift across a fleet of assistants._

  ```bash
  vapi-pp-cli drift assistants --baseline a1b2 --json
  ```

### Agent-native plumbing

- **`calls bulk`** — Read a CSV of customers and print a planned outbound batch with cost estimate; --dry-run by default.

  _Agents can stage and review an outbound campaign before committing._

  ```bash
  vapi-pp-cli calls bulk --csv customers.csv --assistant-id a1b2 --phone-number-id p9 --json
  ```
- **`listen replay`** — Re-fire the last N captured webhook events at a forward target without waiting for a real call.

  _Speeds up local webhook handler iteration._

  ```bash
  vapi-pp-cli listen replay --last 5 --forward-to localhost:3000/webhook
  ```
- **`calls watch`** — Long-poll for newly active or ended calls and print one line per event.

  _A streamable handle on call lifecycle for agent supervision loops._

  ```bash
  vapi-pp-cli calls watch --interval 5s --iterations 3 --json
  ```

## Usage

Run `vapi-pp-cli --help` for the full command reference and flag list.

## Commands

### assistant

Manage assistant

- **`vapi-pp-cli assistant create`** - Create Assistant
- **`vapi-pp-cli assistant find-all`** - List Assistants
- **`vapi-pp-cli assistant find-one`** - Get Assistant
- **`vapi-pp-cli assistant remove`** - Delete Assistant
- **`vapi-pp-cli assistant update`** - Update Assistant

### call

Manage call

- **`vapi-pp-cli call create`** - Create Call
- **`vapi-pp-cli call delete-data`** - Delete Call
- **`vapi-pp-cli call find-all`** - List Calls
- **`vapi-pp-cli call find-one`** - Get Call
- **`vapi-pp-cli call update`** - Update Call

### campaign

Manage campaign

- **`vapi-pp-cli campaign create`** - Create Campaign
- **`vapi-pp-cli campaign find-all`** - List Campaigns
- **`vapi-pp-cli campaign find-one`** - Get Campaign
- **`vapi-pp-cli campaign remove`** - Delete Campaign
- **`vapi-pp-cli campaign update`** - Update Campaign

### chat

Manage chat

- **`vapi-pp-cli chat create`** - Creates a new chat with optional SMS delivery via transport field. Requires at least one of: assistantId/assistant, sessionId, or previousChatId. Note: sessionId and previousChatId are mutually exclusive. Transport field enables SMS delivery with two modes: (1) New conversation - provide transport.phoneNumberId and transport.customer to create a new session, (2) Existing conversation - provide sessionId to use existing session data. Cannot specify both sessionId and transport fields together. The transport.useLLMGeneratedMessageForOutbound flag controls whether input is processed by LLM (true, default) or forwarded directly as SMS (false).
- **`vapi-pp-cli chat create-open-aichat`** - Create Chat (OpenAI Compatible)
- **`vapi-pp-cli chat delete`** - Delete Chat
- **`vapi-pp-cli chat get`** - Get Chat
- **`vapi-pp-cli chat list`** - List Chats

### eval

Manage eval

- **`vapi-pp-cli eval create`** - Create Eval
- **`vapi-pp-cli eval get`** - Get Eval
- **`vapi-pp-cli eval get-paginated`** - List Evals
- **`vapi-pp-cli eval get-run`** - Get Eval Run
- **`vapi-pp-cli eval get-runs-paginated`** - List Eval Runs
- **`vapi-pp-cli eval remove`** - Delete Eval
- **`vapi-pp-cli eval remove-run`** - Delete Eval Run
- **`vapi-pp-cli eval run`** - Create Eval Run
- **`vapi-pp-cli eval update`** - Update Eval

### file

Manage file

- **`vapi-pp-cli file create`** - Upload File
- **`vapi-pp-cli file find-all`** - List Files
- **`vapi-pp-cli file find-one`** - Get File
- **`vapi-pp-cli file remove`** - Delete File
- **`vapi-pp-cli file update`** - Update File

### observability

Manage observability

- **`vapi-pp-cli observability scorecard-create`** - Create Scorecard
- **`vapi-pp-cli observability scorecard-get`** - Get Scorecard
- **`vapi-pp-cli observability scorecard-get-paginated`** - List Scorecards
- **`vapi-pp-cli observability scorecard-remove`** - Delete Scorecard
- **`vapi-pp-cli observability scorecard-update`** - Update Scorecard

### phone-number

Manage phone number

- **`vapi-pp-cli phone-number create`** - Create Phone Number
- **`vapi-pp-cli phone-number find-all`** - List Phone Numbers
- **`vapi-pp-cli phone-number find-all-paginated`** - List Phone Numbers
- **`vapi-pp-cli phone-number find-one`** - Get Phone Number
- **`vapi-pp-cli phone-number remove`** - Delete Phone Number
- **`vapi-pp-cli phone-number update`** - Update Phone Number

### provider

Manage provider

- **`vapi-pp-cli provider resource-create-resource`** - Create Provider Resource
- **`vapi-pp-cli provider resource-delete-resource`** - Delete Provider Resource
- **`vapi-pp-cli provider resource-get-resource`** - Get Provider Resource
- **`vapi-pp-cli provider resource-get-resources-paginated`** - List Provider Resources
- **`vapi-pp-cli provider resource-update-resource`** - Update Provider Resource

### reporting

Manage reporting

- **`vapi-pp-cli reporting insight-create`** - Create Insight
- **`vapi-pp-cli reporting insight-find-all`** - Get Insights
- **`vapi-pp-cli reporting insight-find-one`** - Get Insight
- **`vapi-pp-cli reporting insight-preview`** - Preview Insight
- **`vapi-pp-cli reporting insight-remove`** - Delete Insight
- **`vapi-pp-cli reporting insight-run`** - Run Insight
- **`vapi-pp-cli reporting insight-update`** - Update Insight

### session

Manage session

- **`vapi-pp-cli session create`** - Create Session
- **`vapi-pp-cli session find-all-paginated`** - List Sessions
- **`vapi-pp-cli session find-one`** - Get Session
- **`vapi-pp-cli session remove`** - Delete Session
- **`vapi-pp-cli session update`** - Update Session

### squad

Manage squad

- **`vapi-pp-cli squad create`** - Create Squad
- **`vapi-pp-cli squad find-all`** - List Squads
- **`vapi-pp-cli squad find-one`** - Get Squad
- **`vapi-pp-cli squad remove`** - Delete Squad
- **`vapi-pp-cli squad update`** - Update Squad

### structured-output

Manage structured output

- **`vapi-pp-cli structured-output create`** - Create Structured Output
- **`vapi-pp-cli structured-output find-all`** - List Structured Outputs
- **`vapi-pp-cli structured-output find-one`** - Get Structured Output
- **`vapi-pp-cli structured-output remove`** - Delete Structured Output
- **`vapi-pp-cli structured-output run`** - Run Structured Output
- **`vapi-pp-cli structured-output update`** - Update Structured Output

### tool

Manage tool

- **`vapi-pp-cli tool create`** - Create Tool
- **`vapi-pp-cli tool find-all`** - List Tools
- **`vapi-pp-cli tool find-one`** - Get Tool
- **`vapi-pp-cli tool remove`** - Delete Tool
- **`vapi-pp-cli tool update`** - Update Tool

### vapi-analytics

Manage vapi analytics

- **`vapi-pp-cli vapi-analytics query`** - Create Analytics Queries


## Output Formats

```bash
# Human-readable table (default in terminal, JSON when piped)
vapi-pp-cli chat list

# JSON for scripting and agents
vapi-pp-cli chat list --json

# Filter to specific fields
vapi-pp-cli chat list --json --select id,name,status

# Dry run — show the request without sending
vapi-pp-cli chat list --dry-run

# Agent mode — JSON + compact + no prompts in one flag
vapi-pp-cli chat list --agent
```

## Agent Usage

This CLI is designed for AI agent consumption:

- **Non-interactive** - never prompts, every input is a flag
- **Pipeable** - `--json` output to stdout, errors to stderr
- **Filterable** - `--select id,name` returns only fields you need
- **Previewable** - `--dry-run` shows the request without sending
- **Explicit retries** - add `--idempotent` to create retries and `--ignore-missing` to delete retries when a no-op success is acceptable
- **Confirmable** - `--yes` for explicit confirmation of destructive actions
- **Piped input** - write commands can accept structured input when their help lists `--stdin`
- **Offline-friendly** - sync/search commands can use the local SQLite store when available
- **Agent-safe by default** - no colors or formatting unless `--human-friendly` is set

Exit codes: `0` success, `2` usage error, `3` not found, `4` auth error, `5` API error, `7` rate limited, `10` config error.

## Use with Claude Code

Install the focused skill — it auto-installs the CLI on first invocation:

```bash
npx skills add mvanhorn/printing-press-library/cli-skills/pp-vapi -g
```

Then invoke `/pp-vapi <query>` in Claude Code. The skill is the most efficient path — Claude Code drives the CLI directly without an MCP server in the middle.

<details>
<summary>Use as an MCP server in Claude Code (advanced)</summary>

If you'd rather register this CLI as an MCP server in Claude Code, install the MCP binary first:


Install the MCP binary from this CLI's published public-library entry or pre-built release.

Then register it:

```bash
claude mcp add vapi vapi-pp-mcp -e VAPI_TOKEN=<your-token>
```

</details>

## Use with Claude Desktop

This CLI ships an [MCPB](https://github.com/modelcontextprotocol/mcpb) bundle — Claude Desktop's standard format for one-click MCP extension installs (no JSON config required).

To install:

1. Download the `.mcpb` for your platform from the [latest release](https://github.com/mvanhorn/printing-press-library/releases/tag/vapi-current).
2. Double-click the `.mcpb` file. Claude Desktop opens and walks you through the install.
3. Fill in `VAPI_TOKEN` when Claude Desktop prompts you.

Requires Claude Desktop 1.0.0 or later. Pre-built bundles ship for macOS Apple Silicon (`darwin-arm64`) and Windows (`amd64`, `arm64`); for other platforms, use the manual config below.

<details>
<summary>Manual JSON config (advanced)</summary>

If you can't use the MCPB bundle (older Claude Desktop, unsupported platform), install the MCP binary and configure it manually.


Install the MCP binary from this CLI's published public-library entry or pre-built release.

Add to your Claude Desktop config (`~/Library/Application Support/Claude/claude_desktop_config.json`):

```json
{
  "mcpServers": {
    "vapi": {
      "command": "vapi-pp-mcp",
      "env": {
        "VAPI_TOKEN": "<your-key>"
      }
    }
  }
}
```

</details>

## Health Check

```bash
vapi-pp-cli doctor
```

Verifies configuration, credentials, and connectivity to the API.

## Configuration

Config file: `~/.config/vapi-pp-cli/config.toml`

Static request headers can be configured under `headers`; per-command header overrides take precedence.

Environment variables:

| Name | Kind | Required | Description |
| --- | --- | --- | --- |
| `VAPI_TOKEN` | per_call | Yes | Set to your API credential. |

## Troubleshooting
**Authentication errors (exit code 4)**
- Run `vapi-pp-cli doctor` to check credentials
- Verify the environment variable is set: `echo $VAPI_TOKEN`
**Not found errors (exit code 3)**
- Check the resource ID is correct
- Run the `list` command to see available items

### API-specific

- **401 Invalid Key — 'private key vs public key'** — Vapi has separate public and private keys. Use the private (server) key for this CLI; the public key is for the browser SDK.
- **sync is slow on first run** — Initial sync paginates every resource; subsequent runs are incremental via updatedAtGt. Use `vapi-pp-cli sync calls --since 7d` to scope the first pull.
- **transcripts search returns nothing** — Run `vapi-pp-cli sync calls` first — the FTS index lives in the local store, not the API.
- **listen forwards but my handler never sees events** — Confirm Vapi is delivering to your tunnel URL by tailing `vapi-pp-cli logs webhooks`; check `--skip-verify` if your local handler is HTTPS with a self-signed cert.

---

## Sources & Inspiration

This CLI was built by studying these projects and resources:

- [**VapiAI/cli**](https://github.com/VapiAI/cli) — Go
- [**VapiAI/mcp-server**](https://github.com/VapiAI/mcp-server) — TypeScript
- [**@vapi-ai/server-sdk**](https://github.com/VapiAI/server-sdk-typescript) — TypeScript
- [**vapi-server-sdk (python)**](https://github.com/VapiAI/server-sdk-python) — Python
- [**askjohngeorge/vapi-vct**](https://github.com/askjohngeorge/vapi-vct) — Python

Generated by [CLI Printing Press](https://github.com/mvanhorn/cli-printing-press)
