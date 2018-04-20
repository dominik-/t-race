package model

import (
	"log"
	"os"

	"gopkg.in/yaml.v2"
)

type calltype int

//This basically is an enum. Calltype is actually an integer with two static representations.
const (
	SYNC calltype = iota
	ASYNC
)

//Service models a traced service, as it would be deployed by a user.
type Service struct {
	//Identifier is an arbitrary, unique identifier of the modeled service.
	Identifier string
	//Deployment key is a representation of an environment, to which services are deployed. If two services share the same deployment key, they would be deployed on the same VM.
	DeploymentKey string
	//Work represents the total amount of work done by this service, in nanoseconds. This should include possible interleaved work, that is done between calls to successors. Important for "callers" of this service, as they use this to emulate span duration.
	Work int64
	//CallType is the way THIS service is called, can be either SYNC or ASYNC. If it is ASYNC, that means the parent service does not wait until a response is returned, i.e. calls to multiple async services are basically treated as parallel.
	CallType calltype
	//Successors are the services called by this service. For each successor, a span is generated. (For all successors with SYNC, their "Work" is used as delay before starting the span corresponding to the next service call TODO: do we need/want this?). Furthermore, SYNC causes the successors Work to be added to this services Work. All ASYNC spans are started in parallel.
	Successors []*Service
}

func readFromYamlFile(file string) *Service {
	fileHandle, err := os.Open(file)
	if err != nil {
		log.Fatalf("YAML file with services could not be opened: %s/%s\n", file)
	}

	decoder := yaml.NewDecoder(fileHandle)
	var rootService Service

	err = decoder.Decode(&rootService)
	if err != nil {
		log.Fatalf("Service file could not be parsed: %v", err)
	}

	return &rootService
}
