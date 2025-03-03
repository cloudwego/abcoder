module github.com/cloudwego/abcoder/src/lang

go 1.22.0

toolchain go1.24.0

require (
	github.com/cloudwego/abcoder/src/uniast v0.0.0
	github.com/sourcegraph/go-lsp v0.0.0-20240223163137-f80c5dd31dfd
	github.com/sourcegraph/jsonrpc2 v0.2.0
	github.com/spf13/cobra v1.8.1
)

require (
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
)

replace github.com/cloudwego/abcoder/src/uniast => ../uniast
