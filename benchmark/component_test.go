package benchmark

import (
	"os"
	"strings"
	"testing"

	"gopkg.in/yaml.v2"
)

var filenameValid = "deployment-sample-valid.yaml"
var filenameTestWrite = "deployment-write-test.yaml"

func TestCreateYamlFile(t *testing.T) {
	comp1 := &Component{
		Identifier:     "c01",
		CallType:       SYNC,
		EnvironmentRef: "env01",
		SinkRef:        "agent1",
		SuccessorRefs:  []string{"c02", "c03"},
		Tags:           map[string]string{"Tag1": "Value1"},
	}

	comp2 := &Component{
		Identifier:     "c02",
		CallType:       ASYNC,
		EnvironmentRef: "env02",
		SinkRef:        "agent2",
		SuccessorRefs:  []string{"c03"},
	}

	comp3 := &Component{
		Identifier:     "c03",
		CallType:       ASYNC,
		EnvironmentRef: "env02",
		SinkRef:        "agent2",
	}

	env1 := &Environment{
		Identifier: "env01",
		Provider:   "static",
	}

	env2 := &Environment{
		Identifier: "env02",
		Provider:   "static",
	}

	s1 := &Sink{
		Identifier:     "agent1",
		EnvironmentRef: "env01",
	}

	s2 := &Sink{
		Identifier:     "agent2",
		EnvironmentRef: "env02",
	}

	deployment := &Deployment{
		Name:         "deploymentTestSerialize",
		Components:   []*Component{comp1, comp2, comp3},
		Environments: []*Environment{env1, env2},
		Sinks:        []*Sink{s1, s2},
	}

	fileHandle, _ := os.Create(filenameTestWrite)
	enc := yaml.NewEncoder(fileHandle)
	enc.Encode(deployment)
	enc.Close()
	//TODO validate file somehow...?
	os.Remove(filenameTestWrite)

}
func TestReadFromYamlFile(t *testing.T) {
	deployment, err := readFromYamlFile(filenameValid)
	if err != nil {
		t.Fatalf("YAML parse error: %v", err)
	}
	if strings.Compare(deployment.Components[0].Identifier, "svc01") != 0 {
		t.Fail()
	}
	if len(deployment.Components[0].SuccessorRefs) != 2 {
		t.Fail()
	}
	t.Logf("Successor 0:%v", deployment.Components[0].SuccessorRefs[0])
	t.Logf("Successor 1:%v", deployment.Components[0].SuccessorRefs[1])
}

func TestCallTypeValuesParsing(t *testing.T) {
	deployment, _ := ParseDeploymentDescription(filenameValid)
	if deployment.Components[0].CallType != SYNC {
		t.Fail()
	}
	if deployment.Components[1].CallType != ASYNC {
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
