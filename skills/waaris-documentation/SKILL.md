---
name: waaris-documentation
description: Maintain Waaris living specifications, API/database/deployment docs, ADRs, task records, and skill documentation. Use whenever implementation, interfaces, operations, risks, or decisions change.
---

# Waaris Documentation

Treat `README.md` as the normative specification. Keep documentation concise, testable, and synchronized with code.

1. Identify affected documents: `.context.md`, `PLAN.md`, `PROGRESS.md`, `TASKS.md`, `DECISIONS.md`, `SECURITY.md`, `API_SPEC.md`, `DATABASE.md`, `DEPLOYMENT.md`, and `SKILLS.md`.
2. Record facts, states, owners, constraints, and decisions—not aspirational features as implemented behavior.
3. Update all required living documents before completing the task; add a dated progress entry and use ADRs for durable trade-offs.
4. Check paths, commands, endpoint/state names, and document links against the repository.
5. Keep advanced features explicitly marked deferred until their gate conditions are met.
