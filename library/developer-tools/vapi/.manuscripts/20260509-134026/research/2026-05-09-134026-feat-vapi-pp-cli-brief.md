# Vapi CLI Brief

## API Identity
- Domain: Voice AI agents — programmable phone calls, web voice, conversational AI orchestration over transcriber/model/voice providers (OpenAI, Deepgram, ElevenLabs, etc.)
- Users: developers building voice agents (appointment reminders, AI receptionists, outbound campaigns, support bots, voice-enabled apps)
- Data profile: Assistants, Calls (with transcripts/recordings/cost), Phone Numbers, Tools (function calling), Workflows, Squads, Campaigns (bulk outbound), Files (knowledge base), Sessions, Chats, Eval suites, Insight reports, Observability scorecards, Structured Outputs

## Reachability Risk
- None — official OpenAPI 3.0 spec at https://api.vapi.ai/api-json (79 endpoints, 15 resource tags), bearer-token auth, healthy 401 on unauth probe.

## Top Workflows
1. Create/manage voice assistants (model + voice + tools + first-message + system prompt)
2. Place and monitor outbound phone calls; pull transcripts, recordings, cost breakdowns
3. Manage phone numbers (Twilio/Vonage/byo-Telnyx, BYOC SIP) and bind to assistants
4. Build & deploy bulk outbound campaigns to customer lists
5. Author and version custom tools (function calling, transfer-call, end-call) and bind to assistants
6. Evaluate assistant quality (eval runs) + observability scorecards on calls
7. Test webhooks locally without ngrok (tunnel + replay)

## Table Stakes (from VapiAI/cli + VapiAI/mcp-server)
- Auth (bearer; multi-account via OAuth/login)
- CRUD for: assistants, calls, phone-numbers, tools, workflows, squads, campaigns, chats, files, sessions, eval, insights, scorecards, structured-outputs
- `listen` webhook tunnel + forward
- MCP server setup helpers (`mcp setup cursor|windsurf|vscode`)
- `init` project scaffolding
- Logs / debugging surface

## Data Layer
- Primary entities: assistants, calls, phone_numbers, tools, workflows, squads, campaigns, chats, files, sessions, evals, eval_runs, insights, scorecards, structured_outputs
- Sync cursor: `createdAtGt` / `updatedAtGt` query params on list endpoints (cursor-paginated by id)
- FTS/search: per-table FTS5 on call.transcript, assistant.name+systemPrompt+firstMessage, tool.name+description, workflow.name, campaign.name. Unified `search` across all.
- High-gravity fields: call (id, assistant_id, phone_number_id, status, cost, transcript, started_at, ended_at, ended_reason, customer_number); assistant (id, name, model.provider, voice.provider, first_message)

## Codebase Intelligence
- Source: VapiAI/cli (Go cobra, monorepo with TS MCP server alongside)
- Auth: bearer token; CLI uses browser OAuth (`vapi login`) → stores in `~/.vapi-cli.yaml`; supports multi-account switching
- Data model: 15 resource tags, all CRUD-uniform; calls/assistants reference each other by id; campaigns reference assistant + customer list
- Rate limiting: standard REST, no documented limits in spec; backoff on 429 expected
- Architecture: stateless REST; webhooks for call lifecycle (assistant-request, status-update, end-of-call-report, function-call, transfer-destination-request)

## User Vision
- (User chose "Let's go" — no explicit vision provided. Default to building the GOAT: absorb VapiAI/cli + MCP, add offline analytics that the official CLI cannot do.)

## Product Thesis
- Name: vapi-pp-cli
- Why it should exist: The official VapiAI/cli is solid for individual CRUD + project scaffolding, but offers no offline analytics, no cross-call cost/quality analysis, no SQL composability, and no agent-native batch surface. We absorb every feature, then transcend with a local SQLite store of every call/assistant/tool that powers cost analytics, transcript FTS, drift detection, orphan cleanup, and replayable agent batches.

## Build Priorities
1. Auth + config (bearer token via env or `auth login`/`set-token`); doctor health check
2. CRUD for all 15 resources (generated from OpenAPI) with `--json --select --csv --dry-run`
3. Local SQLite store + sync of assistants/calls/tools/phone-numbers/campaigns/workflows/files; FTS5 across transcripts and names
4. Webhook listen + forward (matching official CLI feature)
5. Transcendence: cost analytics, transcript search, orphan detection, ended-reason histograms, assistant A/B comparison, bulk-call dry-run, MCP server init helper
