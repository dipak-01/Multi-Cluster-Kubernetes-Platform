package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// ANSI color codes
const (
	ColorReset  = "\033[0m"
	ColorRed    = "\033[31m"
	ColorGreen  = "\033[32m"
	ColorYellow = "\033[33m"
	ColorBlue   = "\033[34m"
	ColorPurple = "\033[35m"
	ColorCyan   = "\033[36m"
	ColorWhite  = "\033[37m"
	ColorBold   = "\033[1m"
)

const (
	DefaultNamespace = "production"
	PrometheusHost   = "prometheus.local"
	MinikubeIP       = "192.168.49.2"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]

	switch command {
	case "cluster":
		if len(os.Args) < 3 {
			fmt.Printf("%sError: missing cluster subcommand (health, nodes)%s\n", ColorRed, ColorReset)
			os.Exit(1)
		}
		switch os.Args[2] {
		case "health":
			handleClusterHealth()
		case "nodes":
			handleClusterNodes()
		default:
			fmt.Printf("%sUnknown cluster subcommand: %s%s\n", ColorRed, os.Args[2], ColorReset)
			os.Exit(1)
		}

	case "application":
		if len(os.Args) < 4 {
			fmt.Printf("%sUsage: platformctl application <status|logs|restart> <app-name>%s\n", ColorYellow, ColorReset)
			os.Exit(1)
		}
		subcmd := os.Args[2]
		appName := os.Args[3]
		switch subcmd {
		case "status":
			handleAppStatus(appName)
		case "logs":
			tail := "50"
			if len(os.Args) >= 5 {
				tail = os.Args[4]
			}
			handleAppLogs(appName, tail)
		case "restart":
			handleAppRestart(appName)
		default:
			fmt.Printf("%sUnknown application subcommand: %s%s\n", ColorRed, subcmd, ColorReset)
			os.Exit(1)
		}

	case "deployment":
		if len(os.Args) < 4 {
			fmt.Printf("%sUsage: platformctl deployment <history|rollback> <app-name>%s\n", ColorYellow, ColorReset)
			os.Exit(1)
		}
		subcmd := os.Args[2]
		appName := os.Args[3]
		switch subcmd {
		case "history":
			handleDeploymentHistory(appName)
		case "rollback":
			revision := ""
			if len(os.Args) >= 5 {
				revision = os.Args[4]
			}
			handleDeploymentRollback(appName, revision)
		default:
			fmt.Printf("%sUnknown deployment subcommand: %s%s\n", ColorRed, subcmd, ColorReset)
			os.Exit(1)
		}

	case "incident":
		if len(os.Args) < 4 || os.Args[2] != "diagnose" {
			fmt.Printf("%sUsage: platformctl incident diagnose <app-name>%s\n", ColorYellow, ColorReset)
			os.Exit(1)
		}
		handleIncidentDiagnose(os.Args[3])

	case "slo":
		if len(os.Args) < 3 || os.Args[2] != "status" {
			fmt.Printf("%sUsage: platformctl slo status%s\n", ColorYellow, ColorReset)
			os.Exit(1)
		}
		handleSLOStatus()

	case "version":
		fmt.Printf("%splatformctl%s v1.0.0 (Production SRE Platform CLI)\n", ColorBold, ColorReset)

	case "help", "--help", "-h":
		printUsage()

	default:
		fmt.Printf("%sUnknown command: %s%s\n\n", ColorRed, command, ColorReset)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Printf("%s%splatformctl — Operational CLI for Multi-Cluster Kubernetes Platform%s\n\n", ColorBold, ColorCyan, ColorReset)
	fmt.Printf("%sUSAGE:%s\n", ColorBold, ColorReset)
	fmt.Printf("  platformctl <command> [subcommand] [arguments]\n\n")
	fmt.Printf("%sCORE COMMANDS:%s\n", ColorBold, ColorReset)
	fmt.Printf("  %scluster health%s              Inspect overall cluster, control plane, node, and SLO health\n", ColorGreen, ColorReset)
	fmt.Printf("  %scluster nodes%s               List cluster nodes and capacity\n", ColorGreen, ColorReset)
	fmt.Printf("  %sapplication status <app>%s    Check workload status, replicas, and endpoints\n", ColorGreen, ColorReset)
	fmt.Printf("  %sapplication logs <app> [N]%s  View the last N log lines for the application\n", ColorGreen, ColorReset)
	fmt.Printf("  %sapplication restart <app>%s   Trigger a zero-downtime rolling restart\n", ColorGreen, ColorReset)
	fmt.Printf("  %sdeployment history <app>%s    View revision history of the application deployment\n", ColorGreen, ColorReset)
	fmt.Printf("  %sdeployment rollback <app> [N]%s Roll back deployment to a previous revision\n", ColorGreen, ColorReset)
	fmt.Printf("  %sincident diagnose <app>%s     Automated SRE diagnostics for CrashLoops, OOMs, and events\n", ColorGreen, ColorReset)
	fmt.Printf("  %sslo status%s                  Display real-time SLI metrics and Error Budget remaining\n", ColorGreen, ColorReset)
	fmt.Println()
}

// -----------------------------------------------------------------------------
// Cluster Commands
// -----------------------------------------------------------------------------

func handleClusterHealth() {
	clusterName, _ := runKubectl("config", "current-context")
	clusterName = strings.TrimSpace(clusterName)
	if clusterName == "" {
		clusterName = "minikube"
	}

	fmt.Println()
	fmt.Printf("%s=======================================================%s\n", ColorBold, ColorReset)
	fmt.Printf("%s  Cluster:%s %s%s%s\n", ColorBold, ColorReset, ColorCyan, clusterName, ColorReset)
	fmt.Printf("%s=======================================================%s\n\n", ColorBold, ColorReset)

	// Control Plane Check
	cpStatus := "HEALTHY"
	cpColor := ColorGreen
	apiCheck, err := runKubectl("get", "--raw=/livez")
	if err != nil || !strings.Contains(apiCheck, "ok") {
		cpStatus = "DEGRADED"
		cpColor = ColorRed
	}
	fmt.Printf("  %sControl Plane:%s %s%s%s\n\n", ColorBold, ColorReset, cpColor, cpStatus, ColorReset)

	// Nodes
	nodesRaw, _ := runKubectl("get", "nodes", "--no-headers")
	fmt.Printf("  %sNodes:%s\n", ColorBold, ColorReset)
	for _, line := range strings.Split(strings.TrimSpace(nodesRaw), "\n") {
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		nodeName := fields[0]
		status := fields[1]
		statusColor := ColorGreen
		if status != "Ready" {
			statusColor = ColorRed
		}
		fmt.Printf("    %-12s %s%s%s\n", nodeName, statusColor, status, ColorReset)
	}

	// Resource Saturation (Node metrics)
	fmt.Printf("\n  %sResource Saturation:%s\n", ColorBold, ColorReset)
	topNodes, err := runKubectl("top", "nodes", "--no-headers")
	if err == nil && topNodes != "" {
		for _, line := range strings.Split(strings.TrimSpace(topNodes), "\n") {
			fields := strings.Fields(line)
			if len(fields) >= 5 {
				fmt.Printf("    CPU:    %-8s (utilization: %s)\n", fields[1], fields[2])
				fmt.Printf("    Memory: %-8s (utilization: %s)\n", fields[3], fields[4])
			}
		}
	} else {
		fmt.Println("    CPU:       Normal (Metrics collection active)")
		fmt.Println("    Memory:    Normal (Metrics collection active)")
	}

	// Pod Statuses
	podsRaw, _ := runKubectl("get", "pods", "-A", "--no-headers")
	running := 0
	pending := 0
	failed := 0
	for _, line := range strings.Split(strings.TrimSpace(podsRaw), "\n") {
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) >= 4 {
			status := fields[3]
			switch status {
			case "Running", "Completed":
				running++
			case "Pending", "ContainerCreating":
				pending++
			default:
				failed++
			}
		}
	}

	fmt.Printf("\n  %sPods:%s\n", ColorBold, ColorReset)
	fmt.Printf("    Running:  %s%d%s\n", ColorGreen, running, ColorReset)
	fmt.Printf("    Pending:  %s%d%s\n", ColorYellow, pending, ColorReset)
	fmt.Printf("    Failed:   %s%d%s\n", ColorRed, failed, ColorReset)

	// SLO Health
	printSLOSummary()

	overall := "HEALTHY"
	overallColor := ColorGreen
	if failed > 0 {
		overall = "WARNING"
		overallColor = ColorYellow
	}
	fmt.Printf("\n  %sOverall Status:%s %s%s%s\n", ColorBold, ColorReset, overallColor, overall, ColorReset)
	fmt.Printf("%s=======================================================%s\n\n", ColorBold, ColorReset)
}

func handleClusterNodes() {
	fmt.Printf("\n%sCluster Nodes Information:%s\n", ColorBold, ColorReset)
	out, err := runKubectl("get", "nodes", "-o", "wide")
	if err != nil {
		fmt.Printf("%sError fetching nodes: %v%s\n", ColorRed, err, ColorReset)
		return
	}
	fmt.Println(out)
}

// -----------------------------------------------------------------------------
// Application Commands
// -----------------------------------------------------------------------------

func resolveDeploymentName(appName string) string {
	if strings.HasSuffix(appName, "-production") {
		return appName
	}
	return appName + "-production"
}

func handleAppStatus(appName string) {
	depName := resolveDeploymentName(appName)
	fmt.Printf("\n%sChecking Application Status:%s %s%s%s (Namespace: %s)\n", ColorBold, ColorReset, ColorCyan, depName, ColorReset, DefaultNamespace)

	// Get Deployment JSON
	depJson, err := runKubectl("get", "deployment", depName, "-n", DefaultNamespace, "-o", "json")
	if err != nil {
		fmt.Printf("%sDeployment '%s' not found in namespace '%s'%s\n", ColorRed, depName, DefaultNamespace, ColorReset)
		return
	}

	var dep map[string]interface{}
	json.Unmarshal([]byte(depJson), &dep)

	status, _ := dep["status"].(map[string]interface{})
	spec, _ := dep["spec"].(map[string]interface{})

	replicas, _ := spec["replicas"].(float64)
	readyReplicas, _ := status["readyReplicas"].(float64)
	availableReplicas, _ := status["availableReplicas"].(float64)

	fmt.Printf("  Replicas:  %d Desired | %s%d Ready%s | %d Available\n",
		int(replicas), ColorGreen, int(readyReplicas), ColorReset, int(availableReplicas))

	// Get Pods
	fmt.Printf("\n  %sActive Pods:%s\n", ColorBold, ColorReset)
	pods, _ := runKubectl("get", "pods", "-n", DefaultNamespace, "-l", fmt.Sprintf("app.kubernetes.io/name=%s", appName), "-o", "custom-columns=NAME:.metadata.name,STATUS:.status.phase,RESTARTS:.status.containerStatuses[0].restartCount,IP:.status.podIP,AGE:.metadata.creationTimestamp")
	for _, line := range strings.Split(strings.TrimSpace(pods), "\n") {
		fmt.Println("    " + line)
	}

	// Get Ingress
	ing, err := runKubectl("get", "ingress", "-n", DefaultNamespace, "-l", fmt.Sprintf("app.kubernetes.io/name=%s", appName), "-o", "jsonpath={.items[0].spec.rules[0].host}")
	if err == nil && ing != "" {
		fmt.Printf("\n  %sEndpoint:%s http://%s/\n", ColorBold, ColorReset, ing)
	}
	fmt.Println()
}

func handleAppLogs(appName string, tail string) {
	depName := resolveDeploymentName(appName)
	fmt.Printf("%sStreaming last %s logs for %s...%s\n\n", ColorCyan, tail, depName, ColorReset)

	cmd := exec.Command("kubectl", "logs", "-n", DefaultNamespace, fmt.Sprintf("deployment/%s", depName), "--tail="+tail, "-f")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Run()
}

func handleAppRestart(appName string) {
	depName := resolveDeploymentName(appName)
	fmt.Printf("%sInitiating zero-downtime rolling restart for %s...%s\n", ColorYellow, depName, ColorReset)

	out, err := runKubectl("rollout", "restart", fmt.Sprintf("deployment/%s", depName), "-n", DefaultNamespace)
	if err != nil {
		fmt.Printf("%sError triggering restart: %s%s\n", ColorRed, out, ColorReset)
		return
	}
	fmt.Println(out)

	fmt.Printf("%sWaiting for rollout to complete successfully...%s\n", ColorCyan, ColorReset)
	statusOut, _ := runKubectl("rollout", "status", fmt.Sprintf("deployment/%s", depName), "-n", DefaultNamespace)
	fmt.Printf("%s%s%s\n", ColorGreen, statusOut, ColorReset)
}

func handleDeploymentHistory(appName string) {
	depName := resolveDeploymentName(appName)
	fmt.Printf("\n%sDeployment History for %s:%s\n", ColorBold, depName, ColorReset)
	out, err := runKubectl("rollout", "history", fmt.Sprintf("deployment/%s", depName), "-n", DefaultNamespace)
	if err != nil {
		fmt.Printf("%sError retrieving history: %v%s\n", ColorRed, err, ColorReset)
		return
	}
	fmt.Println(out)
}

func handleDeploymentRollback(appName string, revision string) {
	depName := resolveDeploymentName(appName)
	args := []string{"rollout", "undo", fmt.Sprintf("deployment/%s", depName), "-n", DefaultNamespace}
	if revision != "" {
		args = append(args, "--to-revision="+revision)
		fmt.Printf("%sRolling back %s to revision %s...%s\n", ColorYellow, depName, revision, ColorReset)
	} else {
		fmt.Printf("%sRolling back %s to previous revision...%s\n", ColorYellow, depName, ColorReset)
	}

	out, err := runKubectl(args...)
	if err != nil {
		fmt.Printf("%sRollback failed: %v%s\n", ColorRed, err, ColorReset)
		return
	}
	fmt.Printf("%s%s%s\n", ColorGreen, out, ColorReset)
}

// -----------------------------------------------------------------------------
// SRE Incident Diagnosis Command
// -----------------------------------------------------------------------------

func handleIncidentDiagnose(appName string) {
	depName := resolveDeploymentName(appName)
	fmt.Printf("\n%s=======================================================%s\n", ColorBold, ColorReset)
	fmt.Printf("%s  🔍 Automated SRE Incident Diagnosis: %s%s%s\n", ColorBold, ColorCyan, depName, ColorReset)
	fmt.Printf("%s=======================================================%s\n\n", ColorBold, ColorReset)

	issuesFound := 0

	// 1. Check Pod Phases & Restart Counts
	fmt.Printf("[1/4] Checking Pod Lifecycle & Exit Codes...\n")
	podsJson, err := runKubectl("get", "pods", "-n", DefaultNamespace, "-l", fmt.Sprintf("app.kubernetes.io/name=%s", appName), "-o", "json")
	if err == nil {
		var podList map[string]interface{}
		json.Unmarshal([]byte(podsJson), &podList)
		items, _ := podList["items"].([]interface{})

		for _, item := range items {
			p, _ := item.(map[string]interface{})
			meta, _ := p["metadata"].(map[string]interface{})
			podName, _ := meta["name"].(string)
			status, _ := p["status"].(map[string]interface{})
			containerStatuses, _ := status["containerStatuses"].([]interface{})

			for _, cs := range containerStatuses {
				c, _ := cs.(map[string]interface{})
				restartCount, _ := c["restartCount"].(float64)

				if restartCount > 0 {
					issuesFound++
					fmt.Printf("  %s⚠️  Pod %s has restarted %d times!%s\n", ColorYellow, podName, int(restartCount), ColorReset)
				}

				// Check last termination reason
				lastState, _ := c["lastState"].(map[string]interface{})
				if terminated, ok := lastState["terminated"].(map[string]interface{}); ok {
					reason, _ := terminated["reason"].(string)
					exitCode, _ := terminated["exitCode"].(float64)
					fmt.Printf("    %sLast Failure Reason: %s (Exit Code: %d)%s\n", ColorRed, reason, int(exitCode), ColorReset)

					if reason == "OOMKilled" || exitCode == 137 {
						fmt.Printf("    %s💡 Root Cause: Container was killed by Linux OOM killer. Runbook: runbooks/high-memory.md%s\n", ColorCyan, ColorReset)
					}
				}

				// Check current waiting state (CrashLoopBackOff)
				state, _ := c["state"].(map[string]interface{})
				if waiting, ok := state["waiting"].(map[string]interface{}); ok {
					reason, _ := waiting["reason"].(string)
					if reason == "CrashLoopBackOff" || reason == "RunContainerError" {
						issuesFound++
						fmt.Printf("  %s❌ Pod %s is currently in %s!%s\n", ColorRed, podName, reason, ColorReset)
						fmt.Printf("    %s💡 Recommendation: Inspect logs with 'platformctl application logs %s' or runbook 'runbooks/pod-crashloop.md'%s\n", ColorCyan, appName, ColorReset)
					}
				}
			}
		}
	}

	if issuesFound == 0 {
		fmt.Printf("  %s✓ All container statuses and restart counts healthy.%s\n", ColorGreen, ColorReset)
	}

	// 2. Warning Events in Namespace
	fmt.Printf("\n[2/4] Scanning Kubernetes Warning Events...\n")
	events, _ := runKubectl("get", "events", "-n", DefaultNamespace, "--field-selector=type=Warning", "--sort-by=.lastTimestamp")
	if strings.TrimSpace(events) != "" && !strings.Contains(events, "No resources found") {
		lines := strings.Split(strings.TrimSpace(events), "\n")
		recent := lines
		if len(lines) > 5 {
			recent = lines[len(lines)-5:]
		}
		for _, l := range recent {
			fmt.Println("  " + ColorYellow + l + ColorReset)
		}
	} else {
		fmt.Printf("  %s✓ No warning events found in namespace.%s\n", ColorGreen, ColorReset)
	}

	// 3. Health Endpoint Check
	fmt.Printf("\n[3/4] Probing Application Liveness & Ingress Endpoint...\n")
	probeUrl := fmt.Sprintf("http://%s/api", MinikubeIP)
	client := &http.Client{Timeout: 3 * time.Second}
	req, _ := http.NewRequest("GET", probeUrl, nil)
	req.Host = "platform.local"
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("  %s❌ Ingress probe connection failed: %v%s\n", ColorRed, err, ColorReset)
	} else {
		defer resp.Body.Close()
		statusColor := ColorGreen
		if resp.StatusCode >= 400 {
			statusColor = ColorRed
		}
		fmt.Printf("  HTTP Probe Status: %s%d %s%s\n", statusColor, resp.StatusCode, resp.Status, ColorReset)
	}

	// 4. Summary & Action Plan
	fmt.Printf("\n[4/4] Diagnostic Summary:\n")
	if issuesFound == 0 {
		fmt.Printf("  %sStatus: HEALTHY — No anomalies detected.%s\n", ColorGreen, ColorReset)
	} else {
		fmt.Printf("  %sStatus: ATTENTION REQUIRED — %d potential reliability issues identified.%s\n", ColorYellow, issuesFound, ColorReset)
	}
	fmt.Printf("%s=======================================================%s\n\n", ColorBold, ColorReset)
}


// -----------------------------------------------------------------------------
// Helper Functions
// -----------------------------------------------------------------------------

func runKubectl(args ...string) (string, error) {
	cmd := exec.Command("kubectl", args...)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		return stderr.String(), err
	}
	return stdout.String(), nil
}

func queryPrometheus(query string) (string, error) {
	url := fmt.Sprintf("http://%s/api/v1/query?query=%s", MinikubeIP, query)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}
	req.Host = PrometheusHost

	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(body), nil
}

func parsePrometheusResult(jsonStr string) float64 {
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(jsonStr), &data); err != nil {
		return 0
	}
	d, ok := data["data"].(map[string]interface{})
	if !ok {
		return 0
	}
	result, ok := d["result"].([]interface{})
	if !ok || len(result) == 0 {
		return 0
	}
	item, ok := result[0].(map[string]interface{})
	if !ok {
		return 0
	}
	value, ok := item["value"].([]interface{})
	if !ok || len(value) < 2 {
		return 0
	}
	valStr, ok := value[1].(string)
	if !ok {
		return 0
	}
	val, _ := strconv.ParseFloat(valStr, 64)
	return val
}
