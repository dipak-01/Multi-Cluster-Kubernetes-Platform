# Platform Architecture & Design

## 1. System Overview

This repository implements a production-grade, multi-cluster Site Reliability Engineering (SRE) platform supporting hybrid cloud (AWS) and on-premise infrastructure.

```
                         ENGINEERS
                             │
                             ▼
                         ┌───────┐
                         │  Git  │
                         └───┬───┘
                             │
               ┌─────────────┴─────────────┐
               │                           │
               ▼                           ▼
          GitHub Actions                ArgoCD
               │                           │
        ┌──────┴──────┐                    │
        │             │                    │
      Tests         Build                  │
        │             │                    │
        ▼             ▼                    │
    Security      Container                │
      Scan         Registry                 │
                      │                     │
                      └──────────┐          │
                                 ▼          ▼
                         ┌───────────────────────┐
                         │ Kubernetes Platform   │
                         │                       │
                         │ AWS Cluster           │
                         │                       │
                         │ On-Prem Cluster       │
                         └───────────┬───────────┘
                                     │
                   ┌─────────────────┼────────────────┐
                   │                 │                │
                   ▼                 ▼                ▼
              Applications       Metrics            Logs
                   │                 │                │
                   │            Prometheus          Loki
                   │                 │                │
                   │                 ▼                │
                   │              Grafana             │
                   │                 │                │
                   └─────────────────┼────────────────┘
                                     ▼
                              Alertmanager
                                     │
                                     ▼
                             INCIDENT RESPONSE
                                     │
                         ┌───────────┴───────────┐
                         │                       │
                      Runbooks              platformctl
                         │                       │
                         └───────────┬───────────┘
                                     ▼
                               REMEDIATION
```

---

## 2. Directory Layout & Layer Responsibilities

1. **`applications/`**: Business application workloads (`api/` and `frontend/`).
2. **`terraform/`**: Infrastructure as Code for provisioning AWS VPC, Subnets, Security Groups, IAM, and EC2/EKS compute instances.
3. **`ansible/`**: Automated node configuration, OS hardening, `containerd` setup, kernel tuning, and monitoring agent deployment.
4. **`kubernetes/`**: Base manifests for cluster governance: Namespaces, RBAC, NetworkPolicies, and StorageClasses.
5. **`helm/`**: Parameterized Helm charts for deploying application services across environments (`values-dev.yaml`, `values-production.yaml`).
6. **`argocd/`**: Declarative GitOps application and project manifests managing cluster desired state.
7. **`observability/`**: Configurations for Prometheus metrics scraping, Alertmanager alert rules, Loki log collection, and Grafana dashboards.
8. **`automation/`**: Operational tooling (`platformctl`) for automated cluster diagnosis and self-healing.
9. **`runbooks/`**: Actionable SRE runbooks for production incident triage and remediation.

---

## 3. Workload Network & Security Boundaries

```
[ Internet Traffic ]
        │
        ▼
[ Ingress Controller (nginx) ]
        │
   ┌────┴─────────────────────────┐
   │                              │
   ▼ (Path: /)                    ▼ (Path: /api)
[ Frontend Pods ]               [ API Pods ]
   │ (Port 80)                    │ (Port 3000)
   │                              │
   └──────────────┬───────────────┘
                  ▼
         [ PostgreSQL Database ]
            (AWS / Supabase)
```

- **Zero-Trust NetworkPolicies**: Default deny on ingress and egress. Explicitly whitelisted communication channels between Frontend $\rightarrow$ API and API $\rightarrow$ Database.
- **Least-Privilege Execution**: All application containers run as non-root users (`USER node` in API and non-privileged Nginx) with dropped Linux kernel capabilities (`ALL`).
