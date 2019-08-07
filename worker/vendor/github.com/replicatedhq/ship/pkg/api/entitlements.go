package api

import "time"

// types in this file are copied from
// the `titled` package, need to OSS the type defs
// or find another way to share them

// Meta describes metadata about an entitlements payload
type Meta struct {
	LastUpdated time.Time `json:"lastUpdated"`
	CustomerID  string    `json:"customerID"`
}

// EntitlementLabel is a single entitlement label
type EntitlementLabel struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// EntitlementValue is a single entitlement value
type EntitlementValue struct {
	Key    string             `json:"key"`
	Value  string             `json:"value"`
	Labels []EntitlementLabel `json:"labels"`
}

// Utilization is a single utilization value
type Utilization struct {
	Key   string `json:"key"`
	Value int64  `json:"value"`
}

// Entitlements is a signed object containing entitlments info+metadata
type Entitlements struct {
	Meta         Meta               `json:"meta"`
	Signature    string             `json:"signature"`
	Values       []EntitlementValue `json:"values,omitempty"`
	Utilizations []Utilization      `json:"utilizations,omitempty"`
}
