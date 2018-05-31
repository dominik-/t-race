package benchmark

import (
	"strings"
	"testing"
)

func TestReadFromYamlFile(t *testing.T) {
	component, _ := readFromYamlFile("flow-test.yaml")
	if strings.Compare(component.Identifier, "root") != 0 {
		t.Fail()
	}
	t.Logf("root:%v", component)
	t.Logf("Successor 0:%v", component.Successors[0])
	t.Logf("Successor 1:%v", component.Successors[1])
	if len(component.Successors) != 2 {
		t.Fail()
	}
}

func TestCallTypeValuesParsing(t *testing.T) {
	component, _ := readFromYamlFile("flow-test.yaml")
	if component.CallType != SYNC {
		t.Fail()
	}
	if component.Successors[1].CallType != ASYNC {
		t.Fail()
	}
}
func TestCalculateEffectiveWork(t *testing.T) {
	rootSvc1 := &Component{
		Work: 100,
		Successors: []*Component{
			&Component{
				Work:     200,
				CallType: SYNC,
			},
			&Component{
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
	rootSvc2 := &Component{
		Work: 100,
		Successors: []*Component{
			&Component{
				Work:     200,
				CallType: SYNC,
			},
			&Component{
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
