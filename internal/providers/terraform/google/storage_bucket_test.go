package google_test

import (
	"testing"

	fuzztest "git.fuzzbuzz.io/fuzz/testing"

	"github.com/infracost/infracost/internal/providers/terraform/google"
	"github.com/infracost/infracost/internal/providers/terraform/tftest"
)

func TestStorageBucket(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("skipping test in short mode")
	}

	tftest.GoldenFileResourceTests(t, "storage_bucket_test")
}

func TestFuzzGetDSRegionResourceGroup(t *testing.T) {
	f := fuzztest.NewChecker(t)

	fuzztest.Check(f, google.FuzzGetDSRegionResourceGroup)

	fuzztest.Randomize(f, google.FuzzGetDSRegionResourceGroup, 1000)
}
