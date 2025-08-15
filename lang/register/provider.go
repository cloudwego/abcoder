package register

import (
	javaLsp "github.com/cloudwego/abcoder/lang/java/lsp"
	"github.com/cloudwego/abcoder/lang/lsp"
	"github.com/cloudwego/abcoder/lang/uniast"
)

func RegisterProviders() {
	lsp.RegisterProvider(uniast.Java, &javaLsp.JavaProvider{})

}
