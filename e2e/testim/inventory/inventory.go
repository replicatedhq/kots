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

func NewNightlyTest() Test {
	return Test{
		Name:        "Nightly",
		Label:       "type=existing cluster, env=online, phase=new install, rbac=minimal rbac",
		Namespace:   "qakotsregression",
		UpstreamURI: "qakotsregression/type-existing-cluster-env-on-2",
	}
}
