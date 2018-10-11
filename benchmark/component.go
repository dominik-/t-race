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

//Deployment describes a set of components (which form a dependency tree), a set of sinks (which are endpoints to which components send traces), and a set of environments
//(logical references to deployment environments, which are used by components and sinks to learn about collocation)
type Deployment struct {
	Name         string         `yaml:"name"`
	Components   []*Component   `yaml:"components,flow"`
	Environments []*Environment `yaml:"environments,flow"`
	Sinks        []*Sink        `yaml:"sinks,flow"`
}

//Component models a traced component, as it would be deployed by a user.
type Component struct {
	//Identifier is an arbitrary, unique identifier of the modeled component.
	Identifier string `yaml:"id"`
	//EnvironmentRef is a reference to an environment, to which components are deployed. If two components share the same reference, they would be deployed on the same VM or in the same container.
	EnvironmentRef string `yaml:"envRef"`
	//SinkRef is a reference to a sink, i.e. an endpoint to which the worker representing the component sends its traces.
	SinkRef string `yaml:"sinkRef"`
	//Work represents the total amount of work done by this component, in nanoseconds. This should include possible interleaved work, that is done between calls to successors. Important for "callers" of this component, as they use this to emulate span duration.
	Work int `yaml:"work"`
	//CallType is the way THIS component is called, can be either SYNC or ASYNC. If it is ASYNC, that means the parent component does not wait until a response is returned, i.e. call to next component after an async call is treated as parallel.
	CallType calltype `yaml:"calltype"`
	//Successors are the components called by this component. For each successor, a span is generated. (For all successors with SYNC, their "Work" is used as delay before starting the span corresponding to the next component call TODO: do we need/want this?). Furthermore, SYNC causes the successors Work to be added to this components Work. All ASYNC spans are started in parallel.
	SuccessorRefs []string     `yaml:"successors,flow"`
	Successors    []*Component `yaml:"-"`
	//The effective work to be done by this component, which is the sum of its internal work, integrated with the effective work of all successors. This field is skipped during parsing (key '-').
	EffectiveWork int `yaml:"-"`
	//Tags are key value pairs of arbitrary string values, which are later written with each trace. For simplicity, currently only string-values are allowed, extrapolate other data-types accordingly.
	Tags map[string]string `yaml:"tags"`
}

//Environment is a logical reference to a deployment environment. It has an ID, which is references by sinks and components, and a provider, which is used to infer the address.
type Environment struct {
	Identifier string `yaml:"id"`
	Provider   string `yaml:"provider"`
}

//Sink is a wrapper around a backend component of a tracing system (something like a proxy/agent or storage/collector).
type Sink struct {
	Identifier     string `yaml:"id"`
	Provider       string `yaml:"provider"`
	Address        string `yaml:"address"`
	EnvironmentRef string `yaml:"envRef"`
}

//ParseDeploymentDescription parses a YAML file containing a deployment description. Returns the deployment or respective parsing errors, if the file was invalid.
func ParseDeploymentDescription(yamlFile string) (*Deployment, error) {
	deployment, err := readFromYamlFile(yamlFile)
	if err != nil {
		log.Printf("Could not parse component description, error was: %v", err)
		return nil, err
	}
	validateDeploymentAndCalculateWork(deployment)
	return deployment, nil
}

func readFromYamlFile(file string) (*Deployment, error) {
	fileHandle, err := os.Open(file)
	if err != nil {
		return nil, err
	}

	decoder := yaml.NewDecoder(fileHandle)
	var deployment Deployment

	err = decoder.Decode(&deployment)
	if err != nil {
		return nil, err
	}

	return &deployment, nil
}

func validateDeploymentAndCalculateWork(deployment *Deployment) {
	//TODO check for loops in the deployment here and parse dependencies
	//TODO effective work might be irrelevant, still calculate?
}

//TODO add parameter with all components to detect loops in the deployment description
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

//AddComponentsToEnvMap is a helper function which recursively traverses the components and adds them to a map grouped by Environments of the components. The EnvRef is an identifier for a deployment environment where multiple Components might be co-located.
func (c *Component) AddComponentsToEnvMap(envMap map[string][]*Component) map[string][]*Component {
	if _, exists := envMap[c.EnvironmentRef]; !exists {
		envMap[c.EnvironmentRef] = make([]*Component, 0)
	}
	envMap[c.EnvironmentRef] = append(envMap[c.EnvironmentRef], c)
	if len(c.Successors) > 0 {
		for _, s := range c.Successors {
			s.AddComponentsToEnvMap(envMap)
		}
	}
	return envMap
}

//TODO: so far no tags are written to the model - implement a simple generator for a number of tags with randomized key/value?
func (c *Component) ToSpanModel() *api.SpanModel {
	return &api.SpanModel{
		Delay:         int64(c.Work),
		OperationName: c.Identifier,
	}
}
