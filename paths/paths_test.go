package paths

import (
	"reflect"
	"testing"
)

type transformTestCase struct {
	desc   string
	in     *Paths
	bounds Bounds
	want   *Paths
}

func (ps *Paths) copy() *Paths {
	psc := &Paths{}
	psc.Bounds = ps.Bounds
	for _, p := range ps.P {
		psc.P = append(psc.P, Path{V: append([]Vec2{}, p.V...)})
	}
	return psc
}

func TestTransform(t *testing.T) {
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
	b := func(mx, my, Mx, My float64) Bounds {
		return Bounds{Min: Vec2{mx, my}, Max: Vec2{Mx, My}}
	}
	cases := []transformTestCase{
		{
			desc: "translate 100,0",
			in: &Paths{
				Bounds: b(0, 0, 200, 100),
				P:      []Path{p(50, 20, 100, 40)},
			},
			bounds: b(100, 0, 300, 100),
			want: &Paths{
				Bounds: b(100, 0, 300, 100),
				P:      []Path{p(100+50, 20, 100+100, 40)},
			},
		},
		{
			desc: "scale 2,2",
			in: &Paths{
				Bounds: b(0, 0, 200, 100),
				P:      []Path{p(50, 20, 100, 40)},
			},
			bounds: b(0, 0, 400, 200),
			want: &Paths{
				Bounds: b(0, 0, 400, 200),
				P:      []Path{p(2*50, 2*20, 2*100, 2*40)},
			},
		},
	}
	for _, tc := range cases {
		got := tc.in.copy()
		got.Transform(tc.bounds)

		if !reflect.DeepEqual(tc.want, got) {
			t.Errorf("%v.Transform(%v) = %v, want %v", tc.in, tc.bounds, got, tc.want)
		}
	}
}
