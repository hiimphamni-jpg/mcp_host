---
name: Git Workflow
description: Branching strategy, commit conventions, PR process, and CI/CD rules
---

# 🔀 Git Workflow Standards

> **Principle:** Clean history, fast reviews, safe deployments. No cowboy commits.

---

## 1. Branching Strategy (Gitflow Simplified)

| Branch | Purpose | Protected | Merge Into |
|--------|---------|:---------:|-----------|
| `main` | Production code, always deployable | ✅ | — |
| `develop` | Integration branch, latest dev code | ✅ | `main` (via release) |
| `feature/{ticket}-{desc}` | New feature | ❌ | `develop` |
| `fix/{ticket}-{desc}` | Non-urgent bug fix | ❌ | `develop` |
| `hotfix/{ticket}-{desc}` | Urgent production fix | ❌ | `main` + `develop` |
| `release/v{X.Y.Z}` | Release candidate | ❌ | `main` + `develop` |

### Branch Naming Rules
- Format: `{type}/{TICKET-ID}-{short-description}`
- Examples:
  - `feature/FEAT-00005-calculate-shipping`
  - `fix/BUG-00012-null-participant-name`
  - `hotfix/BUG-00020-payment-crash`
- Use lowercase, hyphens only (no underscores, no spaces)
- **No direct commits** to `main` or `develop` — blocked by branch protection

---

## 2. Commit Message Convention (Conventional Commits)

### Format
```
{type}({scope}): {description}

[optional body]

[optional footer]
```

### Types
| Type | When | Example |
|------|------|---------|
| `feat` | New feature | `feat(auth): add QR login flow` |
| `fix` | Bug fix | `fix(scanner): handle null payload` |
| `refactor` | Internal restructure, no behavior change | `refactor(session): extract payment logic` |
| `test` | Add or update tests | `test(order): add batch create unit test` |
| `docs` | Documentation only | `docs: update API README` |
| `chore` | Build, deps, config, CI | `chore: bump next.js to 16.1` |
| `perf` | Performance improvement | `perf(query): add index on session_id` |
| `style` | Formatting only (no logic change) | `style: fix linting warnings` |
| `ci` | CI/CD configuration | `ci: add staging deploy workflow` |

### Breaking Changes
- Add `!` after type: `feat!: migrate to new auth API`
- Or add footer: `BREAKING CHANGE: removed /v1/auth endpoint`

### Rules
- Subject line ≤ 72 characters
- Use imperative mood: "add feature" not "added feature"
- No period at end of subject
- Body wraps at 80 characters

---

## 3. Pull Request Process

### PR Size Limit
- **Maximum 400 lines changed** (excluding auto-generated files)
- Larger PRs must be split or have written justification
- **Hard rule** — PRs >400 lines will be sent back for splitting

### PR Description Template
Every PR must include:
```markdown
## 🎯 What & Why
Link: [FEAT-00005] / [BUG-00012]
Brief description of what changed and why.

## 🔧 How to Test
1. Step-by-step instructions
2. For local verification

## 📸 Screenshots / Video
(Required for UI changes)

## ✅ Checklist
- [ ] Tests pass locally
- [ ] No lint errors
- [ ] No `console.log` / `fmt.Println`
- [ ] API docs updated (if applicable)
- [ ] Migration tested (if applicable)
```

### Reviewer Assignment
- **Minimum 1 approver** for standard changes
- **Minimum 2 approvers** for:
  - Core business logic changes
  - Security-related changes
  - Database schema changes
  - Breaking API changes
- **Tech Lead required** for architecture-level changes

### Review SLA
- First review within **1 business day**
- Re-review after fixes within **4 hours**

### Review Comment Prefixes
| Prefix | Meaning | Blocking? |
|--------|---------|:---------:|
| `[MUST]` | Must fix before merge | ✅ |
| `[SUGGEST]` | Improvement suggestion | ❌ |
| `[QUESTION]` | Need clarification | ⚠️ |
| `[NITPICK]` | Minor style/formatting | ❌ |

---

## 4. Merge Strategy

- **Squash and merge** for feature/fix branches into `develop`
  - Clean, linear commit history
  - All commits in PR compressed to single commit
- **Merge commit** for release branches into `main`
  - Preserves release boundary
- **Rebase** — only for local cleanup before pushing

---

## 5. CI Pipeline (Required Checks)

All checks must pass before merge is allowed:

### PR Pipeline
```
Lint → Type Check → Unit Tests → Build Check → Security Scan
```

### Staging Pipeline (on merge to `develop`)
```
All Tests → Build Docker → Push Registry → Deploy Staging → Smoke Test
```

### Production Pipeline (manual trigger)
```
QA Sign-off → Build → Deploy (Blue/Green) → Health Check → Notify
```

### Rules
- **All CI checks blocking** — no merge with red checks
- **No skipping checks** — no `[skip ci]` on production-bound branches
- CI must complete in **< 15 minutes** for PR pipeline

---

## 6. Branch Protection Rules

### `main` + `develop`
- Require pull request before merging
- Require status checks to pass
- Require at least 1 approval
- No force push
- No deletion
- Include administrators

---

## 7. Release Process

1. Create `release/vX.Y.Z` branch from `develop`
2. Only bug fixes allowed on release branch
3. QA testing on release branch
4. Merge to `main` + tag with `vX.Y.Z`
5. Merge back to `develop`
6. Deploy from `main` tag

### Semantic Versioning
- `MAJOR.MINOR.PATCH`
- **MAJOR**: Breaking changes
- **MINOR**: New features, backward-compatible
- **PATCH**: Bug fixes
