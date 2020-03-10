package paths

// frac returns s such that x*(1-s)+y*s = t
func frac(x, y, t float64) float64 {
	return (t - x) / (y - x)
}

type outcode uint

const (
	inside outcode = 0
	left   outcode = 1
	right  outcode = 2
	bottom outcode = 4
	top    outcode = 8
)

func computeOutcode(v Vec2, b Bounds) outcode {
	var c outcode
	if v[0] < b.Min[0] {
		c |= left
	} else if v[0] > b.Max[0] {
		c |= right
	}
	if v[1] < b.Min[1] {
		c |= bottom
	} else if v[1] > b.Max[1] {
		c |= top
	}
	return c
}

// Cohen-Sutherland clipping algorithm, from
// https://en.wikipedia.org/wiki/Cohen%E2%80%93Sutherland_algorithm
func clipLine(v0, v1 Vec2, b Bounds) (Vec2, Vec2, bool) {
	outcode0 := computeOutcode(v0, b)
	outcode1 := computeOutcode(v1, b)
	for {
		if outcode0 == 0 && outcode1 == 0 {
			return v0, v1, true
		} else if (outcode0 & outcode1) != 0 {
			return v0, v1, false
		}
		var outcodeOut outcode
		if outcode0 > outcode1 {
			outcodeOut = outcode0
		} else {
			outcodeOut = outcode1
		}

		var v Vec2
		if (outcodeOut & top) != 0 {
			v = Vec2{v0[0] + (v1[0]-v0[0])*(b.Max[1]-v0[1])/(v1[1]-v0[1]), b.Max[1]}
		} else if (outcodeOut & bottom) != 0 {
			v = Vec2{v0[0] + (v1[0]-v0[0])*(b.Min[1]-v0[1])/(v1[1]-v0[1]), b.Min[1]}
		} else if (outcodeOut & right) != 0 {
			v = Vec2{b.Max[0], v0[1] + (v1[1]-v0[1])*(b.Max[0]-v0[0])/(v1[0]-v0[0])}
		} else if (outcodeOut & left) != 0 {
			v = Vec2{b.Min[0], v0[1] + (v1[1]-v0[1])*(b.Min[0]-v0[0])/(v1[0]-v0[0])}
		}
		if outcodeOut == outcode0 {
			v0 = v
			outcode0 = computeOutcode(v0, b)
		} else {
			v1 = v
			outcode1 = computeOutcode(v1, b)
		}
	}
}

func clipPath(p Path, b Bounds) []Path {
	var parts []Path
	var curPath *Path
	var cont bool
	for i := 1; i < len(p.V); i++ {
		v0, v1, ok := clipLine(p.V[i-1], p.V[i], b)
		if !ok {
			cont = false
			continue
		}
		if v0 != p.V[i-1] || !cont {
			parts = append(parts, Path{})
			curPath = &parts[len(parts)-1]
			curPath.V = append(curPath.V, v0)
		}
		curPath.V = append(curPath.V, v1)
		cont = (v1 == p.V[i])
	}
	// remove parts with 0 or 1 vertices if any.
	j := 0
	for i := 0; i < len(parts); i++ {
		if len(parts[i].V) < 2 {
			continue
		}
		parts[j] = parts[i]
		j++
	}
	return parts[:j]
}

// Clip removes all line segments outside the given bounds.
// If a path crosses the bounds, it's broken into multiple paths.
func (ps *Paths) Clip(b Bounds) {
	var result []Path
	for _, p := range ps.P {
		parts := clipPath(p, b)
		result = append(result, parts...)
	}
	ps.P = result
}
