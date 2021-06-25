package aws_test

import (
	"testing"

	"git.fuzzbuzz.io/fuzz"
	fuzztest "git.fuzzbuzz.io/fuzz/testing"
	"github.com/infracost/infracost/internal/resources/aws"
)

func FuzzNewNATGateway(f *fuzz.F) {
	args := &aws.NATGatewayArguments{}

	f.Struct("args", &aws.NATGatewayArguments{}).Populate(args)

	aws.NewNATGateway(args)
}

func TestFuzzNewNATGateway(t *testing.T) {
	f := fuzztest.NewChecker(t)

	fuzztest.Check(f, FuzzNewNATGateway)

	fuzztest.Randomize(f, FuzzNewNATGateway, 1000)
}
