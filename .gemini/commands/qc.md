# /qc Commands

## /qc review {TASK-ID}
- **Purpose**: Code review — logic, naming, clean architecture compliance
- **Input**: Task ID hoặc PR link
- **Output**: Review comments ([MUST] / [SUGGEST] / [QUESTION] / [NITPICK])
- **Prerequisite**: DEV ✅ in Registry
- **Example**: `/qc review FEAT-00005` → "Review shipping calculation code"

## /qc security
- **Purpose**: Security audit — IDOR, injection, data exposure check (STRIDE)
- **Input**: Task ID hoặc feature area
- **Output**: Security findings with severity, mapped to STRIDE categories
- **Follows**: `rules/security.md` + SKILL_QC §3
- **Example**: `/qc security FEAT-00005` → "Audit order deletion endpoint"

## /qc spec {TASK-ID}
- **Purpose**: Spec & Architecture compliance — verify code matches BA spec + Architect TDD
- **Input**: Task ID
- **Output**: Compliance report (Code ↔ Spec ↔ TDD alignment per layer)
- **Example**: `/qc spec FEAT-00005` → "Does implementation match api_spec.md exactly?"

## /qc audit {TASK-ID}
- **Purpose**: Full cross-reference audit — Code vs Spec vs Test vs Registry (all layers)
- **Input**: Task ID
- **Output**: Alignment report: ✅ match / ❌ mismatch per layer
- **Example**: `/qc audit API-00012` → "Code matches spec? Tests cover AC? Registry consistent?"

## /qc perf
- **Purpose**: Performance review — DB queries, N+1, bundle size, SLO compliance
- **Input**: Feature area hoặc specific files
- **Output**: Performance findings with improvement suggestions, SLO delta
- **Follows**: SKILL_ARCHITECT §8 SLO targets
- **Example**: `/qc perf` → "Check N+1 queries in order listing"

## /qc sign-off {TASK-ID}
- **Purpose**: Final approval — verify all gates pass, mark QC ✅ in Registry
- **Input**: Task ID (DEV ✅ and TEST ✅ required)
- **Output**: QC ✅ + Registry status → DONE, OR REJECT message with violations
- **Example**: `/qc sign-off FEAT-00005` → "Final quality gate — approve or reject"
