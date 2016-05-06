package main

import (
	"encoding/xml"
	"fmt"
	"os"
	"testing"
)

func TestParseWorks(t *testing.T) {
	f, err := os.Open("passenger_xml_output.xml")
	if err != nil {
		t.Fatalf("open xml file failed: %v", err)
	}

	info, err := parse(f)
	if err != nil {
		t.Fatalf("parse xml file failed: %v", err)
	}

	for _, sg := range info.SuperGroups {
		for _, proc := range sg.Group.Processes {

			byts, err := xml.MarshalIndent(proc, "  ", "    ")
			if err != nil {
				t.Fatalf("marshal indent process failed: %v", err)
			}
			fmt.Printf("%v\n", string(byts))
		}
	}
	t.Fail()
}
