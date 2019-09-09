package pull

type ReplicatedPullTest struct {
	Name                 string
	LicenseData          string
	ReplicatedAppArchive string
	ExpectedFilesystem   string
}

func ReplicatedPullTests() []ReplicatedPullTest {
	tests := []ReplicatedPullTest{}

	// After generating a new test, add it here
	// tests = append(tests, MyNewTest)

	tests = append(tests, sentryEnterprise)

	return tests
}
