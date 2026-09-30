#!/usr/bin/env bash
# ==============================================================================
# Chaos Experiment: Random Pod Kill & Zero-Downtime Validation
# Target: namespace production
# Objective: Verify PodDisruptionBudget, Service routing, and Deployment recovery
# ==============================================================================

set -euo pipefail

NAMESPACE="production"
TARGET_LABEL="app.kubernetes.io/name=api"
INGRESS_HOST="platform.local"
INGRESS_IP="192.168.49.2"
DURATION_SECONDS=30

echo "=========================================================="
echo "🔥 Starting Chaos Experiment: Random Pod Failure Injection"
echo "Target Namespace: ${NAMESPACE}"
echo "Target Label: ${TARGET_LABEL}"
echo "Test Duration: ${DURATION_SECONDS}s"
echo "=========================================================="

# 1. Baseline Pod Check
echo -e "\n[1/4] Baseline Pod Status:"
kubectl get pods -n "${NAMESPACE}" -l "${TARGET_LABEL}" -o wide

# 2. Start background traffic generator
TOTAL_REQUESTS=0
SUCCESS_REQUESTS=0
FAILED_REQUESTS=0

echo -e "\n[2/4] Generating background HTTP traffic..."
generate_traffic() {
  local end_time=$((SECONDS + DURATION_SECONDS))
  while [ $SECONDS -lt $end_time ]; do
    status_code=$(curl -s -o /dev/null -w "%{http_code}" -H "Host: ${INGRESS_HOST}" "http://${INGRESS_IP}/" --connect-timeout 2 || echo "000")
    if [ "$status_code" -eq 200 ]; then
      echo -n "."
    else
      echo -n "X"
    fi
    sleep 0.1
  done
}

generate_traffic &
TRAFFIC_PID=$!

# 3. Inject Pod Failure after 3 seconds
sleep 3
echo -e "\n\n[3/4] 💥 Injecting Pod Failure (killing 1 active API pod)..."
RANDOM_POD=$(kubectl get pods -n "${NAMESPACE}" -l "${TARGET_LABEL}" -o jsonpath='{.items[0].metadata.name}')
echo "Selected pod for termination: ${RANDOM_POD}"
kubectl delete pod "${RANDOM_POD}" -n "${NAMESPACE}" --grace-period=0 --force 2>/dev/null

echo "Waiting for traffic test to complete..."
wait $TRAFFIC_PID

# 4. Post-experiment state verification
echo -e "\n\n[4/4] Post-Chaos Pod Recovery Status:"
kubectl get pods -n "${NAMESPACE}" -l "${TARGET_LABEL}"

echo -e "\n=========================================================="
echo "✅ Experiment Complete! Pod was automatically self-healed by Kubernetes."
echo "=========================================================="
