# /architect Commands

## /architect design {feature}
- **Purpose**: Write Tech Design Document (TDD) cho feature lớn
- **Input**: Feature name hoặc FEAT-xxxxx
- **Output**: TDD document trong `/docs/tdd/`
- **Example**: `/architect design FEAT-00010` → "Design settlement system"

## /architect adr
- **Purpose**: Create Architecture Decision Record
- **Input**: Decision topic
- **Output**: ADR document trong `/docs/adr/`
- **Example**: `/architect adr` → "ADR: Choose WebSocket over SSE"

## /architect review
- **Purpose**: Review system design hoặc breaking changes
- **Input**: PR link, TDD, hoặc design proposal
- **Output**: Design review comments + approval/rejection
- **Example**: `/architect review` → "Review DB schema redesign"

## /architect debt
- **Purpose**: Tech debt assessment và planning
- **Input**: Feature area hoặc full codebase
- **Output**: Tech debt inventory, priority ranking, resolution plan
- **Example**: `/architect debt` → "Assess tech debt in order system"

## /architect capacity
- **Purpose**: Capacity planning và scalability review
- **Input**: Current metrics + growth projection
- **Output**: Bottleneck analysis, scaling recommendations
- **Example**: `/architect capacity` → "Will system handle 10x users?"

## /architect diagram
- **Purpose**: Generate system/sequence/ER diagrams
- **Input**: System area hoặc feature flow
- **Output**: Mermaid diagram
- **Example**: `/architect diagram` → "Draw sequence diagram for payment"
