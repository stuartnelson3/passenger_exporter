package main

import (
	"bytes"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

var golden bool

func init() {
	flag.BoolVar(&golden, "golden", false, "Write test results to fixture files.")
	flag.Parse()
}

func TestParsing(t *testing.T) {
	tests := map[string]func(t *testing.T) *Info{
		"newExporter": func(t *testing.T) *Info {
			e := newTestExporter()
			info, err := e.status()
			if err != nil {
				t.Fatalf("failed to get status: %v", err)
			}
			return info
		},
		"parseOutput": func(t *testing.T) *Info {
			f, err := os.Open("./testdata/passenger_xml_output.xml")
			if err != nil {
				t.Fatalf("open xml file failed: %v", err)
			}

			info, err := parseOutput(f)
			if err != nil {
				t.Fatalf("parse xml file failed: %v", err)
			}
			return info
		},
	}

	for name, test := range tests {
		info := test(t)

		if len(info.SuperGroups) == 0 {
			t.Fatalf("%v: no supergroups in output", name)
		}
		for _, sg := range info.SuperGroups {
			if want, got := "/src/app/my_app", sg.Group.Options.AppRoot; want != got {
				t.Fatalf("%s: incorrect app_root: wanted %s, got %s", name, want, got)
			}

			if len(sg.Group.Processes) == 0 {
				t.Fatalf("%v: no processes in output", name)
			}
			for _, proc := range sg.Group.Processes {
				if want, got := "2254", proc.ProcessGroupID; want != got {
					t.Fatalf("%s: incorrect process_group_id: wanted %s, got %s", name, want, got)
				}

				uptime := fmt.Sprintf("%v", testParsePassengerDuration(t, proc.Uptime))
				if want, got := strings.Replace(proc.Uptime, " ", "", -1), uptime; want != got {
					t.Fatalf("%s: incorrect process_group_id: wanted %s, got %s", name, want, got)
				}
			}
		}
	}
}

func TestScrape(t *testing.T) {
	prometheus.MustRegister(newTestExporter())
	server := httptest.NewServer(prometheus.Handler())
	defer server.Close()

	res, err := http.Get(server.URL)
	if err != nil {
		t.Fatalf("failed to GET test server: %v", err)
	}
	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		t.Fatalf("failed to read response body: %v", err)
	}

	scrapeFixturePath := "./testdata/scrape_output.txt"
	if golden {
		idx := bytes.Index(body, []byte("# HELP passenger_nginx_app_count Number of apps."))
		ioutil.WriteFile(scrapeFixturePath, body[idx:], 0666)
		t.Skipf("--golden passed: re-writing %s", scrapeFixturePath)
	}

	fixture, err := ioutil.ReadFile(scrapeFixturePath)
	if err != nil {
		t.Fatalf("failed to read scrape fixture: %v", err)
	}

	if !bytes.Contains(body, fixture) {
		t.Fatalf("fixture data not contained within response body")
	}
}

func TestStatusTimeout(t *testing.T) {
	e := NewExporter("sleep 1", time.Millisecond)
	_, err := e.status()
	if err == nil {
		t.Fatalf("failed to timeout")
	}

	if err != timeoutErr {
		t.Fatalf("incorrect err: %v", err)
	}
}

func newTestExporter() *Exporter {
	return NewExporter("cat ./testdata/passenger_xml_output.xml", time.Second)
}

func testParsePassengerDuration(t *testing.T, uptime string) time.Duration {
	parsed, err := parsePassengerInterval(uptime)
	if err != nil {
		t.Fatalf("%v", err)
	}
	return parsed
}
