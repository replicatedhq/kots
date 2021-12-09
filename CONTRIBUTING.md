# Contributing to KOTS

Thank you for your interest in KOTS, we welcome your participation. Please familiarize yourself with our 
[Code of Conduct](https://github.com/replicatedhq/kots/blob/main/CODE_OF_CONDUCT.md) prior to contributing. 

## Pull Requests 

A pull request should address a single issue, feature or bug. For example, lets say you've written code that fixes two
issues. That's great! However, you should submit two small pull requests, one for each issue as opposed to combining them
into a single larger pull request.  In general the size of the pull request should be kept small in order to make it easy 
for a reviewer to understand, and to minimize risks from integrating many changes at the same time. For example, if you 
are working on a large feature you should break it into several smaller PRs by implementing the feature as changes to 
several packages and submitting a separate pull request for each one.  

Code submitted in pull requests must be properly documented, formatted and tested in order to be approved and merged. The following 
guidelines describe the things a reviewer will look for when they evaluate your pull request. Here's a tip. 
If your reviewer doesn't understand what the code is doing, they won't approve the pull request. Strive to make code 
clear and well documented. If possible, request a reviewer that has some context on the PR.

### Pull Request Guidelines 

### Testing

#### Unit Tests 
Unit tests verify the feature you implemented does what it's supposed to do.  In general that means 
testing public methods to ensure that your feature performs in accordance with the requirements it is expected to 
satisfy, including error cases. Avoid writing tests that evaluate the private internals of a package, these types of tests 
are brittle and discourage refactoring code.  The public interface of a package is less volatile and, if the unit 
tests fully exercise the contract that a package exposes will provide sufficient code coverage. 

If the code in a feature is using multiple goroutines, tests should pass with the `-race` flag enabled.  The concurrent 
code must be covered in tests.

If the feature under test needs to interact with external services such as a database
or other services, the interactions should be wrapped in an abstractions that simulate their functionality such that unit
tests run without depending on the presence of external resources. Provide integration tests to verify that the feature
interacts as expected with external services.

#### Integration Tests
If a feature interacts with an external service over a network for instance, provide integration tests that verify that 
the feature can successfully interact with external resources such as databases, RESTFUL APIS etc.  Integration tests
*must* be segregated from unit tests by including a build tag `// +build integration` on the first line of each integration 
test file.  Use environment variables to supply credentials needed to interact with external resources. Documentation 
must be provided for credentials that are needed to run integration tests. 

### Documentation 
All public declarations in the PR code should be documented [Godocs](https://go.dev/blog/godoc).  If the feature includes 
new packages, each new package should contain a file `doc.go` that describes what the package does and how to use it. 

### Formatting 
Run `gofmt` and `goimports` before submitting code. 

### Commit History
Prefer submitting a pull request that contains a single commit with a descriptive comment.  Avoid submitting pull requests
with several 'work in progress' type commits as it clutters the commit history. It's fine to use frequent commits as you 
work, but rebase before you submit your pull request.