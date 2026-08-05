---
name: devops-skill
description: >
  CI/CD pipeline design, deployment strategy, infrastructure management, incident response, and monitoring.
  Use this skill whenever the user is: setting up or reviewing CI/CD pipelines, deploying to any environment,
  managing Docker/containers, writing infrastructure-as-code, handling incidents, configuring monitoring/alerting,
  managing secrets, planning a release, reviewing deployment checklists, or setting up observability.
  Trigger on phrases like "deploy", "pipeline", "CI/CD", "Docker", "staging", "rollback", "incident",
  "health check", "monitoring", "alert", "infra", "Dockerfile", "environment config", or "release plan".
---

# 🚀 DevOps Skill — Deployment & Operations

---

## Quick Reference

| Section | When to use |
|---------|-------------|
| [1. Environment Strategy](#1-environment-strategy) | Planning env setup, config parity |
| [2. CI/CD Pipeline Design](#2-cicd-pipeline-design) | Building or reviewing pipelines |
| [3. Container & Docker Standards](#3-container--docker-standards) | Writing Dockerfiles, image strategy |
| [4. Infrastructure as Code (IaC)](#4-infrastructure-as-code-iac) | Infra provisioning, config management |
| [5. Deployment Checklist](#5-deployment-checklist) | Before every production deploy |
| [6. Rollback Strategy](#6-rollback-strategy) | When a deploy goes wrong |
| [7. Database Migration Deployment](#7-database-migration-deployment) | Releasing schema changes safely |
| [8. Monitoring & Observability](#8-monitoring--observability) | Alerting, logging, tracing |
| [9. Incident Response Playbook](#9-incident-response-playbook) | SEV-1/2/3 handling |
| [10. Secret Management](#10-secret-management) | Secrets, env vars, rotation |
| [11. Cost & Resource Management](#11-cost--resource-management) | Infrastructure cost awareness |

---

## 1. Environment Strategy

| Environment | Purpose | Deploy Trigger | DB |
|-------------|---------|---------------|----|
| **Local** | Developer experiments | Manual | Docker Compose |
| **Staging** | QA testing, demo, pre-release validation | Auto on merge to `develop` | Staging DB (mirror prod schema) |
| **Production** | Live users | Manual trigger + approval | Production DB (backup before deploy) |

### Environment Parity Rules
- Staging must mirror production in: env var structure, secrets format, Docker image, runtime config
- **No "works on staging but not prod"** — parity violations are treated as infrastructure bugs
- All environment-specific values in env vars — **no hardcoded environment conditionals in application code**
- Staging database refreshed from production dump (anonymized) at least monthly

### Deploy Time Rules
- ❌ No production deploys after **4:00 PM** local time
- ❌ No production deploys on **Fridays** or day before public holidays without on-call confirmation
- ✅ On-call engineer must be available and reachable during production deploy window

---

## 2. CI/CD Pipeline Design

### PR Pipeline (every pull request — must pass before review)
```
Lint → Type Check → Unit Tests → Build Check → Security Scan (Trivy / npm audit)
```
- Target: **< 15 minutes** total
- All checks blocking — no merge with red CI
- No `[skip ci]` on production-bound branches (ref: `rules/git-workflow.md §5`)

### Staging Pipeline (auto on merge to `develop`)
```
All Tests → Build Docker Image → Tag with git SHA → Push to Registry →
Deploy to Staging → Smoke Test → Notify #deploys
```

### Production Pipeline (manual trigger, gated)
```
QA Sign-off Verified →
Build Docker (same SHA as staging) →
Snapshot DB (<1h before) →
Run Migrations →
Deploy (Blue/Green or Rolling) →
Health Check (all instances) →
Monitor 30 min →
Notify #deploys + Stakeholders
```

### CI Pipeline Rules
- Docker images tagged with **git SHA** — never `latest` tag in staging/prod
- Production build uses the **exact same image** promoted from staging (no rebuild)
- Security scan must find **0 HIGH/CRITICAL CVEs** before merge
- Coverage report published on every CI run (fail if coverage decreases)

---

## 3. Container & Docker Standards

### Dockerfile Best Practices
```dockerfile
# ✅ Multi-stage build — minimize image size
FROM golang:1.23-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download                   # Cache deps layer separately
COPY . .
RUN CGO_ENABLED=0 go build -o server ./cmd/server

FROM alpine:3.20
RUN apk --no-cache add ca-certificates tzdata
WORKDIR /app
COPY --from=builder /app/server .
EXPOSE 8080
USER nobody                           # Non-root user
ENTRYPOINT ["./server"]
```

### Image Rules
- Base images: use **specific version tags** (`alpine:3.20`) — never `latest`
- **Non-root user** in final image (`USER nobody` / `USER node`)
- **No secrets in Docker image** — pass via environment at runtime
- `.dockerignore` must exclude: `.git`, `node_modules`, `.env`, `*.test`
- Scan images with **Trivy** before push to registry

### Docker Compose (local dev)
```yaml
# One command to run entire stack
docker compose up -d

# Services: app + postgres + redis (if applicable)
# Health checks defined for all services
# Volumes for persistent data (DB)
```

---

## 4. Infrastructure as Code (IaC)

### Principles
- **Everything as code** — no manual infrastructure changes in staging/prod
- Changes to infra go through the same PR review process as application code
- IaC changes require **Tech Lead review** before apply
- All infra state stored remotely (Terraform state in S3/GCS, not local)

### IaC Checklist (per change)
- [ ] Change planned and reviewed (`terraform plan` output attached to PR)
- [ ] Blast radius assessed — what breaks if this fails?
- [ ] Rollback procedure documented
- [ ] No hardcoded credentials or IPs
- [ ] Resource naming follows convention: `{project}-{env}-{resource}` (e.g., `myapp-prod-db`)
- [ ] Tags/labels applied: `environment`, `team`, `cost-center`

### Environment Config Management
```
config/
├── .env.example      # Committed — all keys with empty values + comments
├── .env.local        # Not committed — local dev override
└── infra/
    ├── staging.env   # Non-secret staging config (committed)
    └── prod.env      # Non-secret prod config (committed)
# Secrets: injected at runtime from secret manager
```

---

## 5. Deployment Checklist

### Pre-Deploy (Production)
- [ ] All CI checks pass on the deploy SHA ✅
- [ ] QC sign-off received (QC column → ✅ in Registry)
- [ ] Database snapshot taken < 1 hour before deploy
- [ ] DOWN migration tested on staging if schema change included
- [ ] Rollback procedure confirmed and ready
- [ ] Feature flags configured for major features (can disable without redeploy)
- [ ] On-call engineer available and notified
- [ ] Deploy window confirmed: not after 4pm, not Friday/pre-holiday
- [ ] Stakeholders notified (if customer-facing change)

### Post-Deploy Verification (first 30 minutes)
- [ ] All instances passing health check (`GET /health` → `200 OK`)
- [ ] Error rate stable (< 1% of baseline)
- [ ] Response latency stable (p99 < 500ms, ref: SKILL_ARCHITECT §8 SLOs)
- [ ] Critical user flows manually verified
- [ ] Logs show no unexpected errors
- [ ] Notify `#deploys` channel with deploy summary

### Deploy Announcement Format
```markdown
🚀 **Deploy: {ENV}** — {FEAT-XXXXX} {description}
- **SHA**: `abc1234`
- **By**: {name}
- **Time**: 2025-03-27 15:00 ICT
- **Migrations**: Yes / No
- **Rollback**: `docker rollback {project}-api {prev-sha}`
```

---

## 6. Rollback Strategy

### Layers (fastest → safest, try in order)
| # | Method | Time | When |
|---|--------|:----:|------|
| 1 | **Feature Flag** | < 1 min | Feature-gated code path |
| 2 | **Container Rollback** | < 5 min | Bad code, no schema change |
| 3 | **DOWN Migration** | < 10 min | Schema change is reversible |
| 4 | **DB Snapshot Restore** | < 30 min | Data corruption, irreversible migration |

### Rollback Rules
- Docker images tagged with git SHA — rollback = redeploy previous SHA
- **Every migration MUST have a tested DOWN** — no one-way migrations without explicit sign-off
- Container rollback **does NOT require code review** — IC decides alone
- DB snapshot restore requires EM approval (data may be lost since snapshot)
- After rollback: create SEV-2 incident, do not re-deploy without root cause identified

---

## 7. Database Migration Deployment

### Deployment Order (expand/contract pattern)
```
1. Take DB snapshot
2. Deploy migration (UP) — backward-compatible only
3. Verify schema: check table structure, indexes, constraints
4. Deploy new application code (reads both old + new columns)
5. Monitor 30 min for migration-related errors
6. [Later sprint] Deploy contract migration (remove old columns)
```

### Migration Safety Rules (from `rules/database.md`)
- Backward-compatible changes only — code must work with both old and new schema simultaneously
- **Never rename or drop** a column in the same release as the code that uses the new name
- Large table migrations (>1M rows) → use `batched updates`, never lock the full table
- Always test migration on a **staging DB restore from production** (not clean schema)
- DOWN migration tested on staging before production deploy

### Migration Failure Response
```
1. STOP application deploy immediately
2. Assess: can DOWN migration run safely?
3. If yes: run DOWN migration → restore to previous state
4. If no: escalate to Tech Lead + DBA, suspend traffic if data integrity at risk
5. Document in post-mortem
```

---

## 8. Monitoring & Observability

### Observability Stack (Three Pillars)
| Pillar | Tool | Purpose |
|--------|------|---------|
| **Metrics** | Prometheus + Grafana | System & business metrics |
| **Logs** | Loki / CloudWatch / Seq | Structured log aggregation |
| **Traces** | Jaeger / Tempo | Distributed request tracing |

### SLO-Based Alert Thresholds (ref: SKILL_ARCHITECT §8)
| Metric | Warning | Critical | Action |
|--------|---------|----------|--------|
| Error rate (5xx) | > 0.5% for 2 min | > 1% for 5 min | Page on-call |
| API p99 latency | > 300ms for 5 min | > 500ms for 5 min | Investigate |
| API p50 latency | > 80ms for 5 min | > 100ms for 10 min | Review queries |
| CPU usage | > 70% for 10 min | > 85% for 10 min | Scale out |
| Memory usage | > 75% for 10 min | > 90% for 5 min | Investigate leak |
| Disk usage | > 80% | > 90% | Add storage |
| DB connections | > 70% of pool max | > 85% | Scale pool / DB |

### Structured Logging Standard
```json
{
  "level": "error",
  "timestamp": "2025-03-27T15:00:00Z",
  "service": "order-service",
  "trace_id": "trace_20250327_abc123",
  "message": "failed to lock session",
  "error": "repo find session: connection timeout",
  "session_id": "sess-uuid-here"
}
```
- **Never log**: passwords, tokens, card numbers, PII
- Log retention: **30 days** standard, **90 days** for security events
- Structured JSON only — no plain text logs in production

### Health Check Endpoint (required on every service)
```json
GET /health
{
  "status": "ok",
  "version": "v1.2.3",
  "uptime": "12h30m",
  "dependencies": {
    "database": "ok",
    "redis": "ok"
  }
}
```
- Response must include dependency health (DB, cache, etc.)
- Must respond in < 100ms — if slow, liveness probe may kill instance

---

## 9. Incident Response Playbook

### Severity Levels
| Level | Description | Acknowledge SLA | Resolve SLA |
|-------|-------------|:---------------:|:-----------:|
| **SEV-1** | Production down, data loss, security breach | ≤ 15 min | ≤ 4 hours |
| **SEV-2** | Core feature broken, major degradation | ≤ 30 min | ≤ 8 hours |
| **SEV-3** | Minor degradation, workaround exists | ≤ 4 hours | Within sprint |
| **SEV-4** | Cosmetic, low impact | Next business day | Backlog |

### 5-Step Response (SEV-1 / SEV-2)
```
1. DETECT & DECLARE
   → Create incident channel: #inc-YYYYMMDD-{short-desc}
   → Tag on-call + Engineering Manager
   → Don't try to fix alone silently — always declare first

2. ASSIGN ROLES
   → Incident Commander (IC): coordinate, NOT write code
   → Tech Lead: investigate and execute root cause fix
   → Comms Lead: update stakeholders every 30 minutes

3. MITIGATE FIRST, FIX SECOND
   → Feature flag OFF → Container rollback → Scale up
   → Stop the bleeding before root cause analysis
   → "Good enough now" > "perfect later"

4. COMMUNICATE (every 30 min until resolved)
   → Update status page with clear factual language
   → Internal: current status + next action + ETA
   → External: impact description (no technical jargon)
   → Blameless: "what failed" — never "who failed"

5. POST-MORTEM (within 48 hours for SEV-1, 72 hours for SEV-2)
   → Timeline + Root Cause + Customer Impact + Action Items (owners + deadlines)
   → Action items tracked as OPS-xxxxx tasks in Registry
   → Blameless culture: systems failed, not people
```

### Post-Mortem Template
```markdown
# Post-Mortem: {INC-ID} — {short title}
**Date**: {date} | **SEV**: SEV-{1/2} | **Duration**: Xh Xmin

## Timeline
| Time | Event |
|------|-------|
| 15:00 | Alert triggered — error rate > 1% |
| 15:08 | On-call paged, incident declared |
| 15:25 | Root cause identified: N+1 query on new endpoint |
| 15:35 | Container rollback deployed |
| 15:40 | Error rate back to baseline — incident resolved |

## Root Cause
[Technical explanation — what specifically failed]

## Customer Impact
[What users experienced, duration, estimated affected users]

## What Went Well
- On-call responded in 8 minutes (within SLA)

## What Could Improve
- No alert existed for DB connection pool saturation

## Action Items
| Action | Owner | Due | Task ID |
|--------|-------|-----|---------|
| Add DB connection pool alert | DevOps | Next sprint | OPS-00003 |
| Add integration test for the query | Dev | This sprint | FEAT-00012 |
```

---

## 10. Secret Management

### Rules
- **Never commit secrets** to git — `.env` in `.gitignore`, always
- `.env.example` committed with **empty values + comments** for every key
- All secrets injected at runtime via environment variables or secret manager
- Different secrets per environment — dev ≠ staging ≠ prod (no sharing)
- Rotate secrets: **quarterly** or immediately after team member departure or suspected exposure

### Secret Categories
| Category | Storage | Examples |
|----------|---------|---------|
| App secrets | Secret Manager (Vault / AWS SM) | JWT secret, API keys, webhook tokens |
| DB credentials | Secret Manager | DB host, user, password |
| External APIs | Secret Manager | Payment gateway keys, SMS provider |
| Non-secret config | Environment file (committed) | `LOG_LEVEL`, `PORT`, `APP_ENV` |

### Exposure Response
```
1. Immediately rotate the exposed secret
2. Audit access logs for the exposure window
3. Revoke old secret in all systems
4. Deploy new secret to all environments
5. Document in security incident log
```

---

## 11. Cost & Resource Management

### Resource Naming Convention
```
{project}-{env}-{resource-type}
Examples:
  myapp-prod-db          → production PostgreSQL
  myapp-staging-api      → staging API service
  myapp-prod-redis       → production cache
```

### Cost Awareness Rules
- Review infrastructure costs **monthly** — unexplained spikes require explanation
- **Right-size before scale** — profile before adding instances
- Unused resources (idle for >30 days) must be decommissioned
- Staging environment scaled down outside working hours (weeknights/weekends)

### Scaling Decision Framework
```
Step 1: Profile — is the bottleneck CPU, Memory, DB, or Network?
Step 2: Optimize — can a query/algorithm fix resolve it? (ref: SKILL_ARCHITECT §8)
Step 3: Cache — can Redis/CDN absorb the load?
Step 4: Scale Up — bigger instance (vertical)
Step 5: Scale Out — more instances (horizontal, stateless services only)
```
