package main

import (
	"bytes"
	"encoding/xml"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"math"
	"net/http"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/log"
)

const (
	namespace = "passenger_nginx"
)

var (
	Version = "0.0.0"

	timeoutErr = errors.New("passenger-status command timed out")

	processIdentifiers = make(map[string]int)
)

// Exporter collects metrics from a passenger-nginx integration.
type Exporter struct {
	// binary file path for querying passenger state.
	cmd  string
	args []string

	// Passenger command timeout.
	timeout time.Duration

	// Passenger metrics.
	up                  *prometheus.Desc
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
}

func NewExporter(cmd string, timeout time.Duration) *Exporter {
	cmdComponents := strings.Split(cmd, " ")

	return &Exporter{
		cmd:     cmdComponents[0],
		args:    cmdComponents[1:],
		timeout: timeout,
		up: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "", "up"),
			"Could passenger status be queried.",
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
			prometheus.BuildFQName(namespace, "", "max_processes"),
			"Configured maximum number of processes.",
			nil,
			nil,
		),
		currentProcessCount: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "", "current_processes"),
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
			prometheus.BuildFQName(namespace, "", "requests_processed_total"),
			"Number of processes served by a process.",
			[]string{"name", "id"},
			nil,
		),
		procUptime: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "", "proc_uptime"),
			"Number of seconds since processor started.",
			[]string{"name", "id", "codeRevision"},
			nil,
		),
		procMemory: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "", "proc_memory"),
			"Memory consumed by a process",
			[]string{"name", "id"},
			nil,
		),
	}
}

// Collect fetches the statistics from the configured passenger frontend, and
// delivers them as Prometheus metrics. It implements prometheus.Collector.
func (e *Exporter) Collect(ch chan<- prometheus.Metric) {
	info, err := e.status()
	if err != nil {
		ch <- prometheus.MustNewConstMetric(e.up, prometheus.GaugeValue, 0)
		log.Errorf("failed to collect status from passenger: %s", err)
		return
	}
	ch <- prometheus.MustNewConstMetric(e.up, prometheus.GaugeValue, 1)

	ch <- prometheus.MustNewConstMetric(e.version, prometheus.GaugeValue, 1, info.PassengerVersion)

	ch <- prometheus.MustNewConstMetric(e.toplevelQueue, prometheus.GaugeValue, parseFloat(info.TopLevelRequestsInQueue))
	ch <- prometheus.MustNewConstMetric(e.maxProcessCount, prometheus.GaugeValue, parseFloat(info.MaxProcessCount))
	ch <- prometheus.MustNewConstMetric(e.currentProcessCount, prometheus.GaugeValue, parseFloat(info.CurrentProcessCount))
	ch <- prometheus.MustNewConstMetric(e.appCount, prometheus.GaugeValue, parseFloat(info.AppCount))

	for _, sg := range info.SuperGroups {
		ch <- prometheus.MustNewConstMetric(e.appQueue, prometheus.GaugeValue, parseFloat(sg.RequestsInQueue), sg.Name)
		ch <- prometheus.MustNewConstMetric(e.appProcsSpawning, prometheus.GaugeValue, parseFloat(sg.Group.ProcessesSpawning), sg.Name)

		// Update process identifiers map.
		processIdentifiers = updateProcesses(processIdentifiers, sg.Group.Processes)
		for _, proc := range sg.Group.Processes {
			if bucketID, ok := processIdentifiers[proc.PID]; ok {
				ch <- prometheus.MustNewConstMetric(e.procMemory, prometheus.GaugeValue, parseFloat(proc.RealMemory), sg.Name, strconv.Itoa(bucketID))
				ch <- prometheus.MustNewConstMetric(e.requestsProcessed, prometheus.CounterValue, parseFloat(proc.RequestsProcessed), sg.Name, strconv.Itoa(bucketID))

				if uptime, err := parsePassengerInterval(proc.Uptime); err == nil {
					ch <- prometheus.MustNewConstMetric(e.procUptime, prometheus.GaugeValue, float64(uptime),
						sg.Name, strconv.Itoa(bucketID), proc.CodeRevision,
					)
				}
			}
		}
	}

}

func (e *Exporter) status() (*Info, error) {
	var (
		out bytes.Buffer
		cmd = exec.Command(e.cmd, e.args...)
	)
	cmd.Stdout = &out

	err := cmd.Start()
	if err != nil {
		return nil, err
	}

	errc := make(chan error, 1)
	go func(cmd *exec.Cmd, c chan<- error) {
		c <- cmd.Wait()
	}(cmd, errc)

	select {
	case err := <-errc:
		if err != nil {
			return nil, err
		}
	case <-time.After(e.timeout):
		return nil, timeoutErr
	}

	return parseOutput(&out)
}

func (e *Exporter) Describe(ch chan<- *prometheus.Desc) {
	ch <- e.up
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

// updateProcesses updates the global map from process id:exporter id. Process
// TTLs cause new processes to be created on a user-defined cycle. When a new
// process replaces an old process, the new process's statistics will be
// bucketed with those of the process it replaced.
// Processes are restarted at an offset, user-defined interval. The
// restarted process is appended to the end of the status output.  For
// maintaining consistent process identifiers between process starts,
// pids are mapped to an identifier based on process count. When a new
// process/pid appears, it is mapped to either the first empty place
// within the global map storing process identifiers, or mapped to
// pid:id pair in the map.
func updateProcesses(old map[string]int, processes []Process) map[string]int {
	var (
		updated = make(map[string]int)
		found   = make([]string, len(old))
		missing []string
	)

	for _, p := range processes {
		if id, ok := old[p.PID]; ok {
			found[id] = p.PID
			// id also serves as an index.
			// By putting the pid at a certain index, we can loop
			// through the array to find the values that are the 0
			// value (empty string).
			// If index i has the empty value, then it was never
			// updated, so we slot the first of the missingPIDs
			// into that position. Passenger-status orders output
			// by pid, increasing. We can then assume that
			// unclaimed pid positions map in order to the missing
			// pids.
		} else {
			missing = append(missing, p.PID)
		}
	}

	j := 0
	for i, pid := range found {
		if pid == "" {
			if j >= len(missing) {
				continue
			}
			pid = missing[j]
			j++
		}
		updated[pid] = i
	}

	// If the number of elements in missing iterated through is less
	// than len(missing), there are new elements to be added to the map.
	// Unused pids from the last collection are not copied from old to
	// updated, thereby cleaning the return value of unused PIDs.
	if j < len(missing) {
		count := len(found)
		for i, pid := range missing[j:] {
			updated[pid] = count + i
		}
	}

	return updated
}

// parsePassengerInterval formats and parses the default Passenger time output.
func parsePassengerInterval(val string) (time.Duration, error) {
	return time.ParseDuration(strings.Replace(val, " ", "", -1))
}

func main() {
	var (
		cmd           = flag.String("passenger.command", "passenger-status --show=xml", "Passenger command for querying passenger status.")
		timeout       = flag.Duration("passenger.command.timeout", 500*time.Millisecond, "Timeout for passenger.command.")
		pidFile       = flag.String("passenger.pid-file", "", "Optional path to a file containing the passenger/nginx PID for additional metrics.")
		metricsPath   = flag.String("web.telemetry-path", "/metrics", "Path under which to expose metrics.")
		listenAddress = flag.String("web.listen-address", ":9106", "Address to listen on for web interface and telemetry.")
	)
	flag.Parse()

	if *pidFile != "" {
		prometheus.MustRegister(prometheus.NewProcessCollectorPIDFn(
			func() (int, error) {
				content, err := ioutil.ReadFile(*pidFile)
				if err != nil {
					return 0, fmt.Errorf("error reading pidfile %q: %s", *pidFile, err)
				}
				value, err := strconv.Atoi(strings.TrimSpace(string(content)))
				if err != nil {
					return 0, fmt.Errorf("error parsing pidfile %q: %s", *pidFile, err)
				}
				return value, nil
			},
			namespace),
		)
	}

	prometheus.MustRegister(NewExporter(*cmd, *timeout))

	http.Handle(*metricsPath, prometheus.Handler())

	log.Infof("starting passenger_exporter_nginx v%s at %s", Version, *listenAddress)
	log.Fatal(http.ListenAndServe(*listenAddress, nil))
}
