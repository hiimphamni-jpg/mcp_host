# /devops Commands

## /devops deploy {env}
- **Purpose**: Deploy application to specified environment
- **Input**: Environment (`staging` / `production`)
- **Output**: Deployment status, health check results, deploy announcement
- **Prerequisite**: QC sign-off (QC ✅) for production; CI passing for staging
- **Follows**: SKILL_DEVOPS §5 Deployment Checklist
- **Example**: `/devops deploy staging` → "Deploy develop branch to staging"

## /devops rollback
- **Purpose**: Rollback to previous version (fastest safe layer first)
- **Input**: Target SHA or "previous"; severity if incident-triggered
- **Output**: Rollback execution + health check verification
- **Layers**: Feature flag → Container → DOWN migration → DB snapshot
- **Example**: `/devops rollback` → "Rollback production to previous image SHA"

## /devops pipeline
- **Purpose**: Design or update CI/CD pipeline configuration
- **Input**: Pipeline type (PR / staging / production) + change description
- **Output**: Updated workflow files (GitHub Actions / GitLab CI)
- **Follows**: SKILL_DEVOPS §2 + `rules/git-workflow.md §5`
- **Example**: `/devops pipeline` → "Add Trivy security scan to PR pipeline"

## /devops infra
- **Purpose**: Infrastructure as Code — provision or update infrastructure
- **Input**: Resource description + environment
- **Output**: IaC plan (`terraform plan`) + review, then apply after approval
- **Follows**: SKILL_DEVOPS §4 IaC Checklist
- **Example**: `/devops infra` → "Provision staging Redis instance"

## /devops incident {severity}
- **Purpose**: Activate incident response playbook (SEV-1 / SEV-2 / SEV-3)
- **Input**: Incident description + severity level
- **Output**: Incident channel setup, role assignment, 5-step response execution
- **Follows**: SKILL_DEVOPS §9 Incident Response Playbook
- **Example**: `/devops incident SEV-1` → "Production API down — activate response"

## /devops postmortem {INC-ID}
- **Purpose**: Write post-mortem after incident resolution
- **Input**: Incident ID + timeline notes
- **Output**: Post-mortem document (timeline, root cause, impact, action items as OPS-xxxxx)
- **Follows**: SKILL_DEVOPS §9 Post-Mortem Template
- **Example**: `/devops postmortem INC-20250327` → "Write SEV-2 post-mortem"

## /devops monitor
- **Purpose**: Setup or review monitoring, alerting, and observability stack
- **Input**: Service/metric to monitor
- **Output**: Monitoring config + SLO-based alert rules
- **Follows**: SKILL_DEVOPS §8 SLO thresholds
- **Example**: `/devops monitor` → "Setup error rate and latency alerts"

## /devops secrets
- **Purpose**: Secret management — rotate, audit, or add secrets
- **Input**: Secret name or audit scope
- **Output**: Secret management actions + updated `.env.example`
- **Follows**: SKILL_DEVOPS §10 Secret Management
- **Example**: `/devops secrets rotate` → "Rotate payment gateway API keys"
