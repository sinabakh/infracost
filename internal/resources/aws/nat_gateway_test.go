package aws_test

import (
	"testing"

	fuzztest "git.fuzzbuzz.io/fuzz/testing"
	"github.com/infracost/infracost/internal/resources/aws"
)

func TestFuzzNewNATGateway(t *testing.T) {
	f := fuzztest.NewChecker(t)

	fuzztest.Check(f, aws.FuzzNewNATGateway)

	fuzztest.Randomize(f, aws.FuzzNewNATGateway, 1000)
}
