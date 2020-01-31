package benchmark

import (
	"errors"
	"log"
	"os"
	"strings"

	"gitlab.tubit.tu-berlin.de/dominik-ernst/tracer-benchmarks/api"
	"gopkg.in/yaml.v2"
)

//Calltype enum alias.
type relationshipType int

//This constitutes the relationshipType enum - in golang, an integer with two static representations.
const (
	CHILD relationshipType = iota
	FOLLOWS
)

//The stringified values of the enum
var relationshipTypeNames = [...]string{
	"C",
	"F",
}

//String returns the static string-values of calltype.
func (r relationshipType) String() string {
	return relationshipTypeNames[r]
}

//UnmarshalYAML implements custom unmarshalling for the calltype in YAML files in order to use SYNC/ASYNC instead of 0 and 1.
func (r *relationshipType) UnmarshalYAML(unmarshal func(value interface{}) error) error {
	var stringValue string
	err := unmarshal(&stringValue)
	if err != nil {
		return err
	}
	stringValue = strings.ToLower(stringValue)
	//log.Printf("Parsed string value in YAML: %s\n", stringValue)
	if strings.Compare(stringValue, "c") == 0 {
		*r = CHILD
	} else if strings.Compare(stringValue, "f") == 0 {
		*r = FOLLOWS
	} else {
		return errors.New("couldnt parse relationship type, unknown value. must be either 'C' or 'F'")
	}
	return nil
}

//Model describes a set of services (which form a dependency tree), a set of sinks (which are endpoints to which services send traces), and a set of environments
//(logical references to deployment environments, which are used by services and sinks to learn about collocation)
type Model struct {
	Name         string      `yaml:"name"`
	Services     []*Service  `yaml:"services,flow"`
	Sinks        []*Sink     `yaml:"sinks,flow"`
	WorkUnits    []*WorkUnit `yaml:"workUnits,flow"`
	Environments []string    `yaml:"-"`
}

//Service models a traced service, as it would be deployed into an environment.
type Service struct {
	//Identifier is an arbitrary, unique identifier of the modeled service.
	Identifier string `yaml:"id"`
	//EnvironmentRef is a reference to an environment, to which services are deployed. If services (or sinks) share the same reference, they would be deployed into the same environment.
	EnvironmentRef string `yaml:"envRef"`
	//SinkRef is a reference to a sink, i.e. an endpoint to which the worker representing the service sends its traces.
	SinkRef string `yaml:"sinkRef"`
	//Tags are key value pairs of arbitrary string values, which are later written with each trace. For simplicity, currently only string-values are allowed, extrapolate other data-types accordingly.
	Context *SpanContext `yaml:"context"`
	//Units are wrappers around calls to other services
	Units        []*Unit   `yaml:"units,flow"`
	IsRoot       bool      `yaml:"-"`
	FinalWorkRef string    `yaml:"finalWork"`
	FinalWork    *WorkUnit `yaml:"-"`
}

//Unit is a wrapper around some work to be done and a call to another service.
type Unit struct {
	Rel api.RelationshipType `yaml:"rel"`
	//Local work to be done before a call to the successor is done. String to match defined Work types.
	WorkRef  string    `yaml:"work"`
	WorkUnit *WorkUnit `yaml:"-"`
	//Reference to the called service. String to match defined services.
	SuccessorRef string   `yaml:"svc"`
	Successor    *Service `yaml:"-"`
}

//WorkUnit represents the local work to be emulated by a service before the call to a successor is done.
type WorkUnit struct {
	Identifier string             `yaml:"id"`
	Type       string             `yaml:"type"`
	Params     map[string]float64 `yaml:"params"`
}

//SpanContext contains Tags and Baggage; Tags are local context of services, sent to the tracing backend
//Baggage is propagated to subsequent services (cf. OpenTracing specification https://github.com/opentracing/specification/blob/master/specification.md)
type SpanContext struct {
	Tags    map[LengthOrValue]LengthOrValue `yaml:"tags"`
	Baggage map[LengthOrValue]LengthOrValue `yaml:"baggage"`
}

//LengthOrValue is a helper type to represent either static or dynamically generated keys or values of tags and baggage items.
type LengthOrValue struct {
	Length   int64  `yaml:"-"`
	Value    string `yaml:"-"`
	IsLength bool   `yaml:"-"`
}

//UnmarshalYAML is the custom marshalling for LengthOrValue
func (l LengthOrValue) UnmarshalYAML(unmarshal func(value interface{}) error) error {
	//first try to marshal to int
	var lengthValue int
	err := unmarshal(&lengthValue)
	if err != nil {
		if errors.Is(err, &yaml.TypeError{}) {
			var stringValue string
			err = unmarshal(&stringValue)
			if err != nil {
				return err
			}
			l.Value = stringValue
			return nil
		}
		return err
	}
	l.IsLength = true
	l.Length = int64(lengthValue)
	return nil
}

//Sink is a wrapper around a backend service of a tracing system (something like a proxy/agent, storage/collector, stream pipeline or w/e).
type Sink struct {
	Identifier     string `yaml:"id"`
	Provider       string `yaml:"provider"`
	Address        string `yaml:"address"`
	EnvironmentRef string `yaml:"envRef"`
}

//ParseDeploymentDescription parses a YAML file containing a deployment description. Returns the deployment or respective parsing errors, if the file was invalid.
func ParseDeploymentDescription(yamlFile string) (*Model, error) {
	deployment, err := readFromYamlFile(yamlFile)
	if err != nil {
		log.Printf("Could not parse service description, error was: %v", err)
		return nil, err
	}
	validateDeploymentAndResolveRefs(deployment)
	return deployment, nil
}

func readFromYamlFile(file string) (*Model, error) {
	fileHandle, err := os.Open(file)
	if err != nil {
		return nil, err
	}

	decoder := yaml.NewDecoder(fileHandle)
	var deployment Model

	err = decoder.Decode(&deployment)
	if err != nil {
		return nil, err
	}

	return &deployment, nil
}

func validateDeploymentAndResolveRefs(deployment *Model) {
	//collect all envRefs in this
	envMap := make(map[string]int)
	//create map of serviceId -> service for quick lookup
	serviceIDMap := make(map[string]*Service)
	for i, c := range deployment.Services {
		serviceIDMap[c.Identifier] = c
		if val, exists := envMap[c.EnvironmentRef]; exists {
			envMap[c.EnvironmentRef] = val + 1
		} else {
			envMap[c.EnvironmentRef] = 1
		}
		//TODO: right now we make the first service the root service; should traverse all services and mark those without predecessor as root.
		if i == 0 {
			c.IsRoot = true
		}
	}
	sinkIDMap := make(map[string]*Sink)
	for _, s := range deployment.Sinks {
		sinkIDMap[s.Identifier] = s
		if val, exists := envMap[s.EnvironmentRef]; exists {
			envMap[s.EnvironmentRef] = val + 1
		} else {
			envMap[s.EnvironmentRef] = 1
		}
	}
	workUnitIDMap := make(map[string]*WorkUnit)
	for _, w := range deployment.WorkUnits {
		workUnitIDMap[w.Identifier] = w
	}

	envRefs := make([]string, 0)
	for key := range envMap {
		envRefs = append(envRefs, key)
	}
	deployment.Environments = envRefs

	for _, c := range deployment.Services {
		for _, unit := range c.Units {
			if unit.SuccessorRef != "" {
				referencedService, exists := serviceIDMap[unit.SuccessorRef]
				if !exists {
					log.Fatalf("Reference to non-existing successor id (%s) found in deployment: error in service %s. Aborting.", unit.SuccessorRef, c.Identifier)
				}
				unit.Successor = referencedService
			}
			if unit.WorkRef != "" {
				referencedWork, exists := workUnitIDMap[unit.WorkRef]
				if !exists {
					log.Fatalf("Reference to non-existing work id (%s) found in deployment: error in service %s. Aborting.", unit.WorkRef, c.Identifier)
				}
				unit.WorkUnit = referencedWork
			}
		}
		if c.FinalWorkRef != "" {
			c.FinalWork = workUnitIDMap[c.FinalWorkRef]
		}
	}
	//TODO currently, we assume there is exactly one root service, but the deployment would allow otherwise - what should we assume?
	// (multiple roots would require multiple target throughputs and lead to possible contention between requests from different roots)...
	//TODO check for loops in the service graph here
	//TODO check for validity of refs to envs/sinks
}

//AddServicesToEnvMap is a helper function which recursively traverses the services and adds them to a map grouped by Environments of the services. The EnvRef is an identifier for a deployment environment where multiple Services might be co-located.
func (m *Model) AddServicesToEnvMap() map[string][]*Service {
	envMap := make(map[string][]*Service)
	for _, c := range m.Services {
		if _, exists := envMap[c.EnvironmentRef]; !exists {
			envMap[c.EnvironmentRef] = make([]*Service, 0)
		}
		envMap[c.EnvironmentRef] = append(envMap[c.EnvironmentRef], c)
	}
	return envMap
}
