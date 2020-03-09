package paths

type Vec2 [2]float64

type Path struct {
	V []Vec2
}

type Bounds struct {
	Min, Max Vec2
}

type Paths struct {
	Bounds Bounds
	P      []Path
}

func (ps *Paths) Transform(nb Bounds) {
	ob := ps.Bounds
	for _, p := range ps.P {
		for i, v := range p.V {
			x, y := v[0], v[1]
			x -= ob.Min[0]
			x /= ob.Max[0] - ob.Min[0]
			x *= nb.Max[0] - nb.Min[0]
			x += nb.Min[0]

			y -= ob.Min[1]
			y /= ob.Max[1] - ob.Min[1]
			y *= nb.Max[1] - nb.Min[1]
			y += nb.Min[1]
			p.V[i] = [2]float64{x, y}
		}
	}
	ps.Bounds = nb
}
