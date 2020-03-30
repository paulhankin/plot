// Package paths provides tools for manipulating 2d paths consisting
// of line segments.
package paths

import "math"

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

// TightenBounds adjusts the bounds to exactly contain the paths.
// If there are no paths, the bounds are set to zero.
func (ps *Paths) TightenBounds() {
	inf := math.Inf(1)
	min := Vec2{inf, inf}
	max := Vec2{-inf, -inf}
	i := 0
	for _, p := range ps.P {
		for _, v := range p.V {
			i++
			min[0] = math.Min(min[0], v[0])
			min[1] = math.Min(min[1], v[1])
			max[0] = math.Max(max[0], v[0])
			max[1] = math.Max(max[1], v[1])
		}
	}
	if i == 0 {
		ps.Bounds = Bounds{}
		return
	}
	ps.Bounds = Bounds{
		Min: min,
		Max: max,
	}
}

// Translate moves all the paths by the given amount.
func (ps *Paths) Translate(dx Vec2) {
	b := ps.Bounds
	nb := Bounds{
		Min: vec2AddVec2(b.Min, dx),
		Max: vec2AddVec2(b.Max, dx),
	}
	ps.Transform(nb)
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

// move adds a new (initially empty) path starting at x,
// unless the last path already ends at x.
func (ps *Paths) move(x Vec2) {
	if len(ps.P) == 0 {
		ps.P = append(ps.P, Path{V: []Vec2{x}})
		return
	}
	p := &ps.P[len(ps.P)-1]
	if len(p.V) > 0 && p.V[len(p.V)-1] == x {
		return
	}
	ps.P = append(ps.P, Path{V: []Vec2{x}})
}

// line extends the last path with an edge that goes to x.
func (ps *Paths) line(x Vec2) {
	p := &ps.P[len(ps.P)-1]
	p.V = append(p.V, x)
}
