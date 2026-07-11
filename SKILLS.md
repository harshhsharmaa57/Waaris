# Waaris Engineering Skills

## Selection outcome

The OpenAI skills catalog was reviewed on 2026-07-11. It did not provide direct production workflow skills for Go, React, Kubernetes, Terraform, PostgreSQL, Solidity, OpenAPI, dependency management, or broad Git/PR automation. Only two non-overlapping, high-value skills were installed globally:

| Skill | Source | Why selected | Use it for |
|---|---|---|---|
| `security-best-practices` | OpenAI curated | General secure-engineering review complements project-specific controls | Security-sensitive implementation and review |
| `security-threat-model` | OpenAI curated | Structured threat analysis is essential for false-execution and privacy risks | Architecture/security changes and milestone gates |

The following catalog skills were deliberately not installed: deployment skills are vendor-specific; Figma/Notion/media skills do not match the stack; ASP.NET/WinUI are irrelevant; narrow GitHub helper skills do not replace the requested review/commit/PR workflow; and general-purpose UI/goal skills provide little project-specific value. The catalog had no experimental directory at review time.

## Versioned custom skills

Project-owned skills are versioned under `skills/` so their rules evolve with the repository. They intentionally merge overlapping requests: project memory handles planning, task tracking, commits, review readiness, debugging/refactoring discipline, and test evidence; backend handles API, PostgreSQL, contract tests, and dependency updates; DevOps handles Docker, Kubernetes, Terraform, GitHub Actions, and release operations.

| Skill | Covers | Use it for |
|---|---|---|
| `waaris-architecture` | Architecture review | Cross-component design, state/data flow, ADR decisions |
| `waaris-project-memory` | Planning, project memory, Git workflow | Every task start/end; plans, progress, task and commit synchronization |
| `waaris-security` | Security review | Threats, identity, liveness, privacy, secrets, sensitive integrations |
| `waaris-documentation` | Documentation maintenance | All specifications, ADRs, and living-document consistency |
| `waaris-backend` | Go, React contracts, PostgreSQL, API, testing, debugging, dependencies | Foundation services and safe-MVP application changes |
| `waaris-blockchain` | Solidity and chain/oracle review | Only after the documented prerequisite gate is complete |
| `waaris-devops` | Docker, Kubernetes, Terraform, GitHub Actions | CI/CD, infrastructure, observability, recovery, and release controls |

## Future-session use

1. Read `.context.md` before any work and `README.md` before implementation changes.
2. Use the matching `skills/<name>/SKILL.md` as the project workflow. Use multiple skills when a change crosses boundaries: for example, backend + security + documentation.
3. Treat custom skills as versioned source of truth. To make them auto-discoverable in a Codex installation, copy or link each selected `skills/waaris-*` directory into `$CODEX_HOME/skills` (normally `~/.codex/skills`) after checking it out; do not overwrite a differently versioned global skill silently.
4. Validate each custom skill after edits with `C:\Users\Abhishek\.codex\skills\.system\skill-creator\scripts\quick_validate.py <skill-directory>`.
5. Update this file when skills are added, removed, consolidated, installed, or materially changed.

## Last updated

2026-07-11 — initial curated selection and Waaris-specific skill suite.
