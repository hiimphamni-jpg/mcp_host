# API Specification — MCP Host Gateway

> Owner: BA | Last Updated: 2026-08-07
> Related Tasks: FEAT-00008 (HTTP/SSE Gateway)

## Modules

| Module | Endpoints | File |
|--------|-----------|------|
| Chat Gateway (v1) | `POST /v1/chat`, `GET /healthz` | [gateway.md](specs/api/gateway.md) |

## Changelog

| Version | Date | Change |
|---------|------|--------|
| v1 | 2026-08-07 | Initial gateway contract (FEAT-00008) |

## Related Documents

- `docs/business-logic.md` — gateway user stories and business rules
- `docs/tdd/FEAT-00008_gateway_tdd.md` — technical design and GW1..GW9
- `docs/adr/ADR-0002-http-sse-transport.md` … `ADR-0006-agent-observation-seam.md`