package paths

import (
	"fmt"
	"math/rand"
	"testing"
)

// moved computes the move distance of a pen (excluding draw distance).
func moved(ps *Paths) float64 {
	d := 0.0
	var last Vec2
	for _, p := range ps.P {
		d += vec2dist(last, p.V[0])
		last = p.V[len(p.V)-1]
	}
	return d
}

type testSortCase struct {
	desc        string
	paths       *Paths
	cfg         *SortConfig
	wantMaxMove float64
}

func testSortRandom() testSortCase {
	ps := &Paths{Bounds: Bounds{Min: Vec2{-1000, -1000}, Max: Vec2{1000, 1000}}}
	const N = 100
	for i := 0; i < N; i++ {
		randStart := Vec2{rand.Float64()*2000 - 1000, rand.Float64()*2000 - 1000}
		randEnd := Vec2{rand.Float64()*2000 - 1000, rand.Float64()*2000 - 1000}
		randLine := Path{V: []Vec2{randStart, randEnd}}
		ps.P = append(ps.P, randLine)
	}
	return testSortCase{
		desc:        fmt.Sprintf("%d random lines", N),
		paths:       ps,
		cfg:         &SortConfig{},
		wantMaxMove: 0.5,
	}
}

func TestSort(t *testing.T) {
	cases := []testSortCase{
		testSortRandom(),
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			mvd0 := moved(tc.paths)
			N := len(tc.paths.P)
			tc.paths.Sort(tc.cfg)
			mvd1 := moved(tc.paths)
			if !(mvd1 < tc.wantMaxMove*mvd0) {
				t.Errorf("got move distance %f, want at most %f", mvd1, mvd0*tc.wantMaxMove)
			}
			if len(tc.paths.P) != N {
				// In theory, we could end up with less paths than we started with if the
				// end-point of one matches the start-point of another. It's not very likely
				// though.
				t.Errorf("started with %d paths, ended with %d paths", N, len(tc.paths.P))
			}
		})
	}
}
