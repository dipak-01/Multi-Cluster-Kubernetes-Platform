# SRE Incident Runbook: High API Latency & 5xx Error Rate

## 1. Alert Summary
- **Alerts**: `HighAPIErrorRate` / `HighAPILatency`
- **Severity**: Critical / Warning
- **Trigger**: HTTP 5xx error rate $>5\%$ or p95 latency $>500$ms over 2-3 minutes.

---

## 2. Investigation Workflow

1. **Check Real-Time Ingress Request Status**:
   ```bash
   kubectl logs -n ingress-nginx -l app.kubernetes.io/name=ingress-nginx --tail=50 | grep -E ' 50[0-9] '
   ```

2. **Check Backend API Response Latency**:
   ```bash
   curl -w "@curl-format.txt" -o /dev/null -s -H "Host: platform.local" "http://192.168.49.2/api"
   ```

3. **Check Database Connection Pool Saturation**:
   Inspect backend pod logs for pool timeout errors:
   ```bash
   kubectl logs -n production -l app.kubernetes.io/name=api --tail=100 | grep -i "pool"
   ```

---

## 3. Remediation

1. **Restart Application Pods**:
   ```bash
   kubectl rollout restart deployment api-production -n production
   ```
2. **Scale Deployment**:
   ```bash
   kubectl scale deployment api-production -n production --replicas=6
   ```
3. **Verify Health Probes**:
   ```bash
   curl -I http://192.168.49.2/healthz -H "Host: platform.local"
   ```
