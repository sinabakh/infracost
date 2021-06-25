package aws

import (
	"git.fuzzbuzz.io/fuzz"
)

func FuzzNewNATGateway(f *fuzz.F) {
	args := &NATGatewayArguments{}

	f.Struct("args", &NATGatewayArguments{}).Populate(args)

	NewNATGateway(args)
}
