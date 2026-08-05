# Debug Report - FEAT-00001 Step 1

## Bug Report

### Symptom

The Step 1 verification command removed `google.golang.org/genai` immediately after adding it, causing the dependency-baseline verification to fail.

### Repro Steps

1. Run `go get google.golang.org/genai@latest`.
2. Run `go mod tidy` while no package imports the SDK.
3. Inspect `go.mod`; the SDK requirement is absent.

### Root Cause

`go mod tidy` removes module requirements that are not reachable from source imports. The approved bootstrap scope prohibits a Gemini provider import, so the original plan's requirement to retain the SDK and run `go mod tidy` in the same step was contradictory.

### Fix

Revise Step 1 to retain the approved direct dependency baseline and verify the resolved module with `go list -m google.golang.org/genai`; do not run `go mod tidy` until a task introduces an SDK import.

### Regression Protection

The revised plan explicitly documents why tidy is intentionally deferred and verifies the exact retained SDK module during Step 1.

### Verification

- Command: `go get google.golang.org/genai@latest; go list -m google.golang.org/genai`
- Expected: the resolved SDK module and version are listed.
- Result: NOT RUN; revised plan requires approval before build resumes.
