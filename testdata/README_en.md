# Naming of tests
Each test case is a self-contained project/directory, located at `testdata/{language}/index_{name}`.
The `index` is a 0-based number, which `go test` uses to determine the test execution order.

Note that `go test` occasionally only runs test cases prefixed with `0_xx`.
