package main

import (
	"os"
	"testing"
)

func TestParseOutput(t *testing.T) {
	f, err := os.Open("passenger_xml_output.xml")
	if err != nil {
		t.Fatalf("open xml file failed: %v", err)
	}

	info, err := parseOutput(f)
	if err != nil {
		t.Fatalf("parse xml file failed: %v", err)
	}

	for _, sg := range info.SuperGroups {
		if want, got := "/src/app/my_app", sg.Group.Options.AppRoot; want != got {
			t.Fatalf("incorrect app_root: wanted %s, got %s", want, got)
		}

		for _, proc := range sg.Group.Processes {
			if want, got := "2254", proc.ProcessGroupID; want != got {
				t.Fatalf("incorrect process_group_id: wanted %s, got %s", want, got)
			}
		}
	}
}
