package inventory

func NewChangeLicense() Test {
	return Test{
		Name:        "Change License",
		Suite:       "change-license",
		Namespace:   "change-license",
		UpstreamURI: "change-license/automated",
	}
}

func NewSmokeTest() Test {
	return Test{
		Name:           "Smoke Test",
		Suite:          "smoke-test",
		Namespace:      "smoke-test",
		UpstreamURI:    "qakotstestim/github-actions-qa",
		NeedsSnapshots: true,
	}
}

func NewRegressionTest() Test {
	return Test{
		Name:            "Regression",
		Label:           "type=existing cluster, env=online, phase=new install, rbac=minimal rbac",
		Namespace:       "qakotsregression",
		UpstreamURI:     "qakotsregression/type-existing-cluster-env-on-2",
		UseMinimalRBAC:  true,
		NeedsMonitoring: true,
		NeedsRegistry:   true,
	}
}

func NewStrictPreflightChecks() Test {
	return Test{
		Name:        "Strict Preflight Checks",
		Suite:       "strict-preflight-checks",
		Namespace:   "strict-preflight-checks",
		UpstreamURI: "strict-preflight-checks/automated",
	}
}

func NewMinimalRBACTest() Test {
	return Test{
		Name:        "Minimal RBAC App",
		Suite:       "minimal-rbac",
		Namespace:   "minimal-rbac",
		UpstreamURI: "minimal-rbac/automated",
	}
}

func NewMinimalRBACOverrideTest() Test {
	return Test{
		Name:           "Minimal RBAC Override",
		Suite:          "minimal-rbac",
		Namespace:      "minimal-rbac",
		UpstreamURI:    "minimal-rbac/automated",
		UseMinimalRBAC: true,
	}
}
