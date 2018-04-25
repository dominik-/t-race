package model

import (
	"strings"
	"testing"
)

func TestReadFromYamlFile(t *testing.T) {
	service, _ := readFromYamlFile("flow-test.yaml")
	if strings.Compare(service.Identifier, "root") != 0 {
		t.Fail()
	}
	t.Logf("root:%v", service)
	t.Logf("Successor 0:%v", service.Successors[0])
	t.Logf("Successor 1:%v", service.Successors[1])
	if len(service.Successors) != 2 {
		t.Fail()
	}
}

func TestCallTypeValuesParsing(t *testing.T) {
	service, _ := readFromYamlFile("flow-test.yaml")
	if service.CallType != SYNC {
		t.Fail()
	}
	if service.Successors[1].CallType != ASYNC {
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
}
func TestCalculateEffectiveWorkAsyncOverride(t *testing.T) {
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
	calculateEffectiveWorkRecursively(rootSvc2)
	if rootSvc2.EffectiveWork != 350 {
		t.Fail()
	}
}
