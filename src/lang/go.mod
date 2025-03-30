module github.com/cloudwego/abcoder/src/lang

go 1.23.0

toolchain go1.24.1

require (
	github.com/Knetic/govaluate v3.0.0+incompatible
	github.com/cloudwego/abcoder/src/uniast v0.0.0
	github.com/davecgh/go-spew v1.1.1
	github.com/sourcegraph/go-lsp v0.0.0-20240223163137-f80c5dd31dfd
	github.com/sourcegraph/jsonrpc2 v0.2.0
	github.com/spf13/cobra v1.8.1
	github.com/stretchr/testify v1.10.0
	golang.org/x/mod v0.24.0
	golang.org/x/tools v0.31.0
)

require (
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
	golang.org/x/sync v0.12.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace github.com/cloudwego/abcoder/src/uniast => ../uniast
