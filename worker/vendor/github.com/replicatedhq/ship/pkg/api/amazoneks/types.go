package amazoneks

type EKSCreatedVPC struct {
	Zones          []string `json:"zones,omitempty" yaml:"zones,omitempty" hcl:"zones,omitempty"`
	VPCCIDR        string   `json:"vpc_cidr,omitempty" yaml:"vpc_cidr,omitempty" hcl:"vpc_cidr,omitempty"`
	PublicSubnets  []string `json:"public_subnets,omitempty" yaml:"public_subnets,omitempty" hcl:"public_subnets,omitempty"`
	PrivateSubnets []string `json:"private_subnets,omitempty" yaml:"private_subnets,omitempty" hcl:"private_subnets,omitempty"`
}

type EKSExistingVPC struct {
	VPCID          string   `json:"vpc_id,omitempty" yaml:"vpc_id,omitempty" hcl:"vpc_id,omitempty"`
	PublicSubnets  []string `json:"public_subnets,omitempty" yaml:"public_subnets,omitempty" hcl:"public_subnets,omitempty"`
	PrivateSubnets []string `json:"private_subnets,omitempty" yaml:"private_subnets,omitempty" hcl:"private_subnets,omitempty"`
}

type EKSAutoscalingGroup struct {
	Name        string `json:"name,omitempty" yaml:"name,omitempty" hcl:"name,omitempty"`
	GroupSize   string `json:"group_size,omitempty" yaml:"group_size,omitempty" hcl:"group_size,omitempty"`
	MachineType string `json:"machine_type,omitempty" yaml:"machine_type,omitempty" hcl:"machine_type,omitempty"`
}
