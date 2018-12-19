package benchmark

import (
	"os"
	"reflect"
	"strings"
	"testing"

	"gopkg.in/yaml.v2"
)

var filenameValid = "deployment-sample-valid.yaml"
var filenameTestWrite = "deployment-write-test.yaml"

func TestCreateYamlFile(t *testing.T) {

	deployment := &Deployment{
		Name:         "deploymentTestSerialize",
		Components:   []*Component{},
		Environments: []*Environment{},
		Sinks:        []*Sink{},
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
	if strings.Compare(deployment.Components[0].Identifier, "comp01") != 0 {
		t.Fail()
	}
	if len(deployment.Components[0].Units) != 2 {
		t.Fail()
	}
	t.Logf("Successor 0:%v", deployment.Components[0].Units[0].SuccessorRef)
	t.Logf("Successor 1:%v", deployment.Components[0].Units[1].SuccessorRef)
}

func TestWorkTypeParsing(t *testing.T) {
	deployment, _ := ParseDeploymentDescription(filenameValid)
	t.Logf("Work found: %v", deployment.Components[0].Units[0].WorkClass)
	if reflect.TypeOf(deployment.Components[0].Units[0].WorkClass) != reflect.TypeOf(new(ConstantWork)) {
		t.Fail()
	}
}
