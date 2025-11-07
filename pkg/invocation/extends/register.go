package extends

import (
	"github.com/genmcp/gen-mcp/pkg/invocation"
)

var registry map[string]*invocation.InvocationConfigWrapper

func init() {
	registry = make(map[string]*invocation.InvocationConfigWrapper)
}

func SetBases(bases map[string]*invocation.InvocationConfigWrapper) {
	registry = bases
}

func getBase(name string) (*invocation.InvocationConfigWrapper, bool) {
	base, exists := registry[name]

	return base, exists
}
