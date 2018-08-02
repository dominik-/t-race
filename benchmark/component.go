package benchmark

import (
	"errors"
	"log"
	"os"
	"sort"
	"strings"

	api "gitlab.tubit.tu-berlin.de/dominik-ernst/trace-writer-api"
	"gopkg.in/yaml.v2"
)

//Calltype enum alias.
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
func (c *calltype) UnmarshalYAML(unmarshal func(value interface{}) error) error {
	var stringValue string
	err := unmarshal(&stringValue)
	if err != nil {
		return err
	}
	stringValue = strings.ToLower(stringValue)
	//log.Printf("Parsed string value in YAML: %s\n", stringValue)
	if strings.Compare(stringValue, "sync") == 0 {
		*c = SYNC
	} else if strings.Compare(stringValue, "async") == 0 {
		*c = ASYNC
	} else {
		return errors.New("couldnt parse calltype, unknown value. must be either 'SYNC' or 'ASYNC'")
	}
	return nil
}

//Components is a container for the root yaml element.
type Components struct {
	RootComponent *Component `yaml:"component,flow"`
}

//Component models a traced component, as it would be deployed by a user.
type Component struct {
	//Identifier is an arbitrary, unique identifier of the modeled component.
	Identifier string `yaml:"id"`
	//Deployment key is a representation of an environment, to which components are deployed. If two components share the same deployment key, they would be deployed on the same VM.
	DeploymentKey string `yaml:"deploymentKey"`
	//Work represents the total amount of work done by this component, in nanoseconds. This should include possible interleaved work, that is done between calls to successors. Important for "callers" of this component, as they use this to emulate span duration.
	Work int `yaml:"work"`
	//CallType is the way THIS component is called, can be either SYNC or ASYNC. If it is ASYNC, that means the parent component does not wait until a response is returned, i.e. call to next component after an async call is treated as parallel.
	CallType calltype `yaml:"calltype"`
	//Successors are the components called by this component. For each successor, a span is generated. (For all successors with SYNC, their "Work" is used as delay before starting the span corresponding to the next component call TODO: do we need/want this?). Furthermore, SYNC causes the successors Work to be added to this components Work. All ASYNC spans are started in parallel.
	Successors []*Component `yaml:"successors,flow"`
	//The effective work to be done by this component, which is the sum of its internal work, integrated with the effective work of all successors. This field is skipped during parsing (key '-').
	EffectiveWork int `yaml:"-"`
}

//ParseComponentDescription parses a YAML file containing a component / deployment description. Returns the root component or respective parsing errors, if the file was invalid.
func ParseComponentDescription(yamlFile string) (*Component, error) {
	root, err := readFromYamlFile(yamlFile)
	if err != nil {
		log.Printf("Could not parse component description, error was: %v", err)
		return nil, err
	}
	calculateEffectiveWorkRecursively(root)
	return root, nil
}

func readFromYamlFile(file string) (*Component, error) {
	fileHandle, err := os.Open(file)
	if err != nil {
		return nil, err
	}

	decoder := yaml.NewDecoder(fileHandle)
	var rootComponent Components

	err = decoder.Decode(&rootComponent)
	if err != nil {
		return nil, err
	}

	return rootComponent.RootComponent, nil
}

func calculateEffectiveWorkRecursively(c *Component) {
	//initialize effective work with own work
	c.EffectiveWork = c.Work
	if len(c.Successors) > 0 {
		asyncSuccessorWork := make([]int, 0)
		for _, successor := range c.Successors {
			calculateEffectiveWorkRecursively(successor)
			//if we have successors, their work is added if it is sync, otherwise add to list of async calls.
			if successor.CallType == SYNC {
				c.EffectiveWork += successor.EffectiveWork
			} else {
				asyncSuccessorWork = append(asyncSuccessorWork, successor.EffectiveWork)
			}
		}
		//if the highest async effective work call is longer than the combined effective work of all sync calls with own work, we need to adapt the effective work acccordingly
		sort.Ints(asyncSuccessorWork)
		if c.EffectiveWork < asyncSuccessorWork[len(asyncSuccessorWork)-1] {
			c.EffectiveWork = asyncSuccessorWork[len(asyncSuccessorWork)-1]
		}
	}
}

//AddComponentsToSlice is a helper method which recursively adds all components to a slice.
func AddComponentsToSlice(list []*Component, c *Component) []*Component {
	list = append(list, c)
	log.Printf("Current last component: %v", list[len(list)-1])
	if len(c.Successors) > 0 {
		for _, s := range c.Successors {
			list = AddComponentsToSlice(list, s)
		}
	}
	return list
}

//AddComponentsToEnvMap is a helper function which recursively traverses the components and adds them to a map grouped by deploymentKeys of the components. The deploymentKey is an identifier for a deployment environment where multiple Components might be co-located.
func (c *Component) AddComponentsToEnvMap(envMap map[string][]*Component) map[string][]*Component {
	if _, exists := envMap[c.DeploymentKey]; !exists {
		envMap[c.DeploymentKey] = make([]*Component, 0)
	}
	envMap[c.DeploymentKey] = append(envMap[c.DeploymentKey], c)
	if len(c.Successors) > 0 {
		for _, s := range c.Successors {
			s.AddComponentsToEnvMap(envMap)
		}
	}
	return envMap
}

//TODO: so far no tags are written oder in the model - implement a simple generator for a number of tags with randomized key/value?
func (c *Component) ToSpanModel() *api.SpanModel {
	return &api.SpanModel{
		Delay:         int64(c.EffectiveWork),
		OperationName: c.Identifier,
	}
}
