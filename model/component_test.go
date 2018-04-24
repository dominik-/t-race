package model

import (
	"strings"
	"testing"
)

func TestReadFromYamlFile(t *testing.T) {
	service := readFromYamlFile("flow.yaml")
	t.Logf("parsed: %v", service)
	if strings.Compare(service.Identifier, "root") != 0 {
		t.Fail()
	}
	if len(service.Successors) != 2 {
		t.Fail()
	}
}

func TestCalculateEffectiveWork(t *testing.T) {
	rootSvc1 := &Service{
		Work: 100,
		Successors: []*Service{
			&Service{
				Work:     200,
				CallType: SYNC,
			},
			&Service{
				Work:     200,
				CallType: ASYNC,
			},
		},
	}
	calculateEffectiveWorkRecursively(rootSvc1)
	if rootSvc1.EffectiveWork != 300 {
		t.Fail()
	}
	rootSvc2 := &Service{
		Work: 100,
		Successors: []*Service{
			&Service{
				Work:     200,
				CallType: SYNC,
			},
			&Service{
				Work:     350,
				CallType: ASYNC,
			},
		},
	}
	if rootSvc2.EffectiveWork != 350 {
		t.Fail()
	}
}
