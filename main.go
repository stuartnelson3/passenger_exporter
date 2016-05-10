package main

import (
	"bytes"
	"encoding/xml"
	"flag"
	"io"
	"math"
	"net/http"
	"os/exec"
	"strconv"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/log"
)

const (
	namespace = "passenger_nginx"
)

var (
	Version = "0.0.0"
)

// Exporter collects metrics from a passenger-nginx integration.
type Exporter struct {
	// binary file path for querying passenger state.
	cmd  string
	args []string

	// Passenger metrics.
	up                  *prometheus.Desc
	uptime              *prometheus.Desc
	version             *prometheus.Desc
	toplevelQueue       *prometheus.Desc
	maxProcessCount     *prometheus.Desc
	currentProcessCount *prometheus.Desc
	appCount            *prometheus.Desc

	// App metrics.
	appQueue         *prometheus.Desc
	appProcsSpawning *prometheus.Desc

	// Process metrics.
	requestsProcessed *prometheus.Desc
	procUptime        *prometheus.Desc
	procMemory        *prometheus.Desc
	procStatus        *prometheus.Desc
	// There are several Process level metrics that could be of interest,
	// but I imagine they're already handled by node_exporter? Not sure
	// on their granularity. e.g. passenger status exposes swap memory
	// usage on the process, need to check if node_exporter allows
	// per-process swap metrics.
}

func NewExporter(cmd string) *Exporter {
	cmdComponents := strings.Split(cmd, " ")

	return &Exporter{
		cmd:  cmdComponents[0],
		args: cmdComponents[1:],
		up: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "", "up"),
			"Could passenger status be queried.",
			nil,
			nil,
		),
		uptime: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "", "uptime"),
			"Number of seconds since passenger started.",
			nil,
			nil,
		),
		version: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "", "version"),
			"Version of passenger",
			[]string{"version"},
			nil,
		),
		toplevelQueue: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "", "top_level_queue"),
			"Number of requests in the top-level queue.",
			nil,
			nil,
		),
		maxProcessCount: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "", "max_process_count"),
			"Configured maximum number of processes.",
			nil,
			nil,
		),
		currentProcessCount: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "", "current_process_count"),
			"Current number of processes.",
			nil,
			nil,
		),
		appCount: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "", "app_count"),
			"Number of apps.",
			nil,
			nil,
		),
		appQueue: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "", "app_queue"),
			"Number of requests in app process queues.",
			[]string{"name"},
			nil,
		),
		appProcsSpawning: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "", "app_procs_spawning"),
			"Number of processes spawning.",
			[]string{"name"},
			nil,
		),
		requestsProcessed: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "", "requests_processed"),
			"Number of processes served by a process.",
			[]string{"name", "pid"},
			nil,
		),
		procUptime: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "", "proc_uptime"),
			"Number of seconds since processor started.",
			[]string{"name", "pid"},
			nil,
		),
		procMemory: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "", "proc_memory"),
			"Memory consumed by a process",
			[]string{"name", "pid"},
			nil,
		),
		procStatus: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "", "proc_status"),
			"Running status for a process.",
			[]string{"name", "pid", "codeRevision", "lifeStatus", "enabled"},
			nil,
		),
	}
}

// Collect fetches the statistics from the configured memcached server, and
// delivers them as Prometheus metrics. It implements prometheus.Collector.
func (e *Exporter) Collect(ch chan<- prometheus.Metric) {
	info, err := e.status()
	if err != nil {
		ch <- prometheus.MustNewConstMetric(e.up, prometheus.GaugeValue, 0)
		log.Errorf("failed to collect status from passenger: %s", err)
		return
	}
	ch <- prometheus.MustNewConstMetric(e.up, prometheus.GaugeValue, 1)

	// TODO: Doesn't seem to be a queryable uptime on the main process.
	// Talk with @grobie about how to handle this.

	ch <- prometheus.MustNewConstMetric(e.version, prometheus.GaugeValue, 1, info.PassengerVersion)

	// This value should be 0. Not sure if I should use a gauge or a
	// counter, as we shouldn't be counting anything.
	ch <- prometheus.MustNewConstMetric(e.toplevelQueue, prometheus.CounterValue, parseFloat(info.TopLevelRequestsInQueue))
	ch <- prometheus.MustNewConstMetric(e.maxProcessCount, prometheus.GaugeValue, parseFloat(info.MaxProcessCount))
	ch <- prometheus.MustNewConstMetric(e.currentProcessCount, prometheus.GaugeValue, parseFloat(info.CurrentProcessCount))
	ch <- prometheus.MustNewConstMetric(e.appCount, prometheus.GaugeValue, parseFloat(info.AppCount))

	for _, sg := range info.SuperGroups {
		ch <- prometheus.MustNewConstMetric(e.appQueue, prometheus.GaugeValue, parseFloat(sg.RequestsInQueue), sg.Name)
		ch <- prometheus.MustNewConstMetric(e.appProcsSpawning, prometheus.GaugeValue, parseFloat(sg.RequestsInQueue), sg.Group.ProcessesSpawning, sg.Name)

		for _, proc := range sg.Group.Processes {
			ch <- prometheus.MustNewConstMetric(e.procMemory, prometheus.GaugeValue, parseFloat(proc.RealMemory), sg.Name, proc.PID)
			ch <- prometheus.MustNewConstMetric(e.requestsProcessed, prometheus.CounterValue, parseFloat(proc.RequestsProcessed), sg.Name, proc.PID)
			ch <- prometheus.MustNewConstMetric(e.procUptime, prometheus.CounterValue, parseFloat(proc.Uptime), sg.Name, proc.PID)

			// Is this one really necessary?
			ch <- prometheus.MustNewConstMetric(
				e.procStatus, prometheus.CounterValue, 1,
				sg.Name, proc.PID, proc.CodeRevision, proc.LifeStatus, proc.Enabled,
			)
		}
	}

}

func (e *Exporter) status() (*Info, error) {
	var (
		out bytes.Buffer
		cmd = exec.Command(e.cmd, e.args...)
	)
	cmd.Stdout = &out

	err := cmd.Run()
	if err != nil {
		return nil, err
	}

	return parseOutput(&out)
}

func (e *Exporter) Describe(ch chan<- *prometheus.Desc) {
	ch <- e.up
	ch <- e.uptime
	ch <- e.version
	ch <- e.toplevelQueue
	ch <- e.maxProcessCount
	ch <- e.currentProcessCount
	ch <- e.appCount
	ch <- e.appQueue
	ch <- e.appProcsSpawning
	ch <- e.requestsProcessed
	ch <- e.procUptime
	ch <- e.procMemory
	ch <- e.procStatus
}

func parseOutput(r io.Reader) (*Info, error) {
	var info Info
	err := xml.NewDecoder(r).Decode(&info)
	if err != nil {
		return nil, err
	}
	return &info, nil
}

func parseFloat(val string) float64 {
	v, err := strconv.ParseFloat(val, 64)
	if err != nil {
		log.Errorf("failed to parse %s: %v", val, err)
		v = math.NaN()
	}
	return v
}

func main() {
	var (
		cmd           = flag.String("passenger.command", "passenger-status --show=xml", "Passenger command for querying passenger status.")
		metricsPath   = flag.String("web.telemetry-path", "/metrics", "Path under which to expose metrics.")
		listenAddress = flag.String("web.listen-address", ":9106", "Address to listen on for web interface and telemetry.")
	)
	flag.Parse()

	prometheus.MustRegister(NewExporter(*cmd))

	http.Handle(*metricsPath, prometheus.Handler())

	log.Infof("starting passenger_exporter_nginx v%s at %s", Version, *listenAddress)
	log.Fatal(http.ListenAndServe(*listenAddress, nil))
}
