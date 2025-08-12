module a.b/c/cmdx

go 1.20

replace a.b/c => ../.

require (
	a.b/c v0.0.0-00010101000000-000000000000
	github.com/bytedance/sonic v1.11.3
	github.com/pkg/errors v0.9.1
)

require (
	github.com/chenzhuoyu/base64x v0.0.0-20230717121745-296ad89f973d // indirect
	github.com/chenzhuoyu/iasm v0.9.0 // indirect
	github.com/klauspost/cpuid/v2 v2.0.9 // indirect
	github.com/twitchyliquid64/golang-asm v0.15.1 // indirect
	golang.org/x/arch v0.0.0-20210923205945-b76863e36670 // indirect
)
