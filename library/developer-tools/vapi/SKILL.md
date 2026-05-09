---
name: pp-vapi
description: "Every Vapi feature, plus a local SQLite store that makes cost analytics, transcript search, and orphan cleanup... Trigger phrases: `vapi cost last week`, `search vapi transcripts for`, `why are my vapi calls ending`, `list vapi assistants`, `use vapi-pp-cli`, `run vapi-pp-cli`."
author: "Zakariadiarra"
license: "Apache-2.0"
argument-hint: "<command> [args] | install cli|mcp"
allowed-tools: "Read Bash"
metadata:
  openclaw:
    requires:
      bins:
        - vapi-pp-cli
---

# Vapi — Printing Press CLI

## Prerequisites: Install the CLI

This skill drives the `vapi-pp-cli` binary. **You must verify the CLI is installed before invoking any command from this skill.** If it is missing, install it first:

1. Install via the Printing Press installer:
   ```bash
   npx -y @mvanhorn/printing-press install vapi --cli-only
   ```
2. Verify: `vapi-pp-cli --version`
3. Ensure `$GOPATH/bin` (or `$HOME/go/bin`) is on `$PATH`.

If the `npx` install fails (no Node, offline, etc.), fall back to a direct Go install (requires Go 1.26.3 or newer):

```bash
go install github.com/mvanhorn/printing-press-library/library/developer-tools/vapi/cmd/vapi-pp-cli@latest
```

If `--version` reports "command not found" after install, the install step did not put the binary on `$PATH`. Do not proceed with skill commands until verification succeeds.

Vapi is the developer platform for voice AI agents. vapi-pp-cli matches the official CLI on CRUD across all 15 resources, then adds offline analytics, transcript FTS, and a webhook replay buffer that the official tool can't touch. Every command supports --json/--select/--csv/--dry-run, and the full surface auto-mirrors to MCP for agents.

## When to Use This CLI

Reach for vapi-pp-cli when an agent needs to plan or audit Vapi resources, answer cost/quality questions across many calls, search transcripts, detect orphan tools/files, or stage bulk outbound calls. For one-off browser-OAuth login or framework SDK install, the upstream `vapi` CLI is fine — they coexist.

## Unique Capabilities

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

## Command Reference

**assistant** — Manage assistant

- `vapi-pp-cli assistant create` — Create Assistant
- `vapi-pp-cli assistant find-all` — List Assistants
- `vapi-pp-cli assistant find-one` — Get Assistant
- `vapi-pp-cli assistant remove` — Delete Assistant
- `vapi-pp-cli assistant update` — Update Assistant

**call** — Manage call

- `vapi-pp-cli call create` — Create Call
- `vapi-pp-cli call delete-data` — Delete Call
- `vapi-pp-cli call find-all` — List Calls
- `vapi-pp-cli call find-one` — Get Call
- `vapi-pp-cli call update` — Update Call

**campaign** — Manage campaign

- `vapi-pp-cli campaign create` — Create Campaign
- `vapi-pp-cli campaign find-all` — List Campaigns
- `vapi-pp-cli campaign find-one` — Get Campaign
- `vapi-pp-cli campaign remove` — Delete Campaign
- `vapi-pp-cli campaign update` — Update Campaign

**chat** — Manage chat

- `vapi-pp-cli chat create` — Creates a new chat with optional SMS delivery via transport field. Requires at least one of: assistantId/assistant,...
- `vapi-pp-cli chat create-open-aichat` — Create Chat (OpenAI Compatible)
- `vapi-pp-cli chat delete` — Delete Chat
- `vapi-pp-cli chat get` — Get Chat
- `vapi-pp-cli chat list` — List Chats

**eval** — Manage eval

- `vapi-pp-cli eval create` — Create Eval
- `vapi-pp-cli eval get` — Get Eval
- `vapi-pp-cli eval get-paginated` — List Evals
- `vapi-pp-cli eval get-run` — Get Eval Run
- `vapi-pp-cli eval get-runs-paginated` — List Eval Runs
- `vapi-pp-cli eval remove` — Delete Eval
- `vapi-pp-cli eval remove-run` — Delete Eval Run
- `vapi-pp-cli eval run` — Create Eval Run
- `vapi-pp-cli eval update` — Update Eval

**file** — Manage file

- `vapi-pp-cli file create` — Upload File
- `vapi-pp-cli file find-all` — List Files
- `vapi-pp-cli file find-one` — Get File
- `vapi-pp-cli file remove` — Delete File
- `vapi-pp-cli file update` — Update File

**observability** — Manage observability

- `vapi-pp-cli observability scorecard-create` — Create Scorecard
- `vapi-pp-cli observability scorecard-get` — Get Scorecard
- `vapi-pp-cli observability scorecard-get-paginated` — List Scorecards
- `vapi-pp-cli observability scorecard-remove` — Delete Scorecard
- `vapi-pp-cli observability scorecard-update` — Update Scorecard

**phone-number** — Manage phone number

- `vapi-pp-cli phone-number create` — Create Phone Number
- `vapi-pp-cli phone-number find-all` — List Phone Numbers
- `vapi-pp-cli phone-number find-all-paginated` — List Phone Numbers
- `vapi-pp-cli phone-number find-one` — Get Phone Number
- `vapi-pp-cli phone-number remove` — Delete Phone Number
- `vapi-pp-cli phone-number update` — Update Phone Number

**provider** — Manage provider

- `vapi-pp-cli provider resource-create-resource` — Create Provider Resource
- `vapi-pp-cli provider resource-delete-resource` — Delete Provider Resource
- `vapi-pp-cli provider resource-get-resource` — Get Provider Resource
- `vapi-pp-cli provider resource-get-resources-paginated` — List Provider Resources
- `vapi-pp-cli provider resource-update-resource` — Update Provider Resource

**reporting** — Manage reporting

- `vapi-pp-cli reporting insight-create` — Create Insight
- `vapi-pp-cli reporting insight-find-all` — Get Insights
- `vapi-pp-cli reporting insight-find-one` — Get Insight
- `vapi-pp-cli reporting insight-preview` — Preview Insight
- `vapi-pp-cli reporting insight-remove` — Delete Insight
- `vapi-pp-cli reporting insight-run` — Run Insight
- `vapi-pp-cli reporting insight-update` — Update Insight

**session** — Manage session

- `vapi-pp-cli session create` — Create Session
- `vapi-pp-cli session find-all-paginated` — List Sessions
- `vapi-pp-cli session find-one` — Get Session
- `vapi-pp-cli session remove` — Delete Session
- `vapi-pp-cli session update` — Update Session

**squad** — Manage squad

- `vapi-pp-cli squad create` — Create Squad
- `vapi-pp-cli squad find-all` — List Squads
- `vapi-pp-cli squad find-one` — Get Squad
- `vapi-pp-cli squad remove` — Delete Squad
- `vapi-pp-cli squad update` — Update Squad

**structured-output** — Manage structured output

- `vapi-pp-cli structured-output create` — Create Structured Output
- `vapi-pp-cli structured-output find-all` — List Structured Outputs
- `vapi-pp-cli structured-output find-one` — Get Structured Output
- `vapi-pp-cli structured-output remove` — Delete Structured Output
- `vapi-pp-cli structured-output run` — Run Structured Output
- `vapi-pp-cli structured-output update` — Update Structured Output

**tool** — Manage tool

- `vapi-pp-cli tool create` — Create Tool
- `vapi-pp-cli tool find-all` — List Tools
- `vapi-pp-cli tool find-one` — Get Tool
- `vapi-pp-cli tool remove` — Delete Tool
- `vapi-pp-cli tool update` — Update Tool

**vapi-analytics** — Manage vapi analytics

- `vapi-pp-cli vapi-analytics` — Create Analytics Queries


### Finding the right command

When you know what you want to do but not which command does it, ask the CLI directly:

```bash
vapi-pp-cli which "<capability in your own words>"
```

`which` resolves a natural-language capability query to the best matching command from this CLI's curated feature index. Exit code `0` means at least one match; exit code `2` means no confident match — fall back to `--help` or use a narrower query.

## Recipes


### Field-projected list of recent calls

```bash
vapi-pp-cli call find-all --json --select id,assistantId,status,endedReason,cost,startedAt --limit 50
```

Agent-native; pipe to jq or feed straight back into the LLM context with low token cost.

### Why are my calls ending

```bash
vapi-pp-cli calls why --since 24h --json
```

Histogram of endedReason in one call; great for daily triage agents.

### Find calls about a topic

```bash
vapi-pp-cli transcripts search 'refund' --since 30d --json --select id,assistantId,startedAt
```

FTS5 across every synced transcript with field projection — agent context-friendly.

### Compare two assistants over a window

```bash
vapi-pp-cli assistants compare a1b2 c3d4 --since 7d --json
```

Cost, duration, and ended-reason distribution side by side.

### Stage a bulk call without sending

```bash
vapi-pp-cli calls bulk --help
```

Bulk-call planner. Provide --csv <path> --assistant-id <id> --phone-number-id <id>; default is dry-run, pass --commit to actually place them.

## Auth Setup

Vapi uses bearer-token auth. Set VAPI_API_KEY in your environment, or run `vapi-pp-cli auth set-token <key>`. Get a key from https://dashboard.vapi.ai. The official CLI's browser-OAuth login is intentionally not duplicated; if you want it, install the upstream `vapi` for `vapi login` and let this CLI read the same token.

Run `vapi-pp-cli doctor` to verify setup.

## Agent Mode

Add `--agent` to any command. Expands to: `--json --compact --no-input --no-color --yes`.

- **Pipeable** — JSON on stdout, errors on stderr
- **Filterable** — `--select` keeps a subset of fields. Dotted paths descend into nested structures; arrays traverse element-wise. Critical for keeping context small on verbose APIs:

  ```bash
  vapi-pp-cli chat list --agent --select id,name,status
  ```
- **Previewable** — `--dry-run` shows the request without sending
- **Offline-friendly** — sync/search commands can use the local SQLite store when available
- **Non-interactive** — never prompts, every input is a flag
- **Explicit retries** — use `--idempotent` only when an already-existing create should count as success, and `--ignore-missing` only when a missing delete target should count as success

### Response envelope

Commands that read from the local store or the API wrap output in a provenance envelope:

```json
{
  "meta": {"source": "live" | "local", "synced_at": "...", "reason": "..."},
  "results": <data>
}
```

Parse `.results` for data and `.meta.source` to know whether it's live or local. A human-readable `N results (live)` summary is printed to stderr only when stdout is a terminal — piped/agent consumers get pure JSON on stdout.

## Agent Feedback

When you (or the agent) notice something off about this CLI, record it:

```
vapi-pp-cli feedback "the --since flag is inclusive but docs say exclusive"
vapi-pp-cli feedback --stdin < notes.txt
vapi-pp-cli feedback list --json --limit 10
```

Entries are stored locally at `~/.vapi-pp-cli/feedback.jsonl`. They are never POSTed unless `VAPI_FEEDBACK_ENDPOINT` is set AND either `--send` is passed or `VAPI_FEEDBACK_AUTO_SEND=true`. Default behavior is local-only.

Write what *surprised* you, not a bug report. Short, specific, one line: that is the part that compounds.

## Output Delivery

Every command accepts `--deliver <sink>`. The output goes to the named sink in addition to (or instead of) stdout, so agents can route command results without hand-piping. Three sinks are supported:

| Sink | Effect |
|------|--------|
| `stdout` | Default; write to stdout only |
| `file:<path>` | Atomically write output to `<path>` (tmp + rename) |
| `webhook:<url>` | POST the output body to the URL (`application/json` or `application/x-ndjson` when `--compact`) |

Unknown schemes are refused with a structured error naming the supported set. Webhook failures return non-zero and log the URL + HTTP status on stderr.

## Named Profiles

A profile is a saved set of flag values, reused across invocations. Use it when a scheduled agent calls the same command every run with the same configuration - HeyGen's "Beacon" pattern.

```
vapi-pp-cli profile save briefing --json
vapi-pp-cli --profile briefing chat list
vapi-pp-cli profile list --json
vapi-pp-cli profile show briefing
vapi-pp-cli profile delete briefing --yes
```

Explicit flags always win over profile values; profile values win over defaults. `agent-context` lists all available profiles under `available_profiles` so introspecting agents discover them at runtime.

## Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 2 | Usage error (wrong arguments) |
| 3 | Resource not found |
| 4 | Authentication required |
| 5 | API error (upstream issue) |
| 7 | Rate limited (wait and retry) |
| 10 | Config error |

## Argument Parsing

Parse `$ARGUMENTS`:

1. **Empty, `help`, or `--help`** → show `vapi-pp-cli --help` output
2. **Starts with `install`** → ends with `mcp` → MCP installation; otherwise → see Prerequisites above
3. **Anything else** → Direct Use (execute as CLI command with `--agent`)

## MCP Server Installation

Install the MCP binary from this CLI's published public-library entry or pre-built release, then register it:

```bash
claude mcp add vapi-pp-mcp -- vapi-pp-mcp
```

Verify: `claude mcp list`

## Direct Use

1. Check if installed: `which vapi-pp-cli`
   If not found, offer to install (see Prerequisites at the top of this skill).
2. Match the user query to the best command from the Unique Capabilities and Command Reference above.
3. Execute with the `--agent` flag:
   ```bash
   vapi-pp-cli <command> [subcommand] [args] --agent
   ```
4. If ambiguous, drill into subcommand help: `vapi-pp-cli <command> --help`.
