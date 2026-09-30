# SRE Runbook: Linux Performance & Systems Troubleshooting

This runbook documents diagnosis and remediation procedures for Linux infrastructure running Kubernetes nodes and platform services.

---

## 1. High CPU Saturation

### Diagnostic Procedure
1. Check overall CPU load average:
   ```bash
   uptime
   # Compare load average (1m, 5m, 15m) against available CPU cores:
   nproc
   ```
2. Identify CPU breakdown (user, system, iowait, steal):
   ```bash
   top -b -n 1 | head -n 15
   vmstat 1 5
   # Focus on:
   # %us: User space application load
   # %sy: Kernel / syscall overhead
   # %wa: Blocked on disk/network I/O
   # %st: Hypervisor steal (noisy neighbors in cloud / AWS)
   ```
3. Locate highest CPU consuming processes:
   ```bash
   ps -eo pid,ppid,user,%cpu,%mem,cmd --sort=-%cpu | head -n 15
   ```
4. Profile CPU threads in real time:
   ```bash
   pidstat -u 1 5
   htop
   ```

### Remediation
- **Application spin / infinite loop**: Trace process syscalls:
  ```bash
  strace -p <PID> -c -T
  ```
- **High `%sy` (System Overhead)**: Check for excessive context switching or interrupts:
  ```bash
  vmstat 1
  cat /proc/interrupts
  ```
- **Kubernetes pod CPU throttling**: Check container metrics and adjust resource limits:
  ```bash
  kubectl top pods -A --sort-by=cpu
  ```

---

## 2. Memory Pressure & OOM (Out Of Memory)

### Diagnostic Procedure
1. Inspect physical and swap memory allocation:
   ```bash
   free -h
   vmstat -s
   ```
2. Detect OOM Killer invocations in kernel logs:
   ```bash
   dmesg -T | grep -i -E "oom|out of memory|killed process"
   journalctl -k --grep="Out of memory" --since "1 hour ago"
   ```
3. List top memory-consuming processes (by RSS):
   ```bash
   ps -eo pid,ppid,user,%mem,rss,cmd --sort=-rss | head -n 15
   ```
4. Verify Slab and Buffer/Cache breakdown:
   ```bash
   cat /proc/meminfo | head -n 25
   slabtop -o
   ```

### Remediation
- **Kill runaway memory leak process safely**:
  ```bash
  kill -15 <PID>  # Graceful SIGTERM
  # If unresponsive after 10s:
  kill -9 <PID>   # Forced SIGKILL
  ```
- **Clear harmless filesystem page cache (if emergency memory recovery needed)**:
  ```bash
  sync; echo 3 > /proc/sys/vm/drop_caches
  ```
- **Kubernetes OOMKilled pods**:
  ```bash
  kubectl get pods -A -o jsonpath='{range .items[*]}{.metadata.name}{"\t"}{.status.containerStatuses[*].lastState.terminated.reason}{"\n"}{end}' | grep OOMKilled
  ```

---

## 3. Disk Space & Inode Exhaustion

### Diagnostic Procedure
1. Check filesystem capacity and mount points:
   ```bash
   df -hT
   ```
2. Check inode exhaustion (filesystems can run out of inodes even with disk space left):
   ```bash
   df -ih
   ```
3. Identify largest directories and files:
   ```bash
   du -xh / | sort -rh | head -n 20
   # Find files larger than 500MB:
   find / -xdev -type f -size +500M -exec ls -lh {} +
   ```
4. Find deleted files still held open by running processes (causing phantom disk usage):
   ```bash
   lsof +L1
   ```

### Remediation
- **Truncate huge unrotated logs safely (do not `rm` an open log file)**:
  ```bash
  truncate -s 0 /var/log/<huge-service.log>
  ```
- **Clean systemd journal logs older than 3 days**:
  ```bash
  journalctl --vacuum-time=3d
  journalctl --vacuum-size=1G
  ```
- **Clean unused container layers and images**:
  ```bash
  docker system prune -af --volumes
  # Or containerd:
  crictl rmi --prune
  ```

---

## 4. Disk I/O Saturation & Latency

### Diagnostic Procedure
1. Monitor device throughput, IOPS, and queue depth:
   ```bash
   iostat -xz 1 5
   # Look at:
   # %util: Percentage of time device had I/O requests active (>= 90% is saturated)
   # await: Average time (ms) for I/O requests to be served
   # r_await / w_await: Read/write latency
   # avgqu-sz: Average request queue length
   ```
2. Identify processes generating highest I/O:
   ```bash
   iotop -o -b -n 3
   pidstat -d 1 5
   ```

### Remediation
- Lower process I/O scheduling priority (`ionice`):
  ```bash
  ionice -c 3 -p <PID>  # Idle priority
  ```
- Inspect block device queue scheduler:
  ```bash
  cat /sys/block/sda/queue/scheduler
  ```

---

## 5. Network Congestion & Socket Exhaustion

### Diagnostic Procedure
1. Check active TCP socket states and queue backlog:
   ```bash
   ss -s
   ss -tulpn
   # Check sockets in TIME_WAIT / CLOSE_WAIT:
   ss -tan | awk '{print $1}' | sort | uniq -c
   ```
2. Inspect network interface errors and drops:
   ```bash
   ip -s link
   netstat -s | grep -i -E "listen|drop|reset|retransmit"
   ```
3. Test connectivity and DNS resolution:
   ```bash
   curl -Iv --connect-timeout 5 https://<endpoint>
   dig @127.0.0.53 <hostname> +trace
   ```

### Remediation
- Enable TCP TIME_WAIT reuse for high-traffic proxy nodes in `/etc/sysctl.d/99-network.conf`:
  ```ini
  net.ipv4.tcp_tw_reuse = 1
  net.ipv4.tcp_fin_timeout = 15
  net.core.somaxconn = 65535
  net.ipv4.ip_local_port_range = 1024 65535
  ```
  Apply with:
  ```bash
  sysctl -p /etc/sysctl.d/99-network.conf
  ```

---

## 6. File Descriptor Exhaustion

### Diagnostic Procedure
1. Check system-wide file descriptor allocation:
   ```bash
   cat /proc/sys/fs/file-nr
   # [Allocated file descriptors] [Unused] [Maximum limit]
   ```
2. Check per-process open file descriptor counts:
   ```bash
   lsof -n | awk '{print $2}' | sort | uniq -c | sort -rn | head -n 10
   ```
3. Check process limits:
   ```bash
   cat /proc/<PID>/limits | grep "Max open files"
   ```

### Remediation
- Adjust `/etc/security/limits.d/99-nofile.conf`:
  ```ini
  * soft nofile 65536
  * hard nofile 1048576
  ```
- For systemd services, configure `LimitNOFILE=65536` in the service unit.
