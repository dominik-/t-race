package model

import (
	"errors"
	"log"
	"os"
	"sort"
	"strings"

	"gopkg.in/yaml.v2"
)

//Ccalltype enum alias.
type calltype int

//This basically constitutes the enum. Calltype is an integer with two static representations.
const (
	SYNC calltype = iota
	ASYNC
)

//The stringified values of the enum
var calltypeNames = [...]string{
	"SYNC",
	"ASYNC",
}

//String returns the static string-values of calltype.
func (c calltype) String() string {
	return calltypeNames[c]
}

//UnmarshalYAML implements custom unmarshalling for the calltype in YAML files in order to use SYNC/ASYNC instead of 0 and 1.
func (c calltype) UnmarshalYAML(unmarshal func(value interface{}) error) error {
	var stringValue string
	err := unmarshal(&stringValue)
	if err != nil {
		return err
	}
	if strings.Compare(strings.ToLower(stringValue), "sync") == 0 {
		c = SYNC
	} else if strings.Compare(strings.ToLower(stringValue), "async") == 0 {
		c = ASYNC
	} else {
		return errors.New("couldnt parse calltype, unknown value. must be either 'SYNC' or 'ASYNC'")
	}
	return nil
}

//Services is a container for the root yaml element.
type Services struct {
	RootService *Service `yaml:"service,flow"`
}

//Service models a traced service, as it would be deployed by a user.
type Service struct {
	//Identifier is an arbitrary, unique identifier of the modeled service.
	Identifier string `yaml:"id"`
	//Deployment key is a representation of an environment, to which services are deployed. If two services share the same deployment key, they would be deployed on the same VM.
	DeploymentKey string `yaml:"deploymentKey"`
	//Work represents the total amount of work done by this service, in nanoseconds. This should include possible interleaved work, that is done between calls to successors. Important for "callers" of this service, as they use this to emulate span duration.
	Work int `yaml:"work"`
	//CallType is the way THIS service is called, can be either SYNC or ASYNC. If it is ASYNC, that means the parent service does not wait until a response is returned, i.e. call to next service after an async call is treated as parallel.
	CallType calltype `yaml:"calltype"`
	//Successors are the services called by this service. For each successor, a span is generated. (For all successors with SYNC, their "Work" is used as delay before starting the span corresponding to the next service call TODO: do we need/want this?). Furthermore, SYNC causes the successors Work to be added to this services Work. All ASYNC spans are started in parallel.
	Successors []*Service `yaml:"successors,flow"`
	//The effective work to be done by this service, which is the sum of its internal work, integrated with the effective work of all successors. This field is skipped during parsing (key '-').
	EffectiveWork int `yaml:"-"`
}

//ParseServiceDescription parses a YAML file containing a service / deployment description. Returns the root service or respective parsing errors, if the file was invalid.
func ParseServiceDescription(yamlFile string) (*Service, error) {
	root, err := readFromYamlFile(yamlFile)
	if err != nil {
		log.Printf("Could not parse service description, error was: %v", err)
		return nil, err
	}
	calculateEffectiveWorkRecursively(root)
	return root, nil
}

func readFromYamlFile(file string) (*Service, error) {
	fileHandle, err := os.Open(file)
	if err != nil {
		return nil, err
	}

	decoder := yaml.NewDecoder(fileHandle)
	var rootService Services

	err = decoder.Decode(&rootService)
	if err != nil {
		return nil, err
	}

	return rootService.RootService, nil
}

func calculateEffectiveWorkRecursively(s *Service) {
	//initialize effective work with own work
	s.EffectiveWork = s.Work
	if len(s.Successors) > 0 {
		asyncSuccessorWork := make([]int, 0)
		for _, successor := range s.Successors {
			calculateEffectiveWorkRecursively(successor)
			//if we have successors, their work is added if it is sync, otherwise add to list of async calls.
			if successor.CallType == SYNC {
				s.EffectiveWork += successor.EffectiveWork
			} else {
				asyncSuccessorWork = append(asyncSuccessorWork, successor.EffectiveWork)
			}
		}
		//if the highest async effective work call is longer than the combined effective work of all sync calls with own work, we need to adapt the effective work acccordingly
		sort.Ints(asyncSuccessorWork)
		if s.EffectiveWork < asyncSuccessorWork[len(asyncSuccessorWork)-1] {
			s.EffectiveWork = asyncSuccessorWork[len(asyncSuccessorWork)-1]
		}
	}
}
