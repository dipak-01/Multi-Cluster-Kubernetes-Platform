# Resilix

> **Resilix** is a hands-on Site Reliability Engineering (SRE) and cloud platform workbench built for resilience, real-time SLO error budget tracking, automated Linux node hardening, GitOps delivery, chaos engineering, and custom automation tooling.

---

## 🚀 Quick Navigation

- [📖 Complete End-to-End Engineering Guide](docs/PLATFORM_GUIDE.md)
- [🎯 Service Level Objectives (SLOs) & Error Budgets](docs/SLO.md)
- [🛠️ Ansible Node Automation & Hardening Guide](docs/ANSIBLE_GUIDE.md)
- [🏛️ Architecture Overview](ARCHITECTURE.md)
- [🔥 Chaos Engineering Suite & Postmortems](chaos/CHAOS_EXPERIMENTS.md)
- [📋 Linux Troubleshooting Runbook](runbooks/linux-troubleshooting.md)
- [📦 Workload Helm Charts](helm/)
- [🔄 GitOps ArgoCD Manifests](argocd/)
- [🛡️ Kubernetes Governance & Network Policies](kubernetes/)
- [🤖 SRE Platform CLI (`platformctl`)](automation/go/)

---

## 📁 Repository Structure

```text
├── applications/               # Workload source code
│   ├── api/                    # Node.js Express REST API backend (PID 1, SIGTERM, /healthz)
│   └── frontend/               # React 19 + Vite frontend (unprivileged NGINX port 8080)
├── ansible/                    # Configuration management & OS hardening
│   ├── inventory/              # Hosts & group variables
│   ├── playbooks/              # site.yml, bootstrap, hardening, containerd, k8s, monitoring
│   └── roles/                  # linux, hardening, containerd, kubernetes, node-exporter
├── terraform/                  # Infrastructure as Code (AWS VPC, EC2, IAM)
├── kubernetes/                 # Cluster governance (Namespaces, NetworkPolicies, RBAC)
├── helm/                       # Parameterized Helm charts (api, frontend)
├── argocd/                     # GitOps application manifests & projects (self-healing)
├── observability/              # Observability stack
│   ├── prometheus/             # Prometheus stack values, ServiceMonitors, SLO rules
│   ├── grafana/                # SRE Golden Signals & SLO Dashboard
│   └── alertmanager/           # Multi-window burn-rate alerts & Slack integration
├── chaos/                      # Automated failure injection & resilience test scripts
├── automation/                 # SRE Automation CLI (platformctl in Go)
├── runbooks/                   # Incident management & troubleshooting procedures
├── docs/                       # Technical specifications, SLO math, and operations guides
├── .github/workflows/          # CI/CD pipelines (Helm lint, Docker builds, GitOps sync)
└── docker-compose.yaml         # Local development orchestration
```

---

## 🕹️ Platform Operations Runbook (How to Run Next Time)

### 1. Start the Platform
To boot up the Kubernetes cluster and networking:
```bash
# 1. Start Minikube cluster
minikube start

# 2. Ensure ingress and metrics addons are enabled
minikube addons enable ingress
minikube addons enable metrics-server

# 3. Verify Minikube IP matches /etc/hosts (default: 192.168.49.2)
minikube ip
```

*(Ensure `/etc/hosts` contains: `192.168.49.2 api.local frontend.local argocd.local grafana.local prometheus.local alertmanager.local`)*

---

### 2. Check System Health & SLOs (1-Second Check)
Use the built-in Go CLI `platformctl` to instantly check cluster, resource saturation, and SLO status:

```bash
# Check control plane, node readiness, CPU/RAM utilization, and pod states
platformctl cluster health

# Check real-time 30-day availability, latency SLIs, and multi-window burn rates
platformctl slo status
```

---

### 3. Service Dashboard Directory

All platform components are accessible via Ingress:

| Service | URL | Credentials | Purpose |
|---|---|---|---|
| **Frontend Application** | [http://frontend.local](http://frontend.local) | — | Production React UI connecting to `/api` |
| **Backend API** | [http://api.local/healthz](http://api.local/healthz) | — | Node.js Express REST API |
| **ArgoCD (GitOps)** | [http://argocd.local](http://argocd.local) | `admin` / `oKaYh2mjLyEOwqfn` | Continuous delivery & cluster reconciliation |
| **Grafana** | [http://grafana.local](http://grafana.local) | `admin` / `prom-operator` | SRE Golden Signals & SLO Dashboard |
| **Prometheus Rules** | [http://prometheus.local/rules](http://prometheus.local/rules) | — | Recording rules & burn-rate alerts |
| **Alertmanager** | [http://alertmanager.local](http://alertmanager.local) | — | Incident routing & notification engine |

---

### 4. Common Daily Workflows

#### A. GitOps Application State (ArgoCD)
ArgoCD continuously syncs from the `helm/` directory:
```bash
# Check application sync & health state
kubectl get applications -n argocd

# Inspect production pods (3 API replicas + 3 Frontend replicas)
kubectl get pods -n production
```

#### B. Running Chaos & Resilience Tests
Run automated failure injections to test platform fault-tolerance:
```bash
# Test pod kill resilience under continuous load (100% success rate, 0 dropped requests)
./chaos/pod-failure/test-pod-resilience.sh

# Test automated OOMKill recovery (137 exit code handling)
./chaos/resource-exhaustion/simulate-memory-leak.sh

# Test bad release deployment rejection
./chaos/bad-deployment/test-bad-release.sh
```

#### C. Ansible Node Automation (Milestone 3)
Automated Linux baseline, OS hardening, containerd CRI, and monitoring agent deployment:
```bash
# Syntax-check all roles and playbooks
ansible-playbook -i ansible/inventory/hosts.ini ansible/playbooks/site.yml --syntax-check

# Ping local host driver
ansible -i "localhost," -c local all -m ping -e ansible_become=false

# Execute dry-run check across cluster inventory
ansible-playbook -i ansible/inventory/hosts.ini ansible/playbooks/site.yml --check
```

#### D. Standalone Docker Compose (Fallback Mode)
If running workloads outside Kubernetes:
```bash
docker compose up -d
docker compose down
```

---

### 5. Shutting Down Cleanly
When you finish your work or demo:
```bash
minikube stop
```
*(All configurations, pods, dashboards, and metrics remain safely persisted for your next session).*

---

## 🛠️ SRE CLI Reference (`platformctl`)

The custom SRE command-line utility built in Go ([`automation/go/`](automation/go/)):

```bash
# Cluster health & resource metrics
platformctl cluster health
platformctl cluster nodes

# Application management
platformctl application status api-production
platformctl application logs api-production 100
platformctl application restart api-production

# Incident diagnosis (scans pods, crash loops, OOMKills, events)
platformctl incident diagnose production

# SLO & error budget tracking
platformctl slo status

# GitOps deployment history & rollbacks
platformctl deployment history api-production
platformctl deployment rollback api-production 1
```

---

## 📊 Milestone Roadmap

- [x] **Phase 0**: Project Scaffolding, Workload Packaging & Helm Charts
- [x] **Milestone 1**: Linux Performance Runbook & Systems Engineering ([Runbook](runbooks/linux-troubleshooting.md))
- [ ] **Milestone 2**: Terraform AWS Infrastructure Modules (`network`, `compute`, `security`)
- [x] **Milestone 3**: Ansible Automation & Hardening ([Guide](docs/ANSIBLE_GUIDE.md))
- [x] **Milestone 4**: Kubernetes Cluster Provisioning & Zero-Trust Network Policies
- [x] **Milestone 5**: ArgoCD GitOps Deployment & Reconciliations
- [x] **Milestone 6**: Observability Stack (Prometheus, Grafana, Alertmanager)
- [x] **Milestone 7**: SRE SLOs, Error Budgets & Alert Rules ([docs/SLO.md](docs/SLO.md))
- [x] **Milestone 8**: Chaos Engineering Suite ([Experiments](chaos/CHAOS_EXPERIMENTS.md))
- [x] **Milestone 9**: `platformctl` Operational CLI in Go
- [ ] **Milestone 10**: Full Production Incident Simulation
