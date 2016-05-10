package main

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"time"
)

func TestParsing(t *testing.T) {
	tests := map[string]func(t *testing.T) *Info{
		"newExporter": func(t *testing.T) *Info {
			e := NewExporter("cat ./passenger_xml_output.xml")
			info, err := e.status()
			if err != nil {
				t.Fatalf("failed to get status: %v", err)
			}
			return info
		},
		"parseOutput": func(t *testing.T) *Info {
			f, err := os.Open("passenger_xml_output.xml")
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

func testParsePassengerDuration(t *testing.T, uptime string) time.Duration {
	parsed, err := parsePassengerInterval(uptime)
	if err != nil {
		t.Fatalf("%v", err)
	}
	return parsed
}
