---
description: Select and run appropriate checks for a scope without changing source files.
agent: general
---

Verify this scope: $ARGUMENTS

Inspect the repository to identify its language, test runner, linters, and build commands. Run the smallest relevant checks first, then broader checks when justified. Report each exact command and PASS, FAIL, or NOT_RUN result. Do not modify source files; if a check fails, summarize the failure and recommend `/fix`.
