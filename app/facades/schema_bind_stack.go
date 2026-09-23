package facades

import "runtime"

func runtimeStackImpl(buf []byte) int {
	return runtime.Stack(buf, false)
}
