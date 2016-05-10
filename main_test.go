package main

import (
	"os"
	"testing"
)

func TestParsing(t *testing.T) {
	tests := map[string]func() *Info{
		"newExporter": func() *Info {
			e := NewExporter("cat ./passenger_xml_output.xml")
			info, err := e.status()
			if err != nil {
				t.Fatalf("failed to get status: %v", err)
			}
			return info
		},
		"parseOutput": func() *Info {
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
		info := test()

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
			}
		}
	}
}
