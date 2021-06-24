package examples

import "git.fuzzbuzz.io/fuzz"

func FuzzNOP(f *fuzz.F) {
	f.Bytes("abc").Get()
}
