module github.com/cloudwego/abcoder/src/compress/golang/plugin

go 1.22.0

require (
	github.com/davecgh/go-spew v1.1.1
	golang.org/x/tools v0.30.0
	github.com/cloudwego/abcoder/src/uniast v0.0.0
)

require (
	github.com/Knetic/govaluate v3.0.0+incompatible
	github.com/stretchr/testify v1.9.0
	golang.org/x/mod v0.23.0
)

require (
	github.com/pmezard/go-difflib v1.0.0 // indirect
	golang.org/x/sync v0.11.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace github.com/cloudwego/abcoder/src/uniast => ../../../uniast
