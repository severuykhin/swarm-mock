package swarm

import "github.com/valyala/fasthttp"

type Value struct {
	value    Tuple
	values   []Tuple
	response *fasthttp.Response
}

// Tuple returned in the Value.
func (v Value) Tuple() Tuple {
	return v.value
}

// Tuples returned in the Value.
func (v Value) Tuples() []Tuple {
	return v.values
}

// Release Value after use.
func (v *Value) Release() {
	if v.response != nil {
		fasthttp.ReleaseResponse(v.response)
		v.response = nil
	}
	v.value = nil
}
