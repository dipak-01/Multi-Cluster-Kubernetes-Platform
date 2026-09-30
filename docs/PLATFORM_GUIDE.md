# Resilix — Platform Engineering & SRE Operations Guide

A comprehensive architectural, operational, and engineering manual for **Resilix**, a hands-on Kubernetes SRE workbench.

---

## Table of Contents
1. [Platform Architecture & Design Philosophy](#1-platform-architecture--design-philosophy)
2. [Workload Layer: Frontend & Backend Applications](#2-workload-layer-frontend--backend-applications)
3. [Containerization & Multi-Stage Docker Builds](#3-containerization--multi-stage-docker-builds)
4. [Kubernetes Infrastructure & Cluster Governance](#4-kubernetes-infrastructure--cluster-governance)
5. [Application Packaging with Helm](#5-application-packaging-with-helm)
6. [GitOps Continuous Delivery with ArgoCD](#6-gitops-continuous-delivery-with-argocd)
7. [Observability, Alerting & SRE SLOs](#7-observability-alerting--sre-slos)
8. [Chaos Engineering & Incident Simulation Suite](#8-chaos-engineering--incident-simulation-suite)
9. [Operational Automation: The `platformctl` CLI Tool](#9-operational-automation-the-platformctl-cli-tool)
10. [End-to-End Operational Cheatsheet](#10-end-to-end-operational-cheatsheet)

---

## 1. Platform Architecture & Design Philosophy

The objective of this platform is not merely running containers on Kubernetes; it is building a **resilient, secure, observable, and self-healing SRE platform**.

```
                         DEVELOPERS / SREs
                                │
                  ┌─────────────┴─────────────┐
                  ▼                           ▼
          Git Commit / Push             platformctl CLI
                  │                           │
                  ▼                           ▼
        ┌───────────────────┐       ┌───────────────────┐
        │  GitHub Actions   │       │ Cluster Health &  │
        │  CI Verification  │       │ Auto-Diagnostics  │
        └─────────┬─────────┘       └───────────────────┘
                  │
                  ▼
        ┌───────────────────┐
        │ GitHub Repository │ (Single Source of Truth)
        └─────────┬─────────┘
                  │
                  │ Reconciliation Loop (Auto-Sync & Self-Heal)
                  ▼
        ┌────────────────────────────────────────────────────────┐
        │                   Kubernetes Cluster                   │
        │                                                        │
        │  ┌──────────────────────────────────────────────────┐  │
        │  │ Ingress Controller (NGINX)                       │  │
        │  │  - platform.local  -> Frontend (/) & API (/api)  │  │
        │  │  - argocd.local    -> ArgoCD Web Console         │  │
        │  │  - grafana.local   -> SRE Golden Signals         │  │
        │  │  - prometheus.local-> PromQL Query Engine        │  │
        │  └──────────┬───────────────────────────┬───────────┘  │
        │             │                           │              │
        │             ▼                           ▼              │
        │  ┌───────────────────────┐   ┌──────────────────────┐  │
        │  │ Namespace: production │   │ Namespace: argocd    │  │
        │  │ (PodSecurity:         │   │ (GitOps Engine)      │  │
        │  │  restricted)          │   │                      │  │
        │  │                       │   │ - Application Ctrl   │  │
        │  │ - Frontend (Nginx)    │   │ - Server & Repo Svc  │  │
        │  │ - API (Node.js)       │   └──────────────────────┘  │
        │  │ - HPAs & PDBs         │                             │
        │  │ - NetworkPolicies     │                             │
        │  └──────────┬────────────┘                             │
        │             │ Metrics                                  │
        │             ▼                                          │
        │  ┌──────────────────────────────────────────────────┐  │
        │  │ Namespace: observability                         │  │
        │  │ - Prometheus Operator & StatefulSet              │  │
        │  │ - Alertmanager (Golden Signals Alerts)           │  │
        │  │ - Grafana (SLO & Error Budget Dashboard)         │  │
        │  │ - Node Exporter & Kube State Metrics             │  │
        │  └──────────────────────────────────────────────────┘  │
        └────────────────────────────────────────────────────────┘
```

### Core Design Principles:
1. **GitOps as the Single Source of Truth**: No manual `kubectl apply` or `kubectl edit` in production. Any manual cluster drift is automatically detected and reverted by ArgoCD (`selfHeal: true`).
2. **Zero-Trust Security**: Default deny on network traffic, restricted PodSecurityStandards, non-root container users, dropped Linux kernel capabilities, and read-only filesystems.
3. **Actionable Observability**: Measure the **Four Golden Signals** (Latency, Traffic, Errors, Saturation). Alerts fire on user impact, not noisy raw thresholds.
4. **Resilience by Default**: Deployments use Pod anti-affinity, `PodDisruptionBudgets`, Horizontal Pod Autoscaling (HPA), and multi-replica redundancy to survive sudden node or container deaths without dropped requests.

---

## 2. Workload Layer: Frontend & Backend Applications

The workload running on this platform is a full-stack Pokémon application decoupled into a **RESTful backend API** and a **React Single Page Application (SPA)**.

### 2.1 Backend API (`applications/api`)
- **Runtime**: Node.js 22 LTS (ES Modules)
- **Framework**: Express.js
- **Database**: PostgreSQL (hosted on AWS / Supabase)
- **Primary Files**:
  - `src/index.js`: Express server bootstrap, middleware, probe endpoints, and signal listeners.
  - `src/routes/pokemonRoutes.js`: Router for Pokémon data endpoints.
  - `config/db.js`: Database client connection pool.

#### Critical SRE Hardening in Backend:
1. **Kubernetes Health Probes (`/healthz`)**:
   ```javascript
   app.get("/healthz", (req, res) => {
     res.status(200).json({ status: "healthy", timestamp: new Date().toISOString() });
   });
   ```
   Kubernetes kubelet periodically polls this endpoint. If the process freezes or crashes, kubelet stops routing traffic (readiness) or restarts the container (liveness).

2. **Graceful Shutdown (`SIGTERM` Handling)**:
   ```javascript
   process.on("SIGTERM", () => {
     console.log("SIGTERM signal received: closing HTTP server");
     server.close(() => {
       console.log("HTTP server closed cleanly");
       process.exit(0);
     });
   });
   ```
   When Kubernetes terminates a pod (during a rolling deployment or scale-down), it sends `SIGTERM`. If unhandled, in-flight HTTP requests are dropped. With this handler, Node.js finishes serving active requests before exiting cleanly.

---

### 2.2 Frontend Client (`applications/frontend`)
- **Runtime / Bundler**: React 19, Vite, TailwindCSS
- **Communication**: Axios client calling the API
- **Dynamic Endpoint Fallback**:
  ```javascript
  const backendUrl = import.meta.env.VITE_BACKEND_URL || "";
  const res = await axios.get(`${backendUrl}/api/pokemon`);
  ```
  By defaulting to an empty string `""`, the frontend seamlessly works behind a Kubernetes Ingress where `/api` is served on the same hostname (`platform.local/api`), completely eliminating Cross-Origin Resource Sharing (CORS) issues in production!

---

## 3. Containerization & Multi-Stage Docker Builds

Container security and image footprint directly impact vulnerability surfaces and cold-start scheduling latency.

### 3.1 Backend API Dockerfile (`applications/api/Dockerfile`)
```dockerfile
FROM node:22-alpine
WORKDIR /app

# 1. Dependency caching layer
COPY package*.json ./
RUN npm ci --omit=dev

# 2. Application source code
COPY . .

# 3. Security: Run as non-root user (node uid 1000)
RUN chown -R node:node /app
USER node

ENV PORT=3000
EXPOSE 3000

# 4. Run node directly as PID 1 (avoids npm wrapper swallowing SIGTERM)
CMD ["node", "src/index.js"]
```

#### Why this matters:
- `npm ci --omit=dev`: Excludes development dependencies (e.g. linters, test runners), reducing attack surface.
- `USER node`: Container does not run as root (`UID 0`). If an attacker finds a Remote Code Execution (RCE) vulnerability in an npm package, they are trapped in an unprivileged user context.
- `CMD ["node", "src/index.js"]`: Running `node` directly as PID 1 ensures that Linux signals (`SIGTERM`, `SIGINT`) sent by the Kubernetes kubelet are received directly by the application rather than swallowed by `npm`.

---

### 3.2 Frontend Multi-Stage Dockerfile (`applications/frontend/Dockerfile`)
```dockerfile
# Stage 1: Build production assets
FROM node:22-alpine AS builder
WORKDIR /app
COPY package*.json ./
RUN npm ci
COPY . .
ARG VITE_BACKEND_URL=""
ENV VITE_BACKEND_URL=${VITE_BACKEND_URL}
RUN npm run build

# Stage 2: Production web server (Unprivileged NGINX)
FROM nginx:alpine-slim AS production
COPY --from=builder /app/dist /usr/share/nginx/html
COPY nginx.conf /etc/nginx/nginx.conf
RUN chown -R nginx:nginx /usr/share/nginx/html && \
    chmod -R 755 /usr/share/nginx/html
USER nginx
EXPOSE 8080
CMD ["nginx", "-g", "daemon off;"]

# Stage 3: Local development mode (used by docker compose)
FROM node:22-alpine AS dev
WORKDIR /app
COPY package*.json ./
RUN npm ci
COPY . .
ENV VITE_BACKEND_URL=${VITE_BACKEND_URL}
EXPOSE 5173
CMD ["npm", "run", "dev"]
```

#### Unprivileged NGINX Configuration (`applications/frontend/nginx.conf`):
```nginx
pid /tmp/nginx.pid;

events { worker_connections 1024; }

http {
    include /etc/nginx/mime.types;
    default_type application/octet-stream;

    # Temp directories in /tmp for non-root write access
    client_body_temp_path /tmp/client_temp;
    proxy_temp_path       /tmp/proxy_temp_path;
    fastcgi_temp_path     /tmp/fastcgi_temp;

    server {
        listen 8080;  # Unprivileged port > 1024
        server_name _;
        root /usr/share/nginx/html;
        index index.html;

        # SPA Routing: fallback all routes to index.html
        location / {
            try_files $uri $uri/ /index.html;
        }

        # Kubernetes health probe endpoint
        location /healthz {
            access_log off;
            default_type text/plain;
            return 200 "healthy\n";
        }

        # Cache immutable static assets for 1 year
        location ~* \.(js|css|png|jpg|jpeg|gif|ico|svg|woff|woff2)$ {
            expires 1y;
            add_header Cache-Control "public, no-transform";
        }
    }
}
```

---

## 4. Kubernetes Infrastructure & Cluster Governance

### 4.1 Namespaces & Pod Security Standards (`kubernetes/namespaces/environments.yaml`)
Kubernetes Pod Security Standards (PSS) define security boundaries across three profiles: `Privileged`, `Baseline`, and `Restricted`.

```yaml
apiVersion: v1
kind: Namespace
metadata:
  name: production
  labels:
    environment: production
    pod-security.kubernetes.io/enforce: restricted
    pod-security.kubernetes.io/audit: restricted
    pod-security.kubernetes.io/warn: restricted
```

#### Enforced Constraints in `production`:
1. `seccompProfile.type`: Must be set to `RuntimeDefault` or `Localhost`.
2. `securityContext.runAsNonRoot`: Must be `true`.
3. `securityContext.allowPrivilegeEscalation`: Must be `false`.
4. `securityContext.capabilities.drop`: Must include `ALL`.

Any pod attempting to run as root or without seccomp confinement is rejected by the Kubernetes admission controller at admission time!

---

### 4.2 Zero-Trust Network Policies (`kubernetes/network-policies/network-policies.yaml`)
By default, Kubernetes networking is flat—any pod can communicate with any other pod across any namespace. We enforce **Zero-Trust network segmentation**:

1. **Default Deny All**:
   ```yaml
   apiVersion: networking.k8s.io/v1
   kind: NetworkPolicy
   metadata:
     name: default-deny-all
     namespace: production
   spec:
     podSelector: {}
     policyTypes: [Ingress, Egress]
   ```
2. **Whitelist CoreDNS Egress**: Allows pods to query kube-dns for service discovery on UDP/TCP port 53.
3. **Whitelist Frontend Ingress**: Allows Ingress controller on port 8080; allows Egress only to `app.kubernetes.io/name: api` on port 3000.
4. **Whitelist API Ingress & Egress**: Accepts ingress only from `frontend`; permits egress only to external PostgreSQL database on port 5432.

---

## 5. Application Packaging with Helm

We packaged both workloads into production-grade Helm charts under `helm/api` and `helm/frontend`.

### Chart File Hierarchy:
```text
helm/api/
├── Chart.yaml              # Chart metadata & versioning
├── values.yaml             # Base default configuration
├── values-dev.yaml         # Development environment overrides (1 replica, low resources)
├── values-production.yaml  # Production environment overrides (3 replicas, strict PDB)
└── templates/
    ├── _helpers.tpl        # Standardized label & naming templates
    ├── deployment.yaml     # Pod template, probes, securityContext, anti-affinity
    ├── service.yaml        # ClusterIP service
    ├── configmap.yaml      # Non-sensitive environment variables
    ├── secret.yaml         # Sensitive credentials (DATABASE_URL)
    ├── serviceaccount.yaml # Least-privilege IAM binding
    ├── hpa.yaml            # HorizontalPodAutoscaler
    └── pdb.yaml            # PodDisruptionBudget
```

### Key Reliability Features in Helm Templates:
1. **Pod Anti-Affinity** (`deployment.yaml`):
   ```yaml
   affinity:
     podAntiAffinity:
       preferredDuringSchedulingIgnoredDuringExecution:
         - weight: 100
           podAffinityTerm:
             labelSelector:
               matchExpressions:
                 - key: app.kubernetes.io/name
                   operator: In
                   values: [api]
             topologyKey: kubernetes.io/hostname
   ```
   Prevents all replicas of the API from being scheduled on the same worker node. If a node fails, surviving nodes maintain application availability.

2. **PodDisruptionBudget (`pdb.yaml`)**:
   ```yaml
   spec:
     minAvailable: 2
   ```
   Guarantees that cluster maintenance, node drains, or voluntary disruptions cannot reduce the number of running replicas below 2.

3. **Checksum-Triggered Rolling Restarts**:
   ```yaml
   annotations:
     checksum/config: {{ include (print $.Template.BasePath "/configmap.yaml") . | sha256sum }}
     checksum/secret: {{ include (print $.Template.BasePath "/secret.yaml") . | sha256sum }}
   ```
   When ConfigMaps or Secrets change, Helm automatically calculates a new SHA256 checksum in the Pod template annotations, triggering a zero-downtime rolling update.

---

## 6. GitOps Continuous Delivery with ArgoCD

ArgoCD continuously monitors the Git repository and reconciles any drift between desired state in Git and the actual state of the cluster.

### 6.1 ArgoCD Architecture & Components
- **`argocd-server`**: API server and Web UI.
- **`argocd-repo-server`**: Clones Git repos, generates manifests, and renders Helm charts.
- **`argocd-application-controller`**: Compares live Kubernetes state with Git and triggers self-healing.

### 6.2 Production GitOps Manifest (`argocd/applications/production.yaml`)
```yaml
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: api-production
  namespace: argocd
  finalizers:
    - resources-finalizer.argocd.argoproj.io
spec:
  project: platform
  source:
    repoURL: https://github.com/dipak-01/Multi-Cluster-Kubernetes-Platform.git
    targetRevision: main
    path: helm/api
    helm:
      valueFiles:
        - values.yaml
        - values-production.yaml
      parameters:
        - name: image.repository
          value: platform-api
        - name: image.tag
          value: "1.0.0"
        - name: image.pullPolicy
          value: Never
  destination:
    server: https://kubernetes.default.svc
    namespace: production
  syncPolicy:
    automated:
      prune: true
      selfHeal: true
    syncOptions:
      - CreateNamespace=true
```

#### Why `selfHeal: true` and `prune: true`:
- **`selfHeal: true`**: If an engineer manually alters a pod, deletes a deployment, or patches a service with `kubectl`, ArgoCD immediately detects the drift and overwrites the cluster back to the Git commit state.
- **`prune: true`**: When an object or manifest is deleted in Git, ArgoCD automatically deletes the corresponding resource from the cluster, preventing orphaned objects.

---

## 7. Observability, Alerting & SRE SLOs

We deployed the complete `kube-prometheus-stack` to collect telemetry across the infrastructure, Kubernetes control plane, and application workloads.

```
[ Application Pods ]   [ Nodes / OS ]   [ Kubernetes API ]
         │                    │                 │
         │ (ServiceMonitor)   │ (node-exporter) │ (kube-state-metrics)
         ▼                    ▼                 ▼
   ┌────────────────────────────────────────────────────────┐
   │                   Prometheus Server                    │
   │  - Scrape interval: 15s                                │
   │  - Retention: 24h                                      │
   │  - TSDB Storage: 5Gi                                   │
   └───────────┬────────────────────────────────┬───────────┘
               │ Evaluates Alerts               │ Queries
               ▼                                ▼
   ┌────────────────────────┐      ┌────────────────────────┐
   │      Alertmanager      │      │        Grafana         │
   │ - Golden Signals Rules │      │ - SLO Gauges           │
   │ - Runbook URL links    │      │ - Error Budget Graphs  │
   └────────────────────────┘      └────────────────────────┘
```

### 7.1 Actionable Alert Rules (`observability/alertmanager/application-alerts.yaml`)
Instead of alerting on arbitrary spikes, we alert on the **Four Golden Signals**:

1. **High API Error Rate**:
   ```yaml
   alert: HighAPIErrorRate
   expr: |
     (sum(rate(http_requests_total{status=~"5.."}[5m])) or vector(0))
     /
     (sum(rate(http_requests_total[5m])) > 0 or vector(1)) > 0.05
   for: 2m
   labels: { severity: critical }
   ```
2. **High API Latency**: Fires when p95 latency exceeds 500ms for 3 minutes.
3. **Pod High Memory Usage**: Fires when working set memory reaches 85% of limit, preempting an `OOMKill`.
4. **Pod CrashLooping**: Fires when restarts exceed 2 within 5 minutes.

### 7.2 Custom SRE SLO Dashboard (`observability/grafana/dashboards/sre-slo-dashboard.yaml`)
Automatically loaded into Grafana via the k8s sidecar (`grafana_dashboard: "1"`):
- **Availability SLI**: Target 99.90% monthly availability.
- **Error Budget Remaining**: Percentage gauge tracking error burn.
- **Pod CPU & Memory Saturation**: Dynamic timeseries graphs for every pod in the `production` namespace.

---

## 8. Chaos Engineering & Incident Simulation Suite

Testing failure recovery in production-like conditions is the hallmark of true Site Reliability Engineering.

### Executable Chaos Scripts (`chaos/`):
| Script | Failure Injected | Expected & Observed Result | SRE Runbook |
| :--- | :--- | :--- | :--- |
| `chaos/pod-failure/test-pod-resilience.sh` | Forced termination of active API pod (`--grace-period=0`) under HTTP load. | **100% request success rate (215/215 HTTP 200)**. Replacement pod ready in 27s. | [pod-crashloop.md](../runbooks/pod-crashloop.md) |
| `chaos/resource-exhaustion/simulate-memory-leak.sh` | Container allocates memory in an infinite loop past 64Mi limit. | Linux cgroup terminates container with **`OOMKilled` (Exit code 137)**. Neighboring pods unaffected. | [high-memory.md](../runbooks/high-memory.md) |
| `chaos/resource-exhaustion/cpu-stress-test.sh` | Concurrency burst targeting `/api` to saturate CPU cores. | CPU rises, triggering HPA scale-up event. | [high-cpu.md](../runbooks/high-cpu.md) |
| `chaos/bad-deployment/test-bad-release.sh` | Container command patched with invalid binary. | Pod enters `CrashLoopBackOff`; ArgoCD detects drift and triggers self-healing. | [pod-crashloop.md](../runbooks/pod-crashloop.md) |

Detailed postmortems are cataloged in [`chaos/CHAOS_EXPERIMENTS.md`](../chaos/CHAOS_EXPERIMENTS.md).

---

## 9. Operational Automation: The `platformctl` CLI Tool

To automate cluster inspection, health audits, rollbacks, and incident diagnostics, we built `platformctl` in **Go** (`automation/go/main.go`).

### Why Go?
- Compiles to a single static binary without Python virtual environments or runtime dependencies.
- Native integration with Kubernetes and Prometheus architectures.
- Installed directly into `/home/dipak/.local/bin/platformctl` (in `$PATH`).

### Command Suite:
```bash
# Overall cluster health, nodes, saturation, and SLO check
platformctl cluster health

# List nodes with roles and versions
platformctl cluster nodes

# Inspect application status, replicas, and ingress endpoint
platformctl application status api
platformctl application status frontend

# Stream application logs
platformctl application logs api 50

# Automated SRE incident diagnosis (OOM, CrashLoops, Warning Events)
platformctl incident diagnose api

# Check live SLO targets, latency, and Error Budget
platformctl slo status

# Trigger zero-downtime rolling restart
platformctl application restart api

# View deployment revisions and rollback
platformctl deployment history api
platformctl deployment rollback api
```

---

## 10. End-to-End Operational Cheatsheet

### 10.1 Host Resolution (`/etc/hosts`)
Map the Minikube IP (`192.168.49.2`) to the platform Ingress hosts:
```bash
echo "192.168.49.2 platform.local argocd.local grafana.local prometheus.local alertmanager.local" | sudo tee -a /etc/hosts
```

### 10.2 Service URLs & Credentials
| Service | URL | Credentials |
| :--- | :--- | :--- |
| **Application (UI & API)** | `http://platform.local` | None |
| **ArgoCD GitOps Console** | `http://argocd.local` | **User**: `admin`<br>**Password**: `oKaYh2mjLyEOwqfn` |
| **Grafana SRE Dashboards** | `http://grafana.local` | **User**: `admin`<br>**Password**: `admin` |
| **Direct SRE Dashboard** | `http://grafana.local/d/sre-golden-signals/40e3d64` | **User**: `admin`<br>**Password**: `admin` |
| **Prometheus Query Console** | `http://prometheus.local` | None |
| **Alertmanager Console** | `http://alertmanager.local` | None |

### 10.3 Day-2 Operations Cheatsheet
```bash
# 1. Check overall platform status
platformctl cluster health

# 2. Check ArgoCD sync status
kubectl get applications.argoproj.io -n argocd

# 3. View live production pods
kubectl get pods -n production -o wide

# 4. Run pod failure chaos experiment
./chaos/pod-failure/test-pod-resilience.sh

# 5. Run memory leak OOMKill experiment
./chaos/resource-exhaustion/simulate-memory-leak.sh

# 6. Run automated incident diagnostics
platformctl incident diagnose api
```
