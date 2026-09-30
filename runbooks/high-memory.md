# SRE Incident Runbook: High Memory & OOMKill Prevention

## 1. Alert Summary
- **Alert**: `PodHighMemoryUsage`
- **Severity**: Critical
- **Trigger**: Container working set memory exceeds 85% of configured limit for >3 minutes.
- **Risk**: Linux cgroup OOM killer termination (`OOMKilled`, exit code 137).

---

## 2. Triage & Investigation Steps

1. **Identify Memory-Intensive Pods**:
   ```bash
   kubectl top pods -n production --sort-by=memory
   ```

2. **Check Container Termination History**:
   ```bash
   kubectl get pods -n production -o custom-columns=NAME:.metadata.name,RESTARTS:.status.containerStatuses[*].restartCount,LAST_REASON:.status.containerStatuses[*].lastState.terminated.reason,EXIT_CODE:.status.containerStatuses[*].lastState.terminated.exitCode
   ```

3. **Inspect Application Memory Profile (Node.js Heap)**:
   Check application logs for heap allocation spikes:
   ```bash
   kubectl logs -n production <pod-name> --tail=100 | grep -i -E "heap|out of memory|fatal"
   ```

---

## 3. Remediation Procedures

1. **Perform Rolling Restart (Immediate mitigation for leak)**:
   ```bash
   kubectl rollout restart deployment api-production -n production
   ```

2. **Increase Memory Limit**:
   If workload legitimate data set increased, update `resources.limits.memory` in `helm/api/values-production.yaml`:
   ```yaml
   resources:
     limits:
       memory: 1.5Gi
   ```

3. **Post-Mortem**: Document root cause in `chaos/CHAOS_EXPERIMENTS.md`.
