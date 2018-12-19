package benchmark

import (
	"errors"
	"log"
	"math/rand"
	"os"
	"reflect"
	"strings"

	api "gitlab.tubit.tu-berlin.de/dominik-ernst/trace-writer-api"
	"gopkg.in/yaml.v2"
)

//Calltype enum alias.
type relationshipType int

//Work represents the amount of processing time in nanoseconds.
type Work interface {
	//Sample the next processing time.
	NextVal() int64
}

var workTypeRegistry = make(map[string]reflect.Type)

type ConstantWork struct {
	Value float64 `yaml:"value"`
}

func (w *ConstantWork) NextVal() int64 {
	return int64(w.Value)
}

type GaussianWork struct {
	Mean   float64 `yaml:"mean"`
	StdDev float64 `yaml:"stddev"`
}

func (w *GaussianWork) NextVal() int64 {
	return int64(rand.NormFloat64()*w.StdDev + w.Mean)
}

//This basically constitutes the enum. Calltype is an integer with two static representations.
const (
	CHILD relationshipType = iota
	FOLLOWS
)

//The stringified values of the enum
var relationshipTypeNames = [...]string{
	"C",
	"F",
}

func init() {
	workTypeRegistry["constant"] = reflect.TypeOf(ConstantWork{})
	workTypeRegistry["gaussian"] = reflect.TypeOf(GaussianWork{})
}

//String returns the static string-values of calltype.
func (r relationshipType) String() string {
	return relationshipTypeNames[r]
}

//UnmarshalYAML implements custom unmarshalling for the calltype in YAML files in order to use SYNC/ASYNC instead of 0 and 1.
func (c *relationshipType) UnmarshalYAML(unmarshal func(value interface{}) error) error {
	var stringValue string
	err := unmarshal(&stringValue)
	if err != nil {
		return err
	}
	stringValue = strings.ToLower(stringValue)
	//log.Printf("Parsed string value in YAML: %s\n", stringValue)
	if strings.Compare(stringValue, "c") == 0 {
		*c = CHILD
	} else if strings.Compare(stringValue, "f") == 0 {
		*c = FOLLOWS
	} else {
		return errors.New("couldnt parse relationship type, unknown value. must be either 'C' or 'F'")
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
	WorkUnits    []*WorkUnit    `yaml:"workUnits,flow"`
}

//Component models a traced component, as it would be deployed into an environment.
type Component struct {
	//Identifier is an arbitrary, unique identifier of the modeled component.
	Identifier string `yaml:"id"`
	//EnvironmentRef is a reference to an environment, to which components are deployed. If two components share the same reference, they would be deployed on the same VM or in the same container.
	EnvironmentRef string `yaml:"envRef"`
	//SinkRef is a reference to a sink, i.e. an endpoint to which the worker representing the component sends its traces.
	SinkRef string `yaml:"sinkRef"`
	//Tags are key value pairs of arbitrary string values, which are later written with each trace. For simplicity, currently only string-values are allowed, extrapolate other data-types accordingly.
	Context SpanContext `yaml:"context"`
	//Units are wrappers around calls to other components
	Units []*Unit `yaml:"units,flow"`
}

//Unit is a wrapper around some work to be done and a call to another component.
type Unit struct {
	Rel relationshipType `yaml:"rel"`
	//Local work to be done before a call to the successor is done. String to match defined Work types.
	WorkRef   string `yaml:"work"`
	WorkClass Work   `yaml:"-"`
	//Reference to the called component. String to match defined components.
	SuccessorRef string     `yaml:"component"`
	Successor    *Component `yaml:"-"`
}

type WorkUnit struct {
	Identifier string             `yaml:"id"`
	Type       string             `yaml:"type"`
	Params     map[string]float64 `yaml:"params"`
}

//Environment is a logical reference to a deployment environment. It has an ID, which is references by sinks and components, and a provider, which is used to infer the address.
type Environment struct {
	Identifier string `yaml:"id"`
	Provider   string `yaml:"provider"`
}

//SpanContext is additional data appended to a SpanContext (cf. OpenTracing specification https://github.com/opentracing/specification/blob/master/specification.md)
type SpanContext struct {
	Tags map[string]int `yaml:"tags"`
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
	validateDeploymentAndResolveRefs(deployment)
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

func validateDeploymentAndResolveRefs(deployment *Deployment) {
	//create map of componentId -> component for quick lookup
	componentIDMap := make(map[string]*Component)
	for _, c := range deployment.Components {
		componentIDMap[c.Identifier] = c
	}

	envIDMap := make(map[string]*Environment)
	for _, e := range deployment.Environments {
		envIDMap[e.Identifier] = e
	}

	sinkIDMap := make(map[string]*Sink)
	for _, s := range deployment.Sinks {
		sinkIDMap[s.Identifier] = s
	}
	workUnitIDMap := make(map[string]*WorkUnit)
	for _, w := range deployment.WorkUnits {
		workUnitIDMap[w.Identifier] = w
	}

	for _, c := range deployment.Components {
		for _, unit := range c.Units {
			if unit.SuccessorRef != "" {
				referencedComponent, exists := componentIDMap[unit.SuccessorRef]
				if !exists {
					log.Fatalf("Reference to non-existant successor id (%s) found in deployment: error in component %s. Aborting.", unit.SuccessorRef, c.Identifier)
				}
				unit.Successor = referencedComponent
			}
			if unit.WorkRef != "" {
				referencedWork, exists := workUnitIDMap[unit.WorkRef]
				if !exists {
					log.Fatalf("Reference to non-existant work id (%s) found in deployment: error in component %s. Aborting.", unit.WorkRef, c.Identifier)
				}
				unit.WorkClass = parseWorkUnitToWork(referencedWork)
			}
		}
	}
	//TODO currently, we assume there is exactly one root component, but the deployment would allow otherwise - what should we assume?
	// multiple roots would require multiple target throughputs and lead to possible contention betweend requests from different roots...
	//TODO check for loops in the component graph here
	//TODO check for validity of refs to envs/sinks
}

//AddComponentsToEnvMap is a helper function which recursively traverses the components and adds them to a map grouped by Environments of the components. The EnvRef is an identifier for a deployment environment where multiple Components might be co-located.
func (d *Deployment) AddComponentsToEnvMap() map[string][]*Component {
	envMap := make(map[string][]*Component)
	for _, c := range d.Components {
		if _, exists := envMap[c.EnvironmentRef]; !exists {
			envMap[c.EnvironmentRef] = make([]*Component, 0)
		}
		envMap[c.EnvironmentRef] = append(envMap[c.EnvironmentRef], c)
	}
	return envMap
}

func parseWorkUnitToWork(wu *WorkUnit) Work {
	v := &ConstantWork{
		Value: 100,
	}
	workType, exists := workTypeRegistry[wu.Type]
	if exists {
		if workType == workTypeRegistry["constant"] {
			v.Value = wu.Params["value"]
		}
		//TODO other work types are not yet functional
		/* v := reflect.New(workType)
		for key, val := range wu.Params {
			workType := v.Elem()
			field := workType.FieldByName(strings.ToTitle(key))
			if field.CanSet() {
				field.SetFloat(val)
			} else {
				log.Println("Couldn't write!!!")
			}
		} */
	} else {
		log.Printf("Referenced work type not found: %s\n, using constant work with value = 100", wu.Type)
	}
	return v
}

//TODO: so far no tags are written to the model - implement a simple generator for a number of tags with randomized key/value?
func (c *Component) ToSpanModel() *api.SpanModel {
	return &api.SpanModel{
		Delay:         int64(c.Units[0].WorkClass.NextVal()),
		OperationName: c.Identifier,
	}
}
