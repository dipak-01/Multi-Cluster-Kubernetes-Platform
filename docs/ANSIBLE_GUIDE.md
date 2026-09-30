# Ansible Infrastructure Automation & Node Hardening Guide

This document describes the automated node provisioning, operating system hardening, container runtime setup, and Kubernetes node preparation implemented via Ansible.

---

## Architecture Overview

```text
                        Inventory (hosts.ini)
                                 │
           ┌─────────────────────┴─────────────────────┐
           ▼                                           ▼
[k8s_control_plane]                              [k8s_workers]
  - cp-01                                          - worker-01
                                                   - worker-02
                                 │
                                 ▼
                     Master Playbook (site.yml)
                                 │
  ┌──────────────┬───────────────┼───────────────┬──────────────┐
  ▼              ▼               ▼               ▼              ▼
roles/linux    roles/hardening roles/containerd roles/k8s     roles/node-exporter
(OS Baseline)  (SSH, Limits)   (CRI & cgroups)  (kubelet/etc) (Metrics Agent)
```

---

## Directory Structure

```text
ansible/
├── ansible.cfg                    # Ansible configuration (pipelining, paths, SSH)
├── requirements.yml               # Required collections (community.general, ansible.posix)
├── inventory/
│   ├── hosts.ini                  # Production nodes & local validation target
│   └── group_vars/
│       ├── all.yml                # Cluster-wide parameters (versions, sysctl, SSH policy)
│       ├── k8s_control_plane.yml  # Control plane specific ports & variables
│       └── k8s_workers.yml        # Worker node specific ports
├── playbooks/
│   ├── site.yml                   # Master end-to-end orchestration
│   ├── bootstrap.yml              # Linux baseline, packages, swapoff, kernel modules
│   ├── hardening.yml              # SSH hardening, system resource limits, strict umask
│   ├── containerd.yml             # containerd runtime, systemd cgroup driver
│   ├── kubernetes.yml             # kubelet, kubeadm, kubectl repo & hold
│   └── monitoring.yml             # Prometheus node_exporter service (port 9100)
└── roles/
    ├── linux/                     # Baseline OS configuration & sysctl tuning
    ├── hardening/                 # Security policies & pam limits
    ├── containerd/                # Container runtime installation & config.toml
    ├── kubernetes/                # Kubernetes package management
    └── node-exporter/             # Systemd daemon for host telemetry
```

---

## Role Details

### 1. `roles/linux` (OS Baseline & Kernel Tuning)
- **Swap Disabling**: Disables active swap (`swapoff -a`) and comments out swap partitions in `/etc/fstab` to ensure zero swap across reboots (mandatory for Kubernetes memory scheduling).
- **Kernel Modules**: Ensures `overlay` and `br_netfilter` are loaded immediately and on boot (`/etc/modules-load.d/k8s.conf`).
- **Sysctl Kernel Tuning** (`/etc/sysctl.d/99-kubernetes-cri.conf`):
  - `net.bridge.bridge-nf-call-iptables = 1`
  - `net.bridge.bridge-nf-call-ip6tables = 1`
  - `net.ipv4.ip_forward = 1`
  - `vm.max_map_count = 262144`
  - `fs.file-max = 2097152`
  - `fs.inotify.max_user_watches = 524288`
  - `fs.inotify.max_user_instances = 8192`

### 2. `roles/hardening` (Security Policies)
- **SSH Hardening** (`/etc/ssh/sshd_config.d/99-hardening.conf`):
  - `PermitRootLogin no`
  - `PasswordAuthentication no`
  - `PermitEmptyPasswords no`
  - `MaxAuthTries 4`
  - `ClientAliveInterval 300`
- **File Descriptors & Process Limits** (`/etc/security/limits.d/99-k8s.conf`):
  - `nofile` raised to `1048576` for high-throughput container workloads.
  - `nproc` set to `65536`.
- **System Umask**: Enforces strict `027` umask.

### 3. `roles/containerd` (Container Runtime)
- Configures Docker official repository and installs `containerd.io`.
- Generates `/etc/containerd/config.toml` with:
  - `SystemdCgroup = true` (ensuring alignment with systemd slice management).
  - `sandbox_image = "registry.k8s.io/pause:3.9"`.
- Configures `/etc/crictl.yaml` pointing to `unix:///run/containerd/containerd.sock`.

### 4. `roles/kubernetes` (Node Tooling)
- Adds official Kubernetes community repository (`pkgs.k8s.io`).
- Installs `kubelet`, `kubeadm`, and `kubectl` pinned to target version.
- Places `dpkg` / package hold to prevent accidental upgrades outside GitOps release windows.

### 5. `roles/node-exporter` (Observability Agent)
- Creates dedicated unprivileged system user `node_exporter`.
- Installs official Prometheus `node_exporter` binary (v1.8.1).
- Deploys sandboxed systemd unit (`ProtectSystem=strict`, `ProtectHome=true`, `NoNewPrivileges=true`).
- Exposes node-level telemetry (CPU, Memory, Disk, Network) on port `9100` for Prometheus scraping.

---

## How to Run

### Install Prerequisites
```bash
# Install collections
ansible-galaxy collection install -r ansible/requirements.yml
```

### Syntax Validation
```bash
ansible-playbook -i ansible/inventory/hosts.ini ansible/playbooks/site.yml --syntax-check
```

### Dry Run (Check Mode)
```bash
ansible-playbook -i ansible/inventory/hosts.ini ansible/playbooks/site.yml --check
```

### Execute Master Deployment
```bash
ansible-playbook -i ansible/inventory/hosts.ini ansible/playbooks/site.yml
```

### Run Individual Components
```bash
# Run only security hardening
ansible-playbook -i ansible/inventory/hosts.ini ansible/playbooks/hardening.yml

# Run only monitoring setup
ansible-playbook -i ansible/inventory/hosts.ini ansible/playbooks/monitoring.yml
```
