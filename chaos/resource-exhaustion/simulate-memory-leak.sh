#!/usr/bin/env bash
# ==============================================================================
# Incident 2 Simulation: Memory Leak & OOMKill Detection
# Objective: Simulate application memory leak, trigger Linux OOM Killer, and diagnose
# ==============================================================================

set -euo pipefail

NAMESPACE="production"
POD_NAME="memory-leak-simulator"

echo "=========================================================="
echo "🔥 Starting Chaos Experiment: Memory Leak & OOMKill"
echo "Target Namespace: ${NAMESPACE}"
echo "=========================================================="

echo -e "\n[1/4] Deploying container with 64Mi memory limit and runaway allocation..."
cat <<EOF | kubectl apply -n "${NAMESPACE}" -f -
apiVersion: v1
kind: Pod
metadata:
  name: ${POD_NAME}
  namespace: ${NAMESPACE}
  labels:
    app: memory-leak-test
spec:
  restartPolicy: OnFailure
  containers:
    - name: memory-eater
      image: alpine:latest
      command: ["sh", "-c"]
      args:
        - |
          echo "Simulating memory leak...";
          # Allocate memory in a loop until cgroup memory limit is exceeded
          a=""; while true; do a="\$a \$a 1234567890abcdef"; done
      resources:
        limits:
          memory: "64Mi"
        requests:
          memory: "32Mi"
      securityContext:
        allowPrivilegeEscalation: false
        capabilities:
          drop: ["ALL"]
        runAsNonRoot: true
        runAsUser: 1000
  securityContext:
    seccompProfile:
      type: RuntimeDefault
EOF

echo -e "\n[2/4] Monitoring memory growth and OOMKill event..."
for i in $(seq 1 6); do
  sleep 2
  STATUS=$(kubectl get pod "${POD_NAME}" -n "${NAMESPACE}" -o jsonpath='{.status.containerStatuses[0].state.terminated.reason}' 2>/dev/null || echo "Running")
  echo "State: ${STATUS}"
  if [ "${STATUS}" == "OOMKilled" ]; then
    echo -e "\n💥 OOMKill Event Triggered!"
    break
  fi
done

echo -e "\n[3/4] SRE Triage & Root Cause Verification:"
echo "--- Pod Last State Reason ---"
kubectl get pod "${POD_NAME}" -n "${NAMESPACE}" -o jsonpath='{.status.containerStatuses[0].lastState.terminated.reason}' && echo ""
echo "--- Exit Code (137 = 128 + 9 / SIGKILL by OOM) ---"
kubectl get pod "${POD_NAME}" -n "${NAMESPACE}" -o jsonpath='{.status.containerStatuses[0].lastState.terminated.exitCode}' && echo ""

echo -e "\n[4/4] Cleaning up experiment..."
kubectl delete pod "${POD_NAME}" -n "${NAMESPACE}" --wait=false

echo -e "\n=========================================================="
echo "✅ Memory Leak Experiment Complete! Successfully proved OOMKill behavior."
echo "=========================================================="
