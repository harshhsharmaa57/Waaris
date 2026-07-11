---
name: waaris-devops
description: Plan, implement, test, review, or operate Waaris Docker, Kubernetes, Terraform, GitHub Actions, observability, secrets, deployment, CI/CD, dependency management, and incident-recovery workflows. Use for cloud-native platform and release work.
---

# Waaris DevOps

Read `.context.md`, `DEPLOYMENT.md`, `SECURITY.md`, and relevant ADRs.

1. Start with reproducible local Docker Compose and CI; use Terraform for environment infrastructure and Kubernetes manifests/charts for workloads.
2. Enforce least-privilege IAM, network policies, image/dependency/IaC/secret scans, immutable artifacts, SBOMs, and redacted observability.
3. Configure probes, requests/limits, rolling-release safeguards, transactional/idempotent workers, backups, restore tests, and explicit safe-hold behavior.
4. Do not place user vault keys, trustee shares, or sensitive content in CI variables, Terraform state, Kubernetes secrets, logs, metrics, or traces.
5. Validate plan/apply or manifests in a non-production environment, update runbooks/release gates, and make a focused commit.
