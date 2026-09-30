# SRE Incident Runbook: High CPU Saturation

## 1. Alert Summary
- **Alert**: `PodHighCPUUsage`
- **Severity**: Warning
- **Trigger**: Container CPU usage exceeds 85% of configured resource limit for >3 minutes.

---

## 2. Triage & Investigation Steps

1. **Identify Affected Pods**:
   ```bash
   kubectl top pods -n production --sort-by=cpu
   ```

2. **Verify Node Saturation**:
   ```bash
   kubectl top nodes
   ```

3. **Check Container Process Activity**:
   ```bash
   kubectl exec -it <pod-name> -n production -- top -b -n 1
   ```

4. **Verify HPA Behavior**:
   ```bash
   kubectl get hpa -n production
   # Check if HPA has scaled to maxReplicas:
   kubectl describe hpa api-production -n production
   ```

---

## 3. Immediate Remediation

1. **Temporary Manual Scale Up (if HPA is maxed out)**:
   ```bash
   kubectl scale deployment api-production -n production --replicas=<current_replicas + 2>
   ```

2. **Adjust Resource Limits (if traffic is legitimate)**:
   Update `helm/api/values-production.yaml` with increased `resources.limits.cpu` and commit to Git.

3. **Rate Limiting (if DDoS / traffic anomaly)**:
   Enable rate limiting on Ingress via annotations:
   ```yaml
   nginx.ingress.kubernetes.io/limit-rps: "50"
   ```
