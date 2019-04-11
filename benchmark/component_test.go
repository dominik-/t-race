package benchmark

import (
	"strings"
	"testing"
)

var filenameValid = "deployment-sample-valid.yaml"
var filenameTestWrite = "deployment-write-test.yaml"

func TestReadFromYamlFile(t *testing.T) {
	deployment, err := readFromYamlFile(filenameValid)
	if err != nil {
		t.Fatalf("YAML parse error: %v", err)
	}
	if strings.Compare(deployment.Services[0].Identifier, "comp01") != 0 {
		t.Fail()
	}
	if len(deployment.Services[0].Units) != 2 {
		t.Fail()
	}
	t.Logf("Successor 0:%v", deployment.Services[0].Units[0].SuccessorRef)
	t.Logf("Successor 1:%v", deployment.Services[0].Units[1].SuccessorRef)
}

func TestWorkTypeParsing(t *testing.T) {
	deployment, _ := ParseDeploymentDescription(filenameValid)
	t.Logf("Work found: %v", deployment.Services[0].Units[0].WorkUnit)
	if deployment.Services[0].Units[0].WorkUnit != deployment.WorkUnits[0] {
		t.Fail()
	}
}
