---
name: Database Standards
description: PostgreSQL schema design, migration, and query optimization rules
---

# 🗄️ Database Standards

> **Principle:** The database is the last line of defense for data integrity. Schema changes are permanent — treat them like surgery.

---

## 1. Primary Keys

- **UUID recommended** — prefer UUID v7 (time-sortable, no collisions across services) or UUID v4
- Default: `id UUID PRIMARY KEY DEFAULT gen_random_uuid()`
- **Never** auto-increment integers for public-facing IDs (leaks total count, enables enumeration)
- Internal junction tables may use composite PKs
- If project uses a different PK strategy, document it in an ADR (`docs/adr/`)

---

## 2. Naming Conventions

| Element | Convention | Example |
|---------|-----------|---------|
| Table | `snake_case`, **plural** | `order_items`, `sessions` |
| Column | `snake_case` | `created_at`, `total_amount` |
| Index | `idx_{table}_{column(s)}` | `idx_orders_session_id` |
| Unique constraint | `uq_{table}_{column(s)}` | `uq_users_email` |
| Foreign key | `fk_{table}_{ref_table}` | `fk_orders_session_id` |
| Enum type | `snake_case` | `session_status` |

### Column Naming Rules
- Timestamps: `created_at`, `updated_at`, `deleted_at`
- Foreign keys: `{referenced_table_singular}_id` — `session_id`, `user_id`
- Boolean: `is_` or `has_` prefix — `is_active`, `has_paid`
- Amount/Money: suffix `_amount` — `total_amount`, `shipping_amount`

---

## 3. Schema Design

### Required Columns (every table)
```sql
id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
```

### Soft Delete (when applicable)
```sql
deleted_at TIMESTAMPTZ DEFAULT NULL
```
- Add index: `CREATE INDEX idx_{table}_deleted_at ON {table}(deleted_at) WHERE deleted_at IS NULL`
- All queries must include `WHERE deleted_at IS NULL` (enforce via repository layer)

### Data Types
| Use Case | Type | NOT |
|----------|------|-----|
| Money/Amount | `BIGINT` (store in smallest unit, e.g., VND) | `FLOAT`, `DECIMAL` for VND |
| Status/Enum | `VARCHAR(20)` or PostgreSQL `ENUM` | `INT` magic numbers |
| JSON data | `JSONB` | `JSON` (no indexing) |
| Text | `TEXT` (no arbitrary limits) | `VARCHAR(255)` without reason |
| Flags | `BOOLEAN NOT NULL DEFAULT FALSE` | `INT` 0/1 |

---

## 4. Migration Rules

### File Naming
```
{timestamp}_{description}.up.sql
{timestamp}_{description}.down.sql
```
Example: `20250315143000_add_shipping_fee_to_orders.up.sql`

### Mandatory Principles
1. **Every UP migration must have a DOWN** — rollback must be possible
2. **Backward-compatible** (expand/contract pattern):
   - Phase 1 (expand): Add new column (nullable), deploy code that writes to both
   - Phase 2 (contract): After all code uses new column, remove old column
3. **No breaking changes in one step** — never rename or drop columns directly
4. **Test migrations** on a copy of production data before deploying
5. **Snapshot before deploy** — database backup < 1 hour before migration

### What Requires a Migration
| Change | Migration Required |
|--------|:-----------------:|
| Add table | ✅ |
| Add column | ✅ |
| Add index | ✅ |
| Rename column | ✅ (expand/contract) |
| Drop column | ✅ (contract phase) |
| Change column type | ✅ (expand/contract) |
| Insert seed data | ✅ |

---

## 5. Index Strategy

### When to Add Index
- Every foreign key column
- Columns used in `WHERE` clauses frequently
- Columns used in `ORDER BY`
- Columns used in `JOIN` conditions
- Composite index for multi-column queries (order matters: most selective first)

### When NOT to Add Index
- Tables with < 1000 rows (full scan is faster)
- Columns with very low cardinality (e.g., `boolean`)
- Write-heavy tables where read is rare

### Partial Index (PostgreSQL)
```sql
-- Only index non-deleted records
CREATE INDEX idx_sessions_status ON sessions(status) WHERE deleted_at IS NULL;

-- Only index active sessions
CREATE INDEX idx_sessions_active ON sessions(id) WHERE status = 'OPEN';
```

---

## 6. Query Rules

### Must Do
- **No `SELECT *`** — always specify columns
- **Use parameterized queries** — never string concatenation for SQL
- **Use transactions** for multi-table writes
- **Set query timeouts** via context (Go: `context.WithTimeout`)
- **Use `RETURNING`** clause instead of separate SELECT after INSERT/UPDATE

### Must Avoid
- **N+1 queries** — use `JOIN` or batch queries
- **Large `IN` clauses** (>100 items) — use temp table or `ANY(array)`
- **`LIKE '%prefix%'`** — use full-text search or trigram index
- **Locking entire tables** — use row-level locks

### Example
```sql
-- ✅ Good
SELECT id, nickname, total_amount
FROM participants
WHERE session_id = $1 AND deleted_at IS NULL
ORDER BY created_at ASC;

-- ❌ Bad
SELECT * FROM participants WHERE session_id = '...' ORDER BY created_at;
```

---

## 7. Connection Management

- **Connection pooling** mandatory (PgBouncer or driver-level pool)
- Max connections: configure based on `max_connections / number_of_instances`
- Idle timeout: 5 minutes
- Connection lifetime: 1 hour (prevent stale connections)

---

## 8. Data Integrity

- **Foreign keys** enforced at database level (not just application)
- **NOT NULL** by default — make nullable only when business requires it
- **CHECK constraints** for business rules: `CHECK (amount >= 0)`
- **UNIQUE constraints** at database level for uniqueness requirements
