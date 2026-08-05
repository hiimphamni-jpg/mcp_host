---
description: Assess whether a task is ready to deploy without deploying it.
agent: devops
---

Perform a release-readiness assessment for: $ARGUMENTS

Read the task Registry entry, test and QC artifacts, relevant deployment configuration, and git status. Check approval status, required test evidence, QC sign-off, migration and rollback safety, configuration/secrets requirements, monitoring, and release risks. Return `READY`, `NOT READY`, or `READY WITH EXCEPTIONS`, with evidence and the next required command. Do not deploy or edit files.
