# /test Commands

## /test cases
- **Purpose**: Viết test cases (TEST-xxxxx) từ Acceptance Criteria
- **Input**: Task ID hoặc feature spec
- **Output**: Structured test cases (positive + negative + boundary)
- **Example**: `/test cases FEAT-005` → "Viết test cases cho shipping"

## /test api
- **Purpose**: Test API endpoints — validate JSON schema, status codes
- **Input**: Endpoint URL hoặc Task ID
- **Output**: Test results (pass/fail) với response evidence
- **Example**: `/test api POST /api/v1/sessions` → "Test create session"

## /test ui
- **Purpose**: Test UI/UX — responsive, mobile, interactions
- **Input**: Page/component name
- **Output**: Visual test results, screenshots
- **Example**: `/test ui` → "Test order form trên mobile"

## /test e2e
- **Purpose**: Run E2E automation cho full user flows
- **Input**: Flow name hoặc Task ID
- **Output**: E2E pass/fail report
- **Example**: `/test e2e` → "Run E2E: create → order → lock → pay"

## /test ws
- **Purpose**: Test WebSocket realtime sync giữa các clients
- **Input**: Event type hoặc flow
- **Output**: Sync verification results
- **Example**: `/test ws` → "Test realtime order updates"

## /test bug
- **Purpose**: Report bug vào Registry bug tracker
- **Input**: Bug details (steps, expected, actual)
- **Output**: BUG-xxxxx entry trong `REGISTRY.md`
- **Example**: `/test bug` → "Report: negative qty accepted"

## /test verify
- **Purpose**: Retest fixed bugs
- **Input**: BUG-xxxxx ID
- **Output**: Verified ✅ hoặc Reopened ❌
- **Example**: `/test verify BUG-00003` → "Retest negative qty fix"
