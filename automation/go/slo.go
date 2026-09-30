package main

import (
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"time"
)

// SLO definitions. Keep these in sync with docs/SLO.md and
// observability/prometheus/slo-rules.yaml.
const (
	sloAvailabilityTarget = 0.995 // 99.5% of requests must not be 5xx
	sloLatencyTarget      = 0.95  // 95% of requests must be faster than the threshold
	sloLatencyBucket      = "0.5" // seconds; must be a real ingress-nginx histogram bucket
	sloSelector           = `service="api-production",namespace="production"`
)

// promClient is a minimal Prometheus HTTP API client.
type promClient struct {
	baseURL string
	host    string // optional Host header, used when going through an ingress
	http    *http.Client
}

// newPromClient reads PROMETHEUS_URL (e.g. http://localhost:9090 after a
// port-forward). If unset, it falls back to the Minikube ingress with the
// prometheus host header.
func newPromClient() *promClient {
	base := os.Getenv("PROMETHEUS_URL")
	host := ""
	if base == "" {
		base = "http://" + MinikubeIP
		host = PrometheusHost
	}
	return &promClient{
		baseURL: base,
		host:    host,
		http:    &http.Client{Timeout: 10 * time.Second},
	}
}

type promResponse struct {
	Status string `json:"status"`
	Error  string `json:"error"`
	Data   struct {
		Result []struct {
			Value [2]interface{} `json:"value"`
		} `json:"result"`
	} `json:"data"`
}

// scalar runs an instant query and returns the first sample as a float.
// ok is false when Prometheus answered but there is no usable data
// (empty result or NaN, for example when there has been no traffic).
func (c *promClient) scalar(expr string) (val float64, ok bool, err error) {
	q := url.Values{}
	q.Set("query", expr)
	req, err := http.NewRequest(http.MethodGet, c.baseURL+"/api/v1/query?"+q.Encode(), nil)
	if err != nil {
		return 0, false, err
	}
	if c.host != "" {
		req.Host = c.host
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return 0, false, fmt.Errorf("prometheus unreachable at %s: %w", c.baseURL, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, false, err
	}
	if resp.StatusCode != http.StatusOK {
		return 0, false, fmt.Errorf("prometheus returned HTTP %d: %s", resp.StatusCode, string(body))
	}

	var pr promResponse
	if err := json.Unmarshal(body, &pr); err != nil {
		return 0, false, fmt.Errorf("invalid prometheus response: %w", err)
	}
	if pr.Status != "success" {
		return 0, false, fmt.Errorf("prometheus query failed: %s", pr.Error)
	}
	if len(pr.Data.Result) == 0 {
		return 0, false, nil
	}

	raw, isString := pr.Data.Result[0].Value[1].(string)
	if !isString {
		return 0, false, fmt.Errorf("unexpected sample type %T", pr.Data.Result[0].Value[1])
	}
	f, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return 0, false, err
	}
	if math.IsNaN(f) || math.IsInf(f, 0) {
		return 0, false, nil
	}
	return f, true, nil
}

func errorRatioExpr(window string) string {
	return fmt.Sprintf(
		`sum(rate(nginx_ingress_controller_requests{%s,status=~"5.."}[%s])) / sum(rate(nginx_ingress_controller_requests{%s}[%s]))`,
		sloSelector, window, sloSelector, window)
}

func slowRatioExpr(window string) string {
	return fmt.Sprintf(
		`1 - (sum(rate(nginx_ingress_controller_request_duration_seconds_bucket{%s,le="%s"}[%s])) / sum(rate(nginx_ingress_controller_request_duration_seconds_count{%s}[%s])))`,
		sloSelector, sloLatencyBucket, window, sloSelector, window)
}

// budgetRemaining converts an observed bad-event ratio into the fraction of
// the error budget still available (1 = untouched, 0 = spent, <0 = overspent).
func budgetRemaining(badRatio, target float64) float64 {
	return 1 - badRatio/(1-target)
}

// burnRate is how many times faster than "sustainable" the budget is being consumed.
func burnRate(badRatio, target float64) float64 {
	return badRatio / (1 - target)
}

func budgetColor(remaining float64) string {
	switch {
	case remaining > 0.5:
		return ColorGreen
	case remaining > 0.2:
		return ColorYellow
	default:
		return ColorRed
	}
}

func burnColor(rate float64) string {
	switch {
	case rate >= 6:
		return ColorRed
	case rate > 1:
		return ColorYellow
	default:
		return ColorGreen
	}
}

// handleSLOStatus prints real SLI values and error budget from Prometheus.
// It never prints placeholder numbers: if data is missing, it says so.
func handleSLOStatus() {
	c := newPromClient()

	fmt.Printf("\n%s=======================================================%s\n", ColorBold, ColorReset)
	fmt.Printf("%s  SLO Status: api-production (rolling 30d)%s\n", ColorBold, ColorReset)
	fmt.Printf("%s=======================================================%s\n\n", ColorBold, ColorReset)

	errors30d, ok, err := c.scalar(errorRatioExpr("30d"))
	if err != nil {
		fmt.Printf("  %sCannot read SLO data: %v%s\n", ColorRed, err, ColorReset)
		fmt.Printf("  Set PROMETHEUS_URL (e.g. http://localhost:9090 after a port-forward).\n\n")
		os.Exit(1)
	}

	// Availability
	fmt.Printf("  %sAvailability%s  (target %.2f%%)\n", ColorBold, ColorReset, sloAvailabilityTarget*100)
	if !ok {
		fmt.Printf("    %sNO DATA%s (no requests recorded, or the metric/labels do not match)\n", ColorYellow, ColorReset)
	} else {
		remaining := budgetRemaining(errors30d, sloAvailabilityTarget)
		col := budgetColor(remaining)
		fmt.Printf("    Current:        %s%.3f%%%s\n", col, (1-errors30d)*100, ColorReset)
		if remaining < 0 {
			fmt.Printf("    Error budget:   %sEXHAUSTED (%.0f%% overspent)%s\n", ColorRed, -remaining*100, ColorReset)
		} else {
			fmt.Printf("    Error budget:   %s%.1f%% remaining%s\n", col, remaining*100, ColorReset)
		}
	}

	// Burn rates
	fmt.Printf("\n  %sAvailability burn rate%s  (1.0 = budget lasts exactly 30d)\n", ColorBold, ColorReset)
	for _, w := range []string{"1h", "6h", "1d"} {
		v, have, qerr := c.scalar(errorRatioExpr(w))
		switch {
		case qerr != nil:
			fmt.Printf("    %-4s %sERROR: %v%s\n", w, ColorRed, qerr, ColorReset)
		case !have:
			fmt.Printf("    %-4s %sno data%s\n", w, ColorYellow, ColorReset)
		default:
			r := burnRate(v, sloAvailabilityTarget)
			fmt.Printf("    %-4s %s%.2fx%s\n", w, burnColor(r), r, ColorReset)
		}
	}

	// Latency
	fmt.Printf("\n  %sLatency%s  (target %.0f%% of requests < %sms)\n", ColorBold, ColorReset, sloLatencyTarget*100, secondsToMillis(sloLatencyBucket))
	slow30d, have, qerr := c.scalar(slowRatioExpr("30d"))
	switch {
	case qerr != nil:
		fmt.Printf("    %sERROR: %v%s\n", ColorRed, qerr, ColorReset)
	case !have:
		fmt.Printf("    %sNO DATA%s\n", ColorYellow, ColorReset)
	default:
		remaining := budgetRemaining(slow30d, sloLatencyTarget)
		col := budgetColor(remaining)
		fmt.Printf("    Fast requests:  %s%.2f%%%s\n", col, (1-slow30d)*100, ColorReset)
		fmt.Printf("    Error budget:   %s%.1f%% remaining%s\n", col, remaining*100, ColorReset)
	}

	fmt.Printf("\n  %sNote:%s windows cover only the data Prometheus has retained.\n", ColorBold, ColorReset)
	fmt.Printf("%s=======================================================%s\n\n", ColorBold, ColorReset)
}

// printSLOSummary is a compact version for `cluster health`.
// Call it in place of the old hardcoded "SLO Status" lines.
func printSLOSummary() {
	c := newPromClient()
	fmt.Printf("\n  %sSLO Status:%s\n", ColorBold, ColorReset)

	v, ok, err := c.scalar(errorRatioExpr("30d"))
	switch {
	case err != nil:
		fmt.Printf("    %sUnavailable: %v%s\n", ColorYellow, err, ColorReset)
	case !ok:
		fmt.Printf("    %sNo data%s\n", ColorYellow, ColorReset)
	default:
		remaining := budgetRemaining(v, sloAvailabilityTarget)
		col := budgetColor(remaining)
		fmt.Printf("    Availability: %s%.3f%%%s (Target: %.2f%%)\n", col, (1-v)*100, ColorReset, sloAvailabilityTarget*100)
		fmt.Printf("    Error Budget: %s%.1f%% remaining%s\n", col, remaining*100, ColorReset)
	}
}

func secondsToMillis(s string) string {
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return s
	}
	return strconv.FormatFloat(f*1000, 'f', -1, 64)
}
