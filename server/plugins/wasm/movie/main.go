//go:build wasip1

package main

import "github.com/extism/go-pdk"

//go:wasmexport info
func exportInfo() int32 {
	if err := pdk.OutputJSON(describe()); err != nil {
		pdk.SetError(err)
		return 1
	}
	return 0
}

//go:wasmexport format
func exportFormat() int32 {
	var in struct{ Work entry }
	err := pdk.InputJSON(&in)
	if err == nil {
		err = pdk.OutputJSON(format(in.Work))
	}
	if err != nil {
		pdk.SetError(err)
		return 1
	}
	return 0
}

func main() {}
