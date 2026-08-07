# Task Registry - MCP Host Project

| ID | Description | Priority | Sprint | DEV | TEST | QC | Status |
|----|-------------|----------|--------|-----|------|-----|--------|
| FEAT-00001 | Bootstrap Go module, dependency baseline, and CLI composition root | P0 | S1 | ✅ | ✅ | ✅ | DONE |
| FEAT-00002 | Configuration validation and safe Filesystem MCP process policy | P0 | S1 | ✅ | ✅ | ✅ | DONE |
| FEAT-00003 | Stdio MCP client lifecycle: initialize, discovery, call, timeout, cleanup | P0 | S1 | ✅ | ✅ | ✅ | DONE |
| FEAT-00004 | MCP JSON Schema to Gemini function-declaration mapper | P0 | S1 | ✅ | ✅ | ✅ | DONE |
| FEAT-00005 | Gemini provider adapter and conversation request/response mapping | P0 | S1 | ✅ | ✅ | ✅ | DONE |
| FEAT-00006 | Bounded agentic loop, tool-result context, cancellation, and error handling | P0 | S1 | ✅ | ✅ | ✅ | DONE |
| FEAT-00007 | Headless CLI prompt input and final-response output | P1 | S1 | ✅ | ✅ | ✅ | DONE |
| QA-00001 | Unit tests for config, policy, mapper, and agent loop | P0 | S1 | ⬜ | ⬜ | ⬜ | SPRINT BACKLOG |
| QA-00002 | Fake MCP stdio integration tests for discovery, timeout, and crash | P0 | S1 | ⬜ | ⬜ | ⬜ | SPRINT BACKLOG |
| QA-00003 | Opt-in E2E test with sandboxed Filesystem MCP Server | P0 | S1 | ⬜ | ⬜ | ⬜ | SPRINT BACKLOG |
| DEVOPS-00001 | CI quality gates: go test, go vet, and coverage report | P1 | S1 | ⬜ | ⬜ | ⬜ | SPRINT BACKLOG |
| DOC-00001 | Document Gemini and Filesystem MCP local setup and security constraints | P1 | S1 | ⬜ | ⬜ | ⬜ | SPRINT BACKLOG |
| FEAT-00008 | HTTP/SSE API Gateway: authenticated POST /v1/chat streaming agent progress over SSE + GET /healthz | P2 | S2 | ✅ | ✅ | ✅ | DONE |
| FEAT-00010 | Agent OnEvent observer seam in internal/agent (additive: start/text/tool/final events) | P2 | S2 | ✅ | ✅ | ✅ | DONE |
| FEAT-00011 | GATEWAY_* config keys: ADDR, API_TOKEN, API_TOKENS, MAX_CONCURRENT | P2 | S2 | ✅ | ✅ | ✅ | DONE |
| FEAT-00012 | internal/server HTTP layer: router, Bearer auth, SSE encoder, Handler seam, healthz | P2 | S2 | ✅ | ✅ | ✅ | DONE |
| FEAT-00013 | internal/runner per-request runner: MCP lifecycle, agent loop, event mapping, cancel | P2 | S2 | ✅ | ✅ | ✅ | DONE |
| FEAT-00014 | cmd/gateway entrypoint: config wiring, :8080 listen, health, docs | P2 | S2 | ✅ | ✅ | ✅ | DONE |
| FEAT-00015 | Next.js UI + server-side token proxy (follow-up; out of scope for FEAT-00008) | P2 | S2 | ⬜ | ⬜ | ⬜ | DEFERRED |
| FEAT-00009 | Additional LLM providers and MCP server profiles | P2 | S2 | ⬜ | ⬜ | ⬜ | DEFERRED |

---

## FEAT-00008 Sub-Task Breakdown (per TDD build sequence, `docs/tdd/FEAT-00008_gateway_tdd.md` §9)

| ID | Title | Owner | Dependencies | Priority |
|----|-------|-------|--------------|----------|
| FEAT-00008 | HTTP/SSE Gateway (parent) | PM | — | P2 |
| FEAT-00010 | Agent OnEvent observer seam (internal/agent) | DEV | — | P2 |
| FEAT-00011 | GATEWAY_* config keys | DEV | — (additive to existing internal/config) | P2 |
| FEAT-00012 | internal/server HTTP layer (auth + SSE + healthz + Handler seam) | DEV | FEAT-00010, FEAT-00011 | P2 |
| FEAT-00013 | internal/runner per-request runner (real composition; SDK-free HTTP kept in internal/server) | DEV | FEAT-00010, FEAT-00011, FEAT-00012 | P2 |
| FEAT-00014 | cmd/gateway entrypoint | DEV | FEAT-00011, FEAT-00012, FEAT-00013 | P2 |
| FEAT-00015 | Next.js UI + token proxy (follow-up) | DEV | FEAT-00008 (frozen SSE contract) | P2 |

## Global Status

**DONE** — FEAT-00008 completed the full Flow 1 lifecycle on 2026-08-07: Design (TDD + ADRs 00002–00006) → BA spec → DoR PASS → `/plan` APPROVED → Implement (FEAT-00010..00014) → Test (TEST-00009, GW1–GW9 9/9 PASS) → QC sign-off (no blockers). `go build`/`go vet`/`go test ./...` all green. CLI (`cmd/server`) unchanged. Follow-ups: FEAT-00015 (Next.js UI, DEFERRED), token rotation, rate limiting, pooled MCP transport.

## DoR — FEAT-00008 (2026-08-07, PM)

- [x] 5-digit ID present: `FEAT-00008`
- [x] Linked to FEAT/REQ reference: TDD §6 + spec `docs/specs/api/gateway.md`
- [x] BA spec + AC completed: `docs/specs/api/gateway.md` (endpoints, errors, security)
- [x] Acceptance criteria listed: GW1–GW9 in TDD §6
- [x] Design/TDD + ADR present: `FEAT-00008_gateway_tdd.md` + ADR-0002..0006
- [x] Dependencies identified: sub-task graph above
- [x] Priority confirmed: P2 (S2)
- [ ] Estimate ≤ 8 story points: **not estimated yet** (needs grooming; TDD §10 open questions pending)
