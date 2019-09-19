# Replicated App Integration Tests

## How this works
The replicated app integration test framework has a mock server that returns an archive. This was done to eliminate
any external dependencies on this test and increase reliability.

## Creating a new test
This kots integration tests have the ability to create new tests without requiring a lot of manual effort.

1. Create a directory that has the destired YAML. Let's say this is in ~/my-new-test
2. Run `./bin/kots-integration new-replicatedapp-fixture ~/my-new-test --name my-new-test`
