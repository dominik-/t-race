package api

import (
	"strings"
)

func (r *RelationshipType) UnmarshalYAML(unmarshal func(value interface{}) error) error {
	var stringValue string
	err := unmarshal(&stringValue)
	if err != nil {
		return err
	}
	stringValue = strings.ToLower(stringValue)
	//log.Printf("Parsed string value in YAML: %s\n", stringValue)
	switch stringValue {
	case "child":
		*r = RelationshipType_CHILD
		break
	case "follows":
		*r = RelationshipType_FOLLOWS
		break
	case "client":
		*r = RelationshipType_CLIENT
		break
	case "server":
		*r = RelationshipType_SERVER
		break
	case "consumer":
		*r = RelationshipType_CONSUMER
		break
	case "producer":
		*r = RelationshipType_PRODUCER
		break
	default:
		*r = RelationshipType_INTERNAL
	}
	return nil
}
