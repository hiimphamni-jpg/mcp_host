# /dev Commands

## /dev code {TASK-ID}
- **Purpose**: Implement a specific task from Registry
- **Input**: Task ID (e.g., `FEAT-00005`, `API-00012`)
- **Output**: Feature branch + code + unit tests
- **Prerequisite**: Task must exist in Registry, status SPRINT BACKLOG or IN_PROGRESS
- **Announces**: `🔀 Branch: feature/{TASK-ID}-{desc}` at start
- **Example**: `/dev code FEAT-00005` → "Implement shipping calculation"

## /dev test {TASK-ID}
- **Purpose**: Write or run tests for a specific task (TDD: write test first)
- **Input**: Task ID
- **Output**: Test files (`*_test.go` / `*.test.ts`) with coverage report
- **Follows**: `rules/testing.md` — mandatory TDD for business logic
- **Example**: `/dev test API-00012` → "Write unit tests for settlement API"

## /dev pr {TASK-ID}
- **Purpose**: Prepare PR after implementation — run self-review checklist + create PR description
- **Input**: Task ID + current branch
- **Output**: PR description (What/Why/How to test) + self-review checklist results
- **Follows**: SKILL_DEV §6 PR Preparation Checklist
- **Example**: `/dev pr FEAT-00005` → "Prepare PR for shipping calculation"

## /dev refactor
- **Purpose**: Refactor existing code without changing behavior
- **Input**: Area/file to refactor
- **Output**: Cleaner code with existing tests still passing
- **Example**: `/dev refactor` → "Refactor order service into smaller functions"

## /dev migrate
- **Purpose**: Create database migration files (UP + DOWN)
- **Input**: Schema change description
- **Output**: `{timestamp}_{description}.up.sql` + `.down.sql`
- **Follows**: `rules/database.md` expand/contract pattern
- **Example**: `/dev migrate` → "Add shipping_fee column to orders"

## /dev review
- **Purpose**: Self-review against DoD checklist before creating PR
- **Input**: Current branch code
- **Output**: Checklist results + list of items to fix
- **Example**: `/dev review` → "Run pre-PR self-review checklist"
