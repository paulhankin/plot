package paths

// Vec2 is a 2-dimensional vector.
type Vec2 [2]float64

// A Path is a contiguous series of line segments, from the
// first point in the V slice to the last.
type Path struct {
	V []Vec2
}

// Bounds describes an axis-aligned bounding box.
type Bounds struct {
	Min, Max Vec2
}

// Paths is a set of paths, along with a view bounds.
type Paths struct {
	Bounds Bounds
	P      []Path
}

// Transform resizes all paths so that the rectangle forming the
// current bounds is the size of the new bounds. The bounds
// are also updated to the new bounds.
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
