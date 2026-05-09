# Vapi CLI Absorb Manifest

## Sources audited
1. **VapiAI/cli** (Go cobra, official) — auth, assistant/call/phone/tool/workflow/campaign/chat CRUD, listen, mcp, init, logs, config
2. **VapiAI/mcp-server** (TS, official) — `vapi_list_assistants`, `vapi_create_assistant`, `vapi_update_assistant`, `vapi_get_assistant`, `vapi_list_calls`, `vapi_get_call`, `vapi_create_call`, `vapi_list_tools`, `vapi_get_tool`, `vapi_create_tool`, `vapi_update_tool`, `vapi_delete_tool`
3. **@vapi-ai/server-sdk** (TS) — full REST surface across all 15 resources
4. **vapi-server-sdk** (Python) — same shape as TS SDK
5. **askjohngeorge/vapi-vct** — version control / config sync for assistants
6. **OpenAPI 3.0 spec** (https://api.vapi.ai/api-json) — 79 endpoints, 15 resource tags, bearer auth

## Absorbed (match or beat everything that exists)

| # | Feature | Best Source | Our Implementation | Added Value |
|---|---------|-------------|--------------------|-------------|
| 1 | List assistants | VapiAI/cli `assistant list` | Generated `assistant list` | `--json --select --csv --limit --since`, FTS via local store |
| 2 | Get assistant | VapiAI/cli `assistant get` | Generated `assistant get <id>` | `--json --select` field projection |
| 3 | Create assistant | VapiAI/cli `assistant create` | Generated `assistant create` | `--stdin` JSON body, `--dry-run`, batch via NDJSON |
| 4 | Update assistant | MCP `vapi_update_assistant` | Generated `assistant update <id>` | `--patch` JSON, `--dry-run` |
| 5 | Delete assistant | VapiAI/cli `assistant delete` | Generated `assistant delete <id>` | `--dry-run`, idempotent |
| 6 | List calls | VapiAI/cli + MCP | Generated `call list` | `--assistant-id`, `--phone-number-id`, `--status`, `--since`, FTS on transcript |
| 7 | Get call | VapiAI/cli + MCP | Generated `call get <id>` | `--json --select` (transcript-only, cost-only) |
| 8 | Create outbound call | MCP `vapi_create_call` | Generated `call create` | `--assistant-id`/`--workflow-id`, `--customer-number`, `--scheduled-at`, `--dry-run` |
| 9 | Update call (transfer/end) | OpenAPI PATCH /call/{id} | Generated `call update` | `--patch` JSON |
| 10 | Delete call | OpenAPI DELETE /call/{id} | Generated `call delete` | `--dry-run` |
| 11 | List phone numbers | VapiAI/cli `phone list` | Generated `phone-number list` | `--json`, FTS, store-backed |
| 12 | Get phone number | VapiAI/cli `phone get` | Generated `phone-number get` | `--json --select` |
| 13 | Buy/import phone number | VapiAI/cli `phone create` | Generated `phone-number create` | `--provider twilio/vonage/byo/vapi`, `--stdin` |
| 14 | Update phone | VapiAI/cli `phone update` | Generated `phone-number update` | `--assistant-id` rebind shorthand |
| 15 | Release phone | VapiAI/cli `phone delete` | Generated `phone-number delete` | `--dry-run` |
| 16 | List/get/create/update/delete tools | VapiAI/cli `tool ...` + MCP | Generated `tool ...` | `--json --select --stdin`, batch |
| 17 | Tool types | VapiAI/cli `tool types` | `tool types` | static enum from spec, no API call |
| 18 | List/get/create/update/delete workflows | VapiAI/cli `workflow ...` | Generated `workflow ...` | `--json --select --stdin` |
| 19 | List/get/create/update/delete campaigns | VapiAI/cli `campaign ...` | Generated `campaign ...` | `--json --select --stdin`, `--customer-list-stdin` |
| 20 | List/get/create/continue/delete chats | VapiAI/cli `chat ...` | Generated `chat ...` | `--json --select`, FTS on chat.messages |
| 21 | Files (upload/list/get/update/delete) | OpenAPI /file | Generated `file ...` | `--upload <path>`, `--json --select` |
| 22 | Squads CRUD | OpenAPI /squad | Generated `squad ...` | `--json --select --stdin` |
| 23 | Sessions CRUD | OpenAPI /session | Generated `session ...` | `--json --select` |
| 24 | Eval CRUD + run | OpenAPI /eval, /eval/run | Generated `eval ...`, `eval run create` | `--json --select`, run polling helper |
| 25 | Insight CRUD + run + preview | OpenAPI /reporting/insight | Generated `insight ...` | `--json --select` |
| 26 | Observability scorecards CRUD | OpenAPI /observability/scorecard | Generated `scorecard ...` | `--json --select` |
| 27 | Structured outputs CRUD + run | OpenAPI /structured-output | Generated `structured-output ...` | `--json --select` |
| 28 | Provider-resources CRUD | OpenAPI /provider | Generated `provider ...` | per-provider passthrough |
| 29 | Analytics query | OpenAPI POST /analytics | Generated `analytics query` | `--stdin` query body |
| 30 | Webhook tunnel/forward | VapiAI/cli `listen --forward-to` | Hand-built `listen --forward-to` | local proxy, `--port`, `--skip-verify`, replay last N events |
| 31 | MCP server (full surface) | VapiAI/mcp-server (12 tools) | Auto via cobratree walker | every Cobra command becomes an MCP tool with proper read/write hints; remote `[stdio,http]` transport |
| 32 | MCP setup helper for IDEs | VapiAI/cli `mcp setup cursor/...` | Hand-built `mcp setup <ide>` | writes `.cursor/mcp.json`, `.windsurf/...`, etc. |
| 33 | Auth login (env + bearer + status) | VapiAI/cli `auth ...` | Generated `auth set-token`, `auth status`, `auth logout` | env-var first; doctor check |
| 34 | Logs/debug surface | VapiAI/cli `logs ...` | `logs calls`, `logs errors`, `logs webhooks` | local store query, no API needed once synced |
| 35 | Project init / SDK install | VapiAI/cli `vapi init` | Out of scope (printed CLI is itself the dev tool; generating SDK install is outside the absorb scope and would duplicate the official CLI for negative value) | (stub — out of scope) |
| 36 | Doctor / health check | new | `doctor` | tests auth, base URL, store, key validity |
| 37 | Sync (full + incremental) | new (table-stakes for store) | `sync`, `sync <resource>`, `sync --since` | populates SQLite store via `updatedAtGt` |
| 38 | Search across all entities | new | `search "<query>"` | FTS5 over assistants, calls.transcript, tools, workflows, chats |
| 39 | SQL query | new | `sql "<select ...>"` | read-only SQL against the local store |
| 40 | Agent context | new | `context` | dumps shape/tables/fields for agent priming |

### Stubs (explicit)
| # | Feature | Status | Reason |
|---|---------|--------|--------|
| 35 | Project init / SDK installer | (stub — out of scope) | Duplicates upstream `vapi init`; building it over again gives the user nothing. We document the recommended path: install official `vapi` for `init`, install `vapi-pp-cli` for everything else. |

## Transcendence (only possible with our approach)

| # | Feature | Command | Why Only We Can Do This | Score |
|---|---------|---------|--------------------------|-------|
| 1 | Cost rollup across calls | `cost summary --since 7d --by assistant\|day\|phone-number` | Requires local SQL aggregation across thousands of synced calls; no `/analytics` preset for this exact roll-up shape | 9/10 |
| 2 | Transcript FTS search | `transcripts search "<query>" --assistant-id <id>` | Requires call.transcript indexed in local FTS5; the API has no transcript-search endpoint | 9/10 |
| 3 | Ended-reason histogram | `calls why --since 24h` | Aggregates `endedReason` across recent calls to surface drops/silence-timeouts/customer-hangups; needs local group-by | 8/10 |
| 4 | Orphan resource detection | `orphans` | Finds tools/files/phone-numbers/workflows referenced by 0 assistants; needs cross-table joins on synced data | 8/10 |
| 5 | Assistant A/B comparison | `assistant compare <id-a> <id-b> --since 7d` | Side-by-side cost/duration/end-reason for two assistants over the same window; needs grouped local query | 8/10 |
| 6 | Stale assistant cleanup hint | `stale assistants --days 30` | Lists assistants never used in N days based on synced calls; pure local | 7/10 |
| 7 | Bulk-call dry-run plan | `call bulk --csv customers.csv --assistant-id <id> --dry-run` | Reads CSV of customers, prints planned API calls + cost estimate without sending; campaign endpoint requires dashboard for this UX | 8/10 |
| 8 | Replay last N webhook events | `listen replay --last 5` | The official `listen` doesn't keep a buffer; we persist events and replay against a different forward target | 7/10 |
| 9 | Watch (stream) live calls | `call watch --interval 5s` | Polls the synced store, surfaces newly active/ended calls with cost & duration; agent-friendly long-poll | 6/10 |
| 10 | Drift detect on assistants | `drift assistants --baseline <id>` | Compares all assistants against a baseline JSON, prints minimal diff of system prompt / model / voice / tools; needs local pairing | 6/10 |

All transcendence rows ≥ 5/10 → all ship.

## What we beat the official CLI on
- **Offline analytics** — the official CLI has zero local persistence; it's a thin REST shell. We sync to SQLite and unlock cost rollups, transcript FTS, drift, and orphan detection.
- **Agent-native** — every command supports `--json --select --csv --dry-run --quiet`; typed exit codes; auto-MCP via cobratree walker (~50+ tools mirrored, `mcp:read-only` annotated).
- **Composability** — `sql` lets agents pose ad-hoc questions; we've curated FTS5 indexes on the high-cardinality fields (transcripts, names).
- **Webhook replay buffer** — `listen` keeps a ring buffer; `listen replay` re-fires events.
- **Ergonomics** — bulk-call planning from CSV with cost estimate before commit.
