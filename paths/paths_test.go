package paths

import (
	"reflect"
	"testing"
)

type clipTestCase struct {
	bounds Bounds
	path   Path
	want   []Path
}

func TestClip(t *testing.T) {
	b := func(x0, y0, x1, y1 float64) Bounds {
		return Bounds{Min: Vec2{x0, y0}, Max: Vec2{x1, y1}}
	}
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

	cases := []clipTestCase{
		{
			bounds: b(0, 0, 300, 200),
			path:   p(-100, 100, 150, 100),
			want:   []Path{p(0, 100, 150, 100)},
		},
		{
			bounds: b(0, 0, 300, 200),
			path:   p(-100, 100, 400, 100),
			want:   []Path{p(0, 100, 300, 100)},
		},
		{
			bounds: b(0, 0, 300, 200),
			path:   p(150, 100, 400, 100),
			want:   []Path{p(150, 100, 300, 100)},
		},
		{
			bounds: b(0, 0, 300, 200),
			path:   p(150, 100, 150, 250),
			want:   []Path{p(150, 100, 150, 200)},
		},
		{
			bounds: b(0, 0, 300, 200),
			path:   p(150, -50, 150, 100),
			want:   []Path{p(150, 0, 150, 100)},
		},
		{
			bounds: b(0, 0, 300, 200),
			path:   p(150, -50, 150, 250),
			want:   []Path{p(150, 0, 150, 200)},
		},
		{
			bounds: b(0, 0, 200, 100),
			path:   p(-50, 0, 100, 150, 250, 0),
			want:   []Path{p(0, 50, 50, 100), p(150, 100, 200, 50)},
		},
	}
	for _, c := range cases {
		arg := &Paths{
			Bounds: b(-1000, -1000, 1000, 1000),
			P:      []Path{c.path},
		}
		ps := &Paths{
			Bounds: arg.Bounds,
			P:      []Path{Path{V: append([]Vec2{}, c.path.V...)}},
		}
		ps.Clip(c.bounds)
		if !reflect.DeepEqual(ps.P, c.want) {
			t.Errorf("%v.Clip(%v).P = %v, want %v", arg, c.bounds, ps.P, c.want)
		}
	}

}
