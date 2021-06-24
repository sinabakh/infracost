package google

import "git.fuzzbuzz.io/fuzz"

func FuzzGetDSRegionResourceGroup(f *fuzz.F) {
	location := f.String("location").Get()
	storageClass := f.String("storageClass").Get()
	getDSRegionResourceGroup(location, storageClass)
}
