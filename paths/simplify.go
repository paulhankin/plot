package paths

import (
	"math"
)

func vec2linedist(v, s, e Vec2) float64 {
	ds := vec2dist(v, s)
	de := vec2dist(v, e)
	n := Vec2{e[1] - s[1], s[0] - e[0]}
	dp := v[0]*n[0] + v[1]*n[1]
	return math.Min(math.Min(math.Abs(dp), ds), de)
}

func simplifyPath(v []Vec2, tol float64) []Vec2 {
	worst := 0
	worstD := 0.0
	for i := 1; i < len(v)-1; i++ {
		d := vec2linedist(v[i], v[0], v[len(v)-1])
		if d > worstD {
			worst = i
			worstD = d
		}
	}
	if worstD <= tol {
		return []Vec2{v[0], v[len(v)-1]}
	}
	if worst <= 0 || worst >= len(v)-1 {
		panic("simply the worst")
	}
	lefts := simplifyPath(v[:worst+1], tol)
	rights := simplifyPath(v[worst:], tol)
	return append(lefts, rights[1:]...)
}

// Simplify removes points from paths, with the guarantee that
// all removed points are within the given tolerance (distance)
// from the new path.
func (ps *Paths) Simplify(tol float64) {
	vc := 0
	nvc := 0
	for i, p := range ps.P {
		vc += len(p.V)
		ps.P[i].V = simplifyPath(p.V, tol)
		nvc += len(ps.P[i].V)
	}
}
