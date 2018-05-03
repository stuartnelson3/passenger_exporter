package main

import (
	"flag"
	"math"
	"os"
	"reflect"
	"testing"
	"time"
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

		topLevelQueue := parseFloat(info.TopLevelRequestsInQueue)
		if topLevelQueue == 0 {
			t.Fatalf("%v: no queuing requests parsed from output", name)
		}

		for _, sg := range info.SuperGroups {
			if want, got := "/src/app/my_app", sg.Group.Options.AppRoot; want != got {
				t.Fatalf("%s: incorrect app_root: wanted %s, got %s", name, want, got)
			}

			queue := parseFloat(sg.RequestsInQueue)
			if queue == math.NaN() {
				t.Fatalf("%v: failed to parse requests in queue", name)
			}

			if len(sg.Group.Processes) == 0 {
				t.Fatalf("%v: no processes in output", name)
			}
			for _, proc := range sg.Group.Processes {
				if want, got := "2254", proc.ProcessGroupID; want != got {
					t.Fatalf("%s: incorrect process_group_id: wanted %s, got %s", name, want, got)
				}
			}
		}
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

type updateProcessSpec struct {
	name      string
	input     map[string]int
	processes []Process
	output    map[string]int
}

func newUpdateProcessSpec(
	name string,
	input map[string]int,
	processes []Process,
) updateProcessSpec {
	s := updateProcessSpec{
		name:      name,
		input:     input,
		processes: processes,
	}
	s.output = updateProcesses(s.input, s.processes)
	return s
}

func TestUpdateProcessIdentifiers(t *testing.T) {
	for _, spec := range []updateProcessSpec{
		newUpdateProcessSpec(
			"empty input",
			map[string]int{},
			[]Process{
				Process{PID: "abc"},
				Process{PID: "cdf"},
				Process{PID: "dfe"},
			},
		),
		newUpdateProcessSpec(
			"1:1",
			map[string]int{
				"abc": 0,
				"cdf": 1,
				"dfe": 2,
			},
			[]Process{
				Process{PID: "abc"},
				Process{PID: "cdf"},
				Process{PID: "dfe"},
			},
		),
		newUpdateProcessSpec(
			"increase processes",
			map[string]int{
				"abc": 0,
				"cdf": 1,
				"dfe": 2,
			},
			[]Process{
				Process{PID: "abc"},
				Process{PID: "cdf"},
				Process{PID: "dfe"},
				Process{PID: "ghi"},
				Process{PID: "jkl"},
				Process{PID: "lmn"},
			},
		),
		newUpdateProcessSpec(
			"reduce processes",
			map[string]int{
				"abc": 0,
				"cdf": 1,
				"dfe": 2,
				"ghi": 3,
				"jkl": 4,
				"lmn": 5,
			},
			[]Process{
				Process{PID: "abc"},
				Process{PID: "cdf"},
				Process{PID: "dfe"},
			},
		),
	} {
		if len(spec.output) != len(spec.processes) {
			t.Fatalf("case %s: proceses improperly copied to output: len(output) (%d) does not match len(processes) (%d)", spec.name, len(spec.output), len(spec.processes))
		}

		for _, p := range spec.processes {
			if _, ok := spec.output[p.PID]; !ok {
				t.Fatalf("case %s: pid not copied into map", spec.name)
			}
		}

		newOutput := updateProcesses(spec.output, spec.processes)
		if !reflect.DeepEqual(newOutput, spec.output) {
			t.Fatalf("case %s: updateProcesses is not idempotent", spec.name)
		}
	}
}

func TestInsertingNewProcesses(t *testing.T) {
	spec := newUpdateProcessSpec(
		"inserting processes",
		map[string]int{
			"abc": 0,
			"cdf": 1,
			"dfe": 2,
			"efg": 3,
		},
		[]Process{
			Process{PID: "abc"},
			Process{PID: "dfe"},
			Process{PID: "newPID"},
			Process{PID: "newPID2"},
		},
	)

	if len(spec.output) != len(spec.processes) {
		t.Fatalf("case %s: proceses improperly copied to output: len(output) (%d) does not match len(processes) (%d)", spec.name, len(spec.output), len(spec.processes))
	}

	if want, got := 1, spec.output["newPID"]; want != got {
		t.Fatalf("updateProcesses did not correctly map the new PID: wanted %d, got %d", want, got)
	}
	if want, got := 3, spec.output["newPID2"]; want != got {
		t.Fatalf("updateProcesses did not correctly map the new PID: wanted %d, got %d", want, got)
	}
}

func newTestExporter() *Exporter {
	return NewExporter("cat ./testdata/passenger_xml_output.xml", time.Second)
}
