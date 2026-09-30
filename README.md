# Production-Grade Multi-Cluster Kubernetes Platform

A production-style Site Reliability Engineering (SRE) platform capable of running critical workloads across **AWS and On-Premise infrastructure**.

---

## 🚀 Quick Navigation

- [📖 Complete End-to-End Engineering Guide](docs/PLATFORM_GUIDE.md)
- [Architecture Overview](ARCHITECTURE.md)
- [Roadmap & Specifications](devops_project.md)
- [Linux Troubleshooting Runbook](runbooks/linux-troubleshooting.md)
- [Chaos Engineering Suite & Postmortems](chaos/CHAOS_EXPERIMENTS.md)
- [Workload Helm Charts](helm/)
- [GitOps ArgoCD Manifests](argocd/)
- [Kubernetes Governance Manifests](kubernetes/)

---

## 📁 Repository Structure

```text
├── applications/               # Workload source code
│   ├── api/                    # Node.js Express REST API backend
│   └── frontend/               # React + Vite frontend
├── terraform/                  # Infrastructure as Code (AWS VPC, EC2, IAM)
├── ansible/                    # Configuration management & node hardening
├── kubernetes/                 # Cluster manifests (Namespaces, NetworkPolicies, RBAC)
├── helm/                       # Helm charts for workloads (api, frontend)
├── argocd/                     # GitOps application manifests & projects
├── observability/              # Prometheus, Grafana, Alertmanager, Loki
├── automation/                 # platformctl SRE CLI tool
├── runbooks/                   # Incident management & troubleshooting procedures
├── docs/                       # DR, Security, and Incident Response documentation
├── .github/workflows/          # CI/CD pipelines
└── docker-compose.yaml         # Local development orchestration
```

---

## 🛠️ Local Development

### Prerequisites
- Docker & Docker Compose
- Node.js 22+
- Helm v3

### Running via Docker Compose
```bash
docker compose up -d --build
```
- Frontend: `http://localhost:5173`
- Backend API: `http://localhost:3000`
- Health check: `http://localhost:3000/healthz`

### Linting & Testing Helm Charts
```bash
helm lint helm/api
helm lint helm/frontend

# Render production manifests
helm template api helm/api -f helm/api/values-production.yaml
helm template frontend helm/frontend -f helm/frontend/values-production.yaml
```

---

## 📊 Milestone Roadmap

- [x] **Phase 0**: Project Scaffolding, Workload Packaging & Helm Charts
- [x] **Milestone 1**: Linux Performance Runbook & Systems Engineering ([Runbook](runbooks/linux-troubleshooting.md))
- [ ] **Milestone 2**: Terraform AWS Infrastructure Modules (`network`, `compute`, `security`)
- [ ] **Milestone 3**: Ansible Automation & Hardening (containerd, sysctl, node-exporter)
- [ ] **Milestone 4**: Kubernetes Cluster Provisioning & Network Policies
- [ ] **Milestone 5**: ArgoCD GitOps Deployment & Reconciliations
- [ ] **Milestone 6**: Observability Stack (Prometheus, Grafana, Alertmanager, Loki)
- [ ] **Milestone 7**: SRE SLOs, Error Budgets & Alert Rules
- [ ] **Milestone 8**: Chaos Engineering Suite
- [ ] **Milestone 9**: `platformctl` Operational CLI
- [ ] **Milestone 10**: Full Production Incident Simulation
