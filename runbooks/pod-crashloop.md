# SRE Incident Runbook: Pod CrashLoopBackOff

## 1. Alert Summary
- **Alert**: `PodCrashLooping`
- **Severity**: Critical
- **Trigger**: Container has restarted $>2$ times within 5 minutes.

---

## 2. Diagnosis Steps

1. **Locate Crashing Pods**:
   ```bash
   kubectl get pods -n production | grep -E "CrashLoopBackOff|Error"
   ```

2. **Inspect Previous Container Logs**:
   ```bash
   kubectl logs -n production <pod-name> --previous --tail=50
   ```

3. **Check Termination Exit Code & Reason**:
   ```bash
   kubectl describe pod -n production <pod-name> | grep -A 10 "Last State:"
   ```
   - **Exit Code 1**: Application exception / unhandled error.
   - **Exit Code 137**: Process killed by SIGKILL (OOMKiller or manual termination).
   - **Exit Code 143**: Process terminated gracefully by SIGTERM.

4. **Verify ConfigMap & Secret References**:
   Ensure all referenced secrets exist:
   ```bash
   kubectl get secret -n production
   kubectl get configmap -n production
   ```

---

## 3. Remediation

1. **If Due to Bad Deployment**:
   Trigger instant GitOps rollback in ArgoCD or revert the commit in Git:
   ```bash
   git revert HEAD
   git push origin main
   ```
2. **If Due to Database Connectivity**:
   Test database reachability from an ephemeral pod:
   ```bash
   kubectl run test-net --rm -i --tty --image=alpine -- sh -c "nc -zv aws-0-ap-south-1.pooler.supabase.com 5432"
   ```
