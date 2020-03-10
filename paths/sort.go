package paths

import (
	"math"
	"sort"
)

type SortConfig struct {
	Split   bool // ok to split continuous paths
	Reverse bool // ok to draw paths in the reverse direction
}

// A verticle is a vertex (the "start" vertex of the path),
// with a link to the other "end" of the path.
// This might be an adjacent vertex on the path, or it might
// summarize the whole path from start to end.
type verticle struct {
	path       int // which path it's from
	start, end int // start and end index of segment
}

func (v verticle) reversed() verticle {
	v.start, v.end = v.end, v.start
	return v
}

type vindexNode struct {
	x           Vec2
	v           verticle
	yaxis       bool
	left, right *vindexNode
}

type vindex struct {
	minR float64
	m    map[verticle]struct{}
	node *vindexNode
}

func buildIndex(ps *Paths, vs []verticle, yaxis bool) *vindexNode {
	if len(vs) == 0 {
		return nil
	}
	sort.Slice(vs, func(i, j int) bool {
		vi := ps.P[vs[i].path].V[vs[i].start]
		vj := ps.P[vs[j].path].V[vs[j].start]
		if yaxis {
			return vi[1] < vj[1]
		} else {
			return vi[0] < vj[0]
		}
	})

	if true {
		// check "left" is always the smaller coordinates
		first := vs[0]
		last := vs[len(vs)-1]
		if yaxis && ps.P[last.path].V[last.start][1] < ps.P[first.path].V[first.start][1] {
			panic("y")
		}
		if !yaxis && ps.P[last.path].V[last.start][0] < ps.P[first.path].V[first.start][0] {
			panic("x")
		}
	}

	k := len(vs) / 2
	return &vindexNode{
		x:     ps.P[vs[k].path].V[vs[k].start],
		v:     vs[k],
		yaxis: yaxis,
		left:  buildIndex(ps, vs[:k], !yaxis),
		right: buildIndex(ps, vs[k+1:], !yaxis),
	}
}

func indexVerticles(ps *Paths, vs []verticle, minR float64) *vindex {
	m := map[verticle]struct{}{}
	for _, v := range vs {
		m[v] = struct{}{}
	}
	node := buildIndex(ps, vs, false)
	return &vindex{
		minR: minR,
		m:    m,
		node: node,
	}
}

type vcand struct {
	dist float64
	v    verticle
}

func vec2dist(v0, v1 Vec2) float64 {
	dx := v0[0] - v1[0]
	dy := v0[1] - v1[1]
	return math.Sqrt(dx*dx + dy*dy)
}

func (vi *vindex) findRadius(vn *vindexNode, pos Vec2, r float64) []vcand {
	if vn == nil {
		return nil
	}
	var cand []vcand
	d := vec2dist(vn.x, pos)
	if d <= r {
		if _, ok := vi.m[vn.v]; ok {
			cand = append(cand, vcand{dist: d, v: vn.v})
		}
	}

	left := false // whether our pos is on left or right
	var axdist float64
	if vn.yaxis && pos[1] <= vn.x[1] {
		left = true
		axdist = math.Abs(pos[1] - vn.x[1])
	}
	if !vn.yaxis && pos[0] <= vn.x[0] {
		left = true
		axdist = math.Abs(pos[0] - vn.x[0])
	}

	if left {
		cand = append(cand, vi.findRadius(vn.left, pos, r)...)
		if axdist <= r {
			cand = append(cand, vi.findRadius(vn.right, pos, r)...)
		}
	} else {
		cand = append(cand, vi.findRadius(vn.right, pos, r)...)
		if axdist <= r {
			cand = append(cand, vi.findRadius(vn.left, pos, r)...)
		}
	}

	return cand
}

func (vi *vindex) popNearest(pos Vec2) verticle {
	r := vi.minR
	for {
		cands := vi.findRadius(vi.node, pos, r)
		if len(cands) > 0 {
			best := 0
			for i := 1; i < len(cands); i++ {
				if cands[i].dist < cands[best].dist {
					best = i
				}
			}
			v := cands[best].v
			delete(vi.m, v)
			delete(vi.m, v.reversed())
			return v
		}
		r *= 2
	}
	panic("no verticles")
}

func sortVerticles(ps *Paths, vs []verticle, want int) []verticle {
	minR := (ps.Bounds.Max[0] - ps.Bounds.Min[0]) / 20
	idx := indexVerticles(ps, vs, minR)
	res := make([]verticle, 0, want)
	var pos Vec2
	for len(res) < want {
		v := idx.popNearest(pos)
		res = append(res, v)
		pos = ps.P[v.path].V[v.end]
	}
	return res
}

func (ps *Paths) Sort(cfg *SortConfig) {
	// construct all the verticles
	var vs []verticle
	for i, p := range ps.P {
		if cfg.Split {
			for j := 0; j < len(p.V)-1; j++ {
				vs = append(vs, verticle{i, j, j + 1})
				if cfg.Reverse {
					vs = append(vs, verticle{i, j + 1, j})
				}
			}
		} else {
			vs = append(vs, verticle{i, 0, len(p.V) - 1})
			if cfg.Reverse {
				vs = append(vs, verticle{i, len(p.V) - 1, 0})
			}
		}
	}
	n := len(vs)
	if cfg.Reverse {
		n /= 2
	}
	svs := sortVerticles(ps, vs, n)

	np := &Paths{Bounds: ps.Bounds}
	for _, v := range svs {
		d := 1
		if v.end < v.start {
			d = -1
		}
		for i := v.start; i != v.end; i += d {
			np.move(ps.P[v.path].V[i])
			np.line(ps.P[v.path].V[i+d])
		}
	}
	*ps = *np
}
