package libyaml

import (
	"fmt"
	"reflect"
	"strings"

	validator "gopkg.in/go-playground/validator.v8"
)

var (
	SwarmNodeRoleManager SwarmNodeRole = "manager"
	SwarmNodeRoleWorker  SwarmNodeRole = "worker"

	SwarmNodeRoles = []SwarmNodeRole{
		SwarmNodeRoleManager,
		SwarmNodeRoleWorker,
	}
)

type SwarmNodeRole string

type Swarm struct {
	MinNodeCount string        `yaml:"minimum_node_count,omitempty" json:"minimum_node_count,omitempty" validate:"omitempty,number"` // uint
	Nodes        []SwarmNode   `yaml:"nodes,omitempty" json:"nodes,omitempty" validate:"dive"`
	Secrets      []SwarmSecret `yaml:"secrets,omitempty" json:"secrets,omitempty" validate:"dive"`
	Configs      []SwarmConfig `yaml:"configs,omitempty" json:"configs,omitempty" validate:"dive"`
}

type SwarmNode struct {
	Role             SwarmNodeRole     `yaml:"role,omitempty" json:"role,omitempty" validate:"omitempty,swarmnoderole"`
	Labels           map[string]string `yaml:"labels,omitempty" json:"labels,omitempty"`
	MinCount         string            `yaml:"minimum_count,omitempty" json:"minimum_count,omitempty" validate:"omitempty,number"` // uint
	HostRequirements HostRequirements  `yaml:"host_requirements,omitempty" json:"host_requirements,omitempty"`
}

type SwarmSecret struct {
	Name   string            `yaml:"name" json:"name" validate:"required"`
	Value  string            `yaml:"value" json:"value" validate:"required"`
	Labels map[string]string `yaml:"labels,omitempty" json:"labels,omitempty" validate:"mapkeylengthnonzero"`
}

type SwarmConfig struct {
	Name   string            `yaml:"name" json:"name" validate:"required"`
	Value  string            `yaml:"value" json:"value" validate:"required"`
	Labels map[string]string `yaml:"labels,omitempty" json:"labels,omitempty" validate:"mapkeylengthnonzero"`
}

func init() {
	RegisterValidation("swarmnoderole", ValidationSwarmNodeRole, ValidationErrorSwarmNodeRole)
}

func ValidationErrorSwarmNodeRole(formatted string, key string, fieldErr *validator.FieldError, root *RootConfig) error {
	var roles []string
	for _, role := range SwarmNodeRoles {
		roles = append(roles, fmt.Sprintf("'%s'", role))
	}
	return fmt.Errorf("Swarm node role must be one of %s at key %q", strings.Join([]string(roles), ", "), formatted)
}

func ValidationSwarmNodeRole(v *validator.Validate, topStruct reflect.Value, currentStructOrField reflect.Value, field reflect.Value, fieldType reflect.Type, fieldKind reflect.Kind, param string) bool {
	if fieldKind != reflect.String {
		return true
	}

	valueStr := field.String()
	for _, role := range SwarmNodeRoles {
		if valueStr == string(role) {
			return true
		}
	}
	return false
}
