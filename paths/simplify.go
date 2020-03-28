package paths

import (
	"math"
)

func vec2linedist(v, s, e Vec2) float64 {
	ds := vec2dist(v, s)
	de := vec2dist(v, e)
	diff := Vec2{e[0] - s[0], e[1] - s[1]}
	dlen := math.Sqrt(diff[0]*diff[0] + diff[1]*diff[1])
	if dlen == 0 {
		return ds
	}
	dp := math.Abs(diff[1]*v[0]-diff[0]*v[1]+e[0]*s[1]-e[1]*s[0]) / dlen
	return math.Min(math.Min(dp, ds), de)
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
	for i, p := range ps.P {
		ps.P[i].V = simplifyPath(p.V, tol)
	}
}
