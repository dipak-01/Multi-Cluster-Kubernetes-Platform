# Chaos Engineering & Incident Postmortems

This document records the hypotheses, execution traces, telemetry observations, and remediation procedures for the automated Chaos Engineering suite running on the Kubernetes Platform.

---

## Experiment 1: Random Pod Termination & Zero-Downtime Validation

- **Experiment Script**: [`chaos/pod-failure/test-pod-resilience.sh`](pod-failure/test-pod-resilience.sh)
- **Target**: `api-production` in namespace `production`
- **Blast Radius**: 1 active container replica
- **Hypothesis**: Terminating an active pod under load will cause zero user-facing HTTP 5xx errors or dropped connections because Kubernetes Service endpoints and `PodDisruptionBudget` ensure traffic shifts immediately to remaining Ready replicas.
- **Expected Result**: 100% request success rate during deletion; new pod scheduled and ready within 30 seconds.
- **Actual Result**:
  - `api-production-578cc8cb99-8tnrw` terminated abruptly (`--grace-period=0`).
  - Total HTTP requests during disruption: 215.
  - Successful responses (HTTP 200): 215 (100%).
  - Failed responses (HTTP 5xx / dropped): 0 (0%).
  - Replacement pod `api-production-578cc8cb99-2s7qd` reached `1/1 Running` within 27 seconds.
- **Lessons Learned**:
  - Setting `readinessProbe` with `failureThreshold: 2` and `periodSeconds: 5` ensures healthy pod endpoints are populated before ingress traffic is directed to new containers.
  - Non-zero `minAvailable: 2` in `PodDisruptionBudget` prevents involuntary cluster operations from causing service outages.

---

## Experiment 2: Memory Leak & Linux cgroup OOMKill

- **Experiment Script**: [`chaos/resource-exhaustion/simulate-memory-leak.sh`](resource-exhaustion/simulate-memory-leak.sh)
- **Target**: `memory-leak-simulator` in namespace `production`
- **Blast Radius**: Isolated test pod with 64Mi memory limit
- **Hypothesis**: A container allocating unbounded memory will be terminated by the Linux cgroup memory controller with exit code 137 (`SIGKILL`) when it exceeds its `resources.limits.memory`, without affecting neighboring pods.
- **Expected Result**: State transitions to `OOMKilled`; Prometheus detects restart event.
- **Actual Result**:
  - Memory consumption grew rapidly until 64Mi threshold.
  - Linux kernel cgroup OOM killer triggered termination.
  - Pod status reported `OOMKilled` (Exit code 137).
  - Neighboring production pods remained unaffected.
- **Lessons Learned**:
  - Strict resource limits are vital to prevent a single leaking container from starving the host node and triggering node-level memory pressure.
  - Alert rule `PodHighMemoryUsage` (>85%) fires prior to OOMKill, giving on-call SREs an alert window to scale or remediate before process termination.

---

## Experiment 3: GitOps Drift & Bad Release Self-Healing

- **Experiment Script**: [`chaos/bad-deployment/test-bad-release.sh`](bad-deployment/test-bad-release.sh)
- **Target**: `api-production` Deployment in namespace `production`
- **Blast Radius**: Production deployment command specification
- **Hypothesis**: If an unauthorized manual patch or broken configuration is injected into the cluster, ArgoCD will detect configuration drift and automatically reconcile back to the desired Git commit state within its sync interval (`selfHeal: true`).
- **Expected Result**: Drift detected; ArgoCD triggers automated sync and restores the original Deployment manifest.
- **Actual Result**:
  - Broken container command was injected into `Deployment/api-production`.
  - ArgoCD controller detected diff against `main` Git tree.
  - Self-healing reconciled the resource, reverting the manual modification back to the Git source of truth.
- **Lessons Learned**:
  - Never treat `kubectl patch` or `kubectl edit` as production deployment tools.
  - Enabling `selfHeal: true` and `prune: true` guarantees cluster state matches Git commits, preventing configuration drift across multi-cluster environments.

---

## Experiment 4: CPU Saturation & Horizontal Pod Autoscaling (HPA)

- **Experiment Script**: [`chaos/resource-exhaustion/cpu-stress-test.sh`](resource-exhaustion/cpu-stress-test.sh)
- **Target**: `api-production` Deployment in namespace `production`
- **Blast Radius**: Cluster CPU resource allocation
- **Hypothesis**: High-concurrency incoming requests will saturate API CPU usage beyond the 70% target utilization, causing the Horizontal Pod Autoscaler to dynamically increase replica count up to `maxReplicas: 10`.
- **Expected Result**: Metric server detects CPU spike; HPA scales deployment.
- **Remediation & Runbook**: [`runbooks/high-cpu.md`](../runbooks/high-cpu.md)
