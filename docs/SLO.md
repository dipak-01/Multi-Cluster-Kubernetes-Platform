# Service Level Objectives (SLOs)

This document defines the reliability targets for the `api-production` service, how they are measured, and what the team does when the error budget runs low.

- **Service:** `api-production` (namespace `production`)
- **Measured at:** the ingress-nginx controller (what users actually experience), not inside the app
- **Window:** rolling 30 days
- **Rules:** [`observability/prometheus/slo-rules.yaml`](../observability/prometheus/slo-rules.yaml)
- **CLI:** `platformctl slo status`

---

## 1. Service Level Indicators (SLIs)

| SLI | Definition | Metric source |
|---|---|---|
| **Availability** | Fraction of requests that do **not** return HTTP 5xx | `nginx_ingress_controller_requests` |
| **Latency** | Fraction of requests served in under **500 ms** | `nginx_ingress_controller_request_duration_seconds_bucket{le="0.5"}` |

Notes on the choices:

- 4xx responses are **not** counted as failures. They are client errors (bad input, unknown pokemon ID), not service faults.
- 500 ms was chosen because `0.5` is a native histogram bucket in ingress-nginx. `0.3` is not, and PromQL cannot interpolate a bucket boundary that does not exist.
- Measuring at the ingress means the SLI also catches failures where the pod never answers (connection errors, 502/503/504).

## 2. Objectives

| SLO | Target | Error budget (30d) |
|---|---|---|
| Availability | **99.5%** of requests succeed | 0.5% of requests, about **3 h 36 min** of full downtime equivalent |
| Latency | **95%** of requests under 500 ms | 5% of requests may be slower |

Why 99.5% and not 99.9%: this platform runs on a single local cluster with a small number of replicas. A 99.9% target (43 min/month) would be a claim the infrastructure cannot honestly support yet. Tighten the target after multi-node or multi-cluster failover is proven by chaos experiments.

## 3. Error budget math

```text
error_budget        = 1 - SLO target                  (availability: 0.005)
burn_rate           = observed_error_ratio / error_budget
budget_remaining    = 1 - (error_ratio_over_30d / error_budget)
```

A burn rate of **1** uses the budget exactly over 30 days. A burn rate of **14.4** exhausts it in about 2 days.

## 4. Alerting: multi-window, multi-burn-rate

Each alert needs a **long** window (to confirm it is real) and a **short** window (to confirm it is still happening), which avoids paging on a spike that already ended. This follows the approach in the Google SRE Workbook.

| Severity | Burn rate | Long / short window | Budget consumed when it fires | Action |
|---|---|---|---|---|
| **page** | 14.4x | 1h / 5m | 2% | Wake someone up |
| **page** | 6x | 6h / 30m | 5% | Wake someone up |
| **ticket** | 3x | 1d / 2h | 10% | Fix during working hours |
| **ticket** | 1x | 3d / 6h | 10% | Investigate trend |

Additional alerts:

- `APISLIMetricMissing`: fires if the request metric disappears, so a broken scrape does not silently look like "no errors".
- Two latency burn-rate alerts (fast and slow) using the same pattern.

## 5. Error budget policy

| Budget remaining | What changes |
|---|---|
| **> 50%** | Normal operation. Ship features, run chaos experiments. |
| **20 to 50%** | Review recent deploys. Risky changes need an extra reviewer. |
| **< 20%** | Feature deploys paused. Only reliability work and fixes go out. |
| **Exhausted (<= 0%)** | Change freeze except fixes for the SLO. Write a postmortem within 5 working days. |

## 6. How this connects to the chaos experiments

Each experiment in [`chaos/CHAOS_EXPERIMENTS.md`](../chaos/CHAOS_EXPERIMENTS.md) should record:

1. Burn rate observed during the experiment.
2. Whether the matching alert fired, and how long after injection.
3. Error budget consumed.

If an experiment hurts users and no alert fires, that is a monitoring gap and a finding worth writing up.

## 7. Known limitations (be honest about these)

- **Low traffic makes ratios noisy.** At a few requests per minute, one 5xx can look like a 50% error rate. Mitigation: run a synthetic load generator or a blackbox-exporter probe so there is always baseline traffic.
- **Short history.** If Prometheus retention is less than 30 days, the 30d figures cover only the data that exists.
- **Single-cluster measurement.** Numbers reflect the local cluster only, not a multi-region view.
- **Label names may differ.** The rules assume ingress-nginx labels `service` and `namespace`. Verify with `kubectl -n ingress-nginx port-forward` and a query of `nginx_ingress_controller_requests` before relying on them.
