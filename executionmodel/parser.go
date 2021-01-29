package executionmodel

import (
	"log"
	"os"

	"gopkg.in/yaml.v3"
)

//ParseArchitectureDescription parses a YAML file containing a architecture description. Returns the architecture or respective parsing errors, if the file was invalid.
func ParseArchitectureDescription(yamlFile string) (*Architecture, error) {
	architecture, err := readFromYamlFile(yamlFile)
	if err != nil {
		log.Printf("Could not parse sequence description, error was: %v", err)
		return nil, err
	}
	validateArchitectureAndResolveRefs(architecture)
	return architecture, nil
}

func readFromYamlFile(file string) (*Architecture, error) {
	fileHandle, err := os.Open(file)
	if err != nil {
		return nil, err
	}

	decoder := yaml.NewDecoder(fileHandle)
	var architecture Architecture

	err = decoder.Decode(&architecture)
	if err != nil {
		return nil, err
	}

	return &architecture, nil
}

func validateArchitectureAndResolveRefs(architecture *Architecture) {
	//collect all envRefs in this
	envMap := make(map[string]int)
	//create map of services for quick lookup
	serviceIDMap := make(map[string]*Service)
	for _, c := range architecture.Services {
		serviceIDMap[c.Identifier] = c
		if val, exists := envMap[c.EnvironmentRef]; exists {
			envMap[c.EnvironmentRef] = val + 1
		} else {
			envMap[c.EnvironmentRef] = 1
		}
		//TODO: do we need root and other hierarchical information at the service level?
		//c.IsRoot = true
		//c.Predecessors = make(map[string]*Sequence)
	}
	sinkIDMap := make(map[string]*Sink)
	for _, s := range architecture.Sinks {
		sinkIDMap[s.Identifier] = s
		if val, exists := envMap[s.EnvironmentRef]; exists {
			envMap[s.EnvironmentRef] = val + 1
		} else {
			envMap[s.EnvironmentRef] = 1
		}
	}
	workUnitIDMap := make(map[string]*Work)
	for _, w := range architecture.WorkTemplates {
		workUnitIDMap[w.Identifier] = w
	}

	envRefs := make([]string, 0)
	for key := range envMap {
		envRefs = append(envRefs, key)
	}
	architecture.Environments = envRefs

	//First, parse all units across all services and add them to a pool of (globally unique) units
	//currently, no uniquiness-check is done, maybe units should be prepended with the service name...?
	allUnitsMap := make(map[string]*Unit)
	for _, s := range architecture.Services {
		for _, unit := range s.Units {
			allUnitsMap[s.Identifier+"-"+unit.Identifier] = unit
		}
	}

	//for all services, parse unit references and replace with units from pool
	for _, s := range architecture.Services {
		for _, unit := range s.Units {
			//for _, input := range unit.InputRefs {
			//	if input.Service == "" {
			//		referencedUnit, exists := allUnitsMap[input]
			//		if !exists {
			//			log.Fatalf("Reference to non-existing input id (%s) found in architecture: error in unit with ID %s. Aborting.", input, unit.Identifier)
			//		}
			//		unit.Inputs[input] = referencedUnit
			//	}
			//}
			//for _, sucessor := range unit.SuccessorRefs {
			//	if sucessor.Ref != "" {
			//		referencedUnit, exists := allUnitsMap[sucessor.Ref]
			//		if !exists {
			//			log.Fatalf("Reference to non-existing successor id (%s) found in architecture: error in unit with ID %s. Aborting.", sucessor.Ref, unit.Identifier)
			//		}
			//		sucessor.Successor = referencedUnit
			//	}
			//}
			if unit.WorkRef != "" {
				referencedWork, exists := workUnitIDMap[unit.WorkRef]
				if !exists {
					log.Fatalf("Reference to non-existing work id (%s) found in architecture: error in sequence %s. Aborting.", unit.WorkRef, s.Identifier)
				}
				unit.WorkTemplate = referencedWork
			}
		}
	}
	for _, unit := range allUnitsMap {
		if len(unit.InputRefs) < 1 {
			unit.IsRoot = true
		}
	}
	// (multiple roots could use multiple target throughputs and lead to possible contention between requests from different roots)...
	//TODO check for loops in the unit graph here?
}

//AddServicesToEnvMap is a helper function which recursively traverses services and adds them to a map grouped by Environments assigned to each of them. The EnvRef is an identifier for a deployment environment where multiple services might be co-located.
func (m *Architecture) AddServicesToEnvMap() map[string][]*Service {
	envMap := make(map[string][]*Service)
	for _, c := range m.Services {
		if _, exists := envMap[c.EnvironmentRef]; !exists {
			envMap[c.EnvironmentRef] = make([]*Service, 0)
		}
		envMap[c.EnvironmentRef] = append(envMap[c.EnvironmentRef], c)
	}
	return envMap
}
