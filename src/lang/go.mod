module github.com/cloudwego/abcoder/src/lang

go 1.22.0

toolchain go1.24.0

require (
	github.com/cloudwego/abcoder/src/compress/golang/plugin v0.0.0-20240905074027-8f815c26a391
	github.com/sourcegraph/go-lsp v0.0.0-20240223163137-f80c5dd31dfd
	github.com/sourcegraph/jsonrpc2 v0.2.0
	github.com/spf13/cobra v1.8.1
)

require (
	github.com/Knetic/govaluate v3.0.0+incompatible // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
	golang.org/x/mod v0.23.0 // indirect
	golang.org/x/sync v0.11.0 // indirect
	golang.org/x/tools v0.30.0 // indirect
)

replace github.com/cloudwego/abcoder/src/compress/golang/plugin => ../compress/golang/plugin
