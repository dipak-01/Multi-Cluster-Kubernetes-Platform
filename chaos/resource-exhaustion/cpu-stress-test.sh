#!/usr/bin/env bash
# ==============================================================================
# Chaos Experiment: CPU Saturation & HPA Autoscaling
# Objective: Saturate pod CPU to verify HorizontalPodAutoscaler (HPA) triggers scale-up
# ==============================================================================

set -euo pipefail

NAMESPACE="production"
DEPLOYMENT="api-production"
INGRESS_HOST="platform.local"
INGRESS_IP="192.168.49.2"
CONCURRENCY=30
DURATION_SECONDS=45

echo "=========================================================="
echo "🔥 Starting Chaos Experiment: CPU Load Injection"
echo "Target Deployment: ${DEPLOYMENT}"
echo "Current Replicas & HPA Status:"
kubectl get hpa -n "${NAMESPACE}"
kubectl get pods -n "${NAMESPACE}" -l app.kubernetes.io/name=api
echo "=========================================================="

echo -e "\n[1/3] Generating high-concurrency traffic to induce CPU saturation..."
# Run background workers sending concurrent requests
for i in $(seq 1 $CONCURRENCY); do
  (
    end_time=$((SECONDS + DURATION_SECONDS))
    while [ $SECONDS -lt $end_time ]; do
      curl -s -o /dev/null -H "Host: ${INGRESS_HOST}" "http://${INGRESS_IP}/api" || true
    done
  ) &
done

echo -e "\n[2/3] Traffic load running for ${DURATION_SECONDS}s. Monitoring HPA metrics..."
for i in $(seq 1 6); do
  sleep 7
  echo -e "\n--- Snapshot at ${i}x7s ---"
  kubectl get hpa -n "${NAMESPACE}"
  kubectl get deployment "${DEPLOYMENT}" -n "${NAMESPACE}" -o wide
done

wait

echo -e "\n[3/3] Post-Load Scale Status:"
kubectl get hpa -n "${NAMESPACE}"
kubectl get pods -n "${NAMESPACE}" -l app.kubernetes.io/name=api

echo -e "\n=========================================================="
echo "✅ CPU Saturation Experiment Complete! Verify Grafana Dashboard."
echo "=========================================================="
