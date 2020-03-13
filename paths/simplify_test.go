package paths

import (
	"reflect"
	"testing"
)

type simplifyTestCase struct {
	desc string
	path Path
	tol  float64
	want []Path
}

func TestSimplify(t *testing.T) {
	p := func(args ...float64) Path {
		if len(args)%2 != 0 {
			t.Fatalf("p helper needs an even number of args, got %v", args)
		}
		path := Path{}
		for i := 0; i < len(args); i += 2 {
			path.V = append(path.V, Vec2{args[i], args[i+1]})
		}
		return path
	}

	cases := []simplifyTestCase{
		{
			desc: "line with slightly displaced midpoint, high tolerance",
			path: p(-1, 0, 0, 0.25, 1.0, 0),
			tol:  0.5,
			want: []Path{p(-1, 0, 1, 0)},
		},
		{
			desc: "line with slightly displaced midpoint, low tolerance",
			path: p(-1, 0, 0, 0.5, 1.0, 0),
			tol:  0.2,
			want: []Path{p(-1, 0, 0, 0.5, 1.0, 0)},
		},
		{
			desc: "square with slightly displaced midpoints, high tolerance",
			path: p(-1, -1, 0, -1.1, 1, -1, 0.9, 0, 1, 1, 0, 1.1, -1, 1, -0.9, 0, -1, -1),
			tol:  0.2,
			want: []Path{p(-1, -1, 1, -1, 1, 1, -1, 1, -1, -1)},
		},
	}
	for _, c := range cases {
		arg := &Paths{
			Bounds: Bounds{Min: Vec2{-1000, -1000}, Max: Vec2{1000, 1000}},
			P:      []Path{c.path},
		}
		ps := &Paths{
			Bounds: arg.Bounds,
			P:      []Path{Path{V: append([]Vec2{}, c.path.V...)}},
		}
		ps.Simplify(c.tol)
		if !reflect.DeepEqual(ps.P, c.want) {
			t.Errorf("%v.Simplify(%v).P = %v, want %v", arg, c.tol, ps.P, c.want)
		}
	}
}
