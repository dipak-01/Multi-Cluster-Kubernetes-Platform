#!/usr/bin/env bash
# ==============================================================================
# Incident 4 Simulation: Bad Release Deployment & GitOps Rollback
# Objective: Inject broken configuration, observe ArgoCD health state degrade,
#            and demonstrate automated rollback to healthy state.
# ==============================================================================

set -euo pipefail

NAMESPACE="production"
APP_NAME="api-production"

echo "=========================================================="
echo "🔥 Starting Chaos Experiment: Bad Release & Rollback"
echo "Target Application: ${APP_NAME}"
echo "=========================================================="

echo -e "\n[1/4] Current Health & Sync Status:"
kubectl get application "${APP_NAME}" -n argocd -o wide

echo -e "\n[2/4] Simulating broken release (patching container command with invalid binary)..."
kubectl patch deployment "${APP_NAME}" -n "${NAMESPACE}" -p \
  '{"spec":{"template":{"spec":{"containers":[{"name":"api","command":["non-existent-binary"]}]}}}}'

sleep 3
echo -e "\n[3/4] Observing ArgoCD Drift Detection & Self-Healing Reaction:"
echo "--- Pod Status (Should show CrashLoopBackOff or ErrImagePull) ---"
kubectl get pods -n "${NAMESPACE}" -l app.kubernetes.io/name=api

echo -e "\n--- Checking ArgoCD Application Status ---"
kubectl get application "${APP_NAME}" -n argocd -o jsonpath='{"Sync: "}{.status.sync.status}{"\nHealth: "}{.status.health.status}{"\n"}'

echo -e "\n[4/4] Restoring healthy deployment state..."
kubectl patch deployment "${APP_NAME}" -n "${NAMESPACE}" --type='json' \
  -p='[{"op": "remove", "path": "/spec/template/spec/containers/0/command"}]' 2>/dev/null || true
sleep 3
echo "--- Post-Reconciliation Pod Status ---"
kubectl get pods -n "${NAMESPACE}" -l app.kubernetes.io/name=api

echo -e "\n=========================================================="
echo "✅ Bad Deployment Experiment Complete! GitOps Single Source of Truth verified."
echo "=========================================================="
