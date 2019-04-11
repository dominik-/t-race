package api

import (
	"errors"
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
	case "c":
		*r = RelationshipType_CHILD
		break
	case "f":
		*r = RelationshipType_FOLLOWING
		break
	default:
		return errors.New("couldnt parse relationship type, unknown value. must be either 'C' or 'F'")
	}
	return nil
}
