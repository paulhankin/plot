package paths

import (
	"bufio"
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
	"text/scanner"

	"github.com/JoshVarga/svgparser"
	"golang.org/x/net/html/charset"
)

func parseBounds(e *svgparser.Element) (Bounds, error) {
	width, werr := strconv.ParseFloat(e.Attributes["width"], 64)
	height, herr := strconv.ParseFloat(e.Attributes["height"], 64)
	if werr != nil {
		return Bounds{}, werr
	}
	if herr != nil {
		return Bounds{}, herr
	}
	// TODO: parse view box
	return Bounds{
		Max: Vec2{float64(width), float64(height)},
	}, nil
}

func parseLine(ps *Paths, xform *svgXform, e *svgparser.Element) error {
	var ferr error
	pf := func(s string) float64 {
		if ferr != nil {
			return 0
		}
		f, err := strconv.ParseFloat(s, 64)
		ferr = err
		return f
	}
	x1 := pf(e.Attributes["x1"])
	x2 := pf(e.Attributes["x2"])
	y1 := pf(e.Attributes["y1"])
	y2 := pf(e.Attributes["y2"])
	ps.move(xform.Apply(Vec2{x1, y1}))
	ps.line(xform.Apply(Vec2{x2, y2}))
	return ferr
}

type xformScannerState int

const (
	xfsName xformScannerState = 1 + iota
	xfsBra
	xfsMaybeComma
	xfsArg
)

func parseFloats(a []string) ([]float64, error) {
	var r []float64
	for _, x := range a {
		f, err := strconv.ParseFloat(x, 64)
		if err != nil {
			return nil, err
		}
		r = append(r, f)
	}
	return r, nil
}

func svgXformTranslate(x, y float64) *svgXform {
	return &svgXform{
		M: [3][3]float64{
			{1, 0, x},
			{0, 1, y},
			{0, 0, 1},
		},
	}
}

func svgXformScale(x, y float64) *svgXform {
	return &svgXform{
		M: [3][3]float64{
			{x, 0, 0},
			{0, y, 0},
			{0, 0, 1},
		},
	}
}

func parseSingleXform(name string, args []string) (*svgXform, error) {
	switch name {
	case "translate":
		fa, err := parseFloats(args)
		if err != nil {
			return nil, err
		}
		if len(fa) != 1 && len(fa) != 2 {
			return nil, fmt.Errorf("translate should have one or two parameters: got %s", args)
		}
		if len(fa) == 1 {
			fa = append(fa, 0)
		}
		return svgXformTranslate(fa[0], fa[1]), nil
	case "scale":
		fa, err := parseFloats(args)
		if err != nil {
			return nil, err
		}
		if len(fa) != 1 && len(fa) != 2 {
			return nil, fmt.Errorf("scale should have one or two parameters: got %s", args)
		}
		if len(fa) == 1 {
			fa = append(fa, fa[0])
		}
		return svgXformScale(fa[0], fa[1]), nil
	default:
		return nil, fmt.Errorf("unknown transform function %q", name)
	}
}

func parseSVGXForm(x string) (*svgXform, error) {
	var s scanner.Scanner
	xf := svgIdentity
	s.Init(strings.NewReader(x))
	state := xfsName
	fname := ""
	var args []string
	for tok := s.Scan(); tok != scanner.EOF; tok = s.Scan() {
		switch state {
		case xfsName:
			if tok != scanner.Ident {
				return nil, fmt.Errorf("failed to parse transform: expected transform name, but got %q", s.TokenText())
			}
			fname = s.TokenText()
			state = xfsBra
		case xfsBra:
			if tok != '(' {
				return nil, fmt.Errorf("failed to parse transform: expected (, but got %q", s.TokenText())
			}
			state = xfsArg
		case xfsMaybeComma:
			if tok == ',' {
				continue
			}
			fallthrough
		case xfsArg:
			if tok == ')' {
				newxform, err := parseSingleXform(fname, args)
				if err != nil {
					return nil, err
				}
				xf = xf.Compose(newxform)
				state = xfsName
				args = nil
			} else if tok == scanner.Float || tok == scanner.Int {
				args = append(args, s.TokenText())
				state = xfsMaybeComma
			} else {
				return nil, fmt.Errorf("unexpected token %q parsing transform %q", s.TokenText(), x)
			}
		}
	}
	if state != xfsName {
		return nil, fmt.Errorf("failed to parse transform: %q", x)
	}
	return xf, nil
}

func parsePath(ps *Paths, xf *svgXform, e *svgparser.Element) error {
	parts := strings.Fields(e.Attributes["d"])
	move := false
	var xy Vec2
	var xyp int
	for _, p := range parts {
		if p == "M" {
			if xyp != 0 {
				return fmt.Errorf("got odd number of components before M")
			}
			move = true
			continue
		}
		if p == "L" {
			if xyp != 0 {
				return fmt.Errorf("got odd number of components before L")
			}
			continue
		}
		p = strings.TrimRight(p, ",")
		x, err := strconv.ParseFloat(p, 64)
		if err != nil {
			return err
		}
		xy[xyp] = x
		xyp++
		if xyp == 2 {
			if move {
				path := Path{}
				ps.P = append(ps.P, path)
			}
			ps.P[len(ps.P)-1].V = append(ps.P[len(ps.P)-1].V, xf.Apply(xy))
			move = false
			xyp = 0
		}
	}
	if xyp != 0 {
		return fmt.Errorf("got stray component in path")
	}
	return nil
}

type svgXform struct {
	M [3][3]float64
}

func (xf *svgXform) Compose(xf2 *svgXform) *svgXform {
	var a svgXform
	for i := 0; i < 3; i++ {
		for j := 0; j < 3; j++ {
			for k := 0; k < 3; k++ {
				a.M[i][k] += xf.M[i][j] * xf2.M[j][k]
			}
		}
	}
	return &a
}

func (xf *svgXform) Apply(v Vec2) Vec2 {
	x := [3]float64{v[0], v[1], 1.0}
	var r [3]float64
	for i := 0; i < 3; i++ {
		for j := 0; j < 3; j++ {
			r[i] += xf.M[i][j] * x[j]
		}
	}
	return Vec2{r[0] / r[2], r[1] / r[2]}
}

var svgIdentity = &svgXform{
	M: [3][3]float64{{1, 0, 0}, {0, 1, 0}, {0, 0, 1}},
}

func parsePaths(p *Paths, xform *svgXform, e *svgparser.Element) error {
	for _, c := range e.Children {
		switch c.Name {
		case "g":
			gxf, err := parseSVGXForm(c.Attributes["transform"])
			if err != nil {
				return err
			}
			xf2 := xform.Compose(gxf)
			if err := parsePaths(p, xf2, c); err != nil {
				return err
			}
		case "path":
			if err := parsePath(p, xform, c); err != nil {
				return err
			}
		case "line":
			if err := parseLine(p, xform, c); err != nil {
				return err
			}
		case "defs":
			continue
		default:
			fmt.Fprintf(os.Stderr, "unknown child node type %q\n", c.Name)
		}
	}
	return nil
}

// FromSVG parses an SVG file, extracting paths.
// This provides only limited SVG parsing support, and
// will fail or produce incorrect results if the SVG file
// uses features that it doesn't understand.
func FromSVG(r io.Reader) (p *Paths, rerr error) {
	raw, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}
	decoder := xml.NewDecoder(bytes.NewReader(raw))
	decoder.CharsetReader = charset.NewReaderLabel
	elt, err := svgparser.DecodeFirst(decoder)
	if err != nil {
		return nil, err
	}
	if err := elt.Decode(decoder); err != nil && err != io.EOF {
		return nil, err
	}
	bs, err := parseBounds(elt)
	if err != nil {
		return nil, err
	}
	p = &Paths{Bounds: bs}
	return p, parsePaths(p, svgIdentity, elt)
}

var (
	svgh = `<svg height="%d" width="%d" viewBox="%d %d %d %d" version="1.1" xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink">`
)

// SVG writes an SVG file that contains black strokes along the paths.
func (ps *Paths) SVG(w io.Writer) error {
	var werr error
	bi := bufio.NewWriter(w)
	wr := func(f string, args ...interface{}) {
		if werr != nil {
			return
		}
		_, werr = fmt.Fprintf(bi, f, args...)
	}
	wr(svgh, int(ps.Bounds.Max[1]), int(ps.Bounds.Max[0]), int(ps.Bounds.Min[0]), int(ps.Bounds.Min[1]), int(ps.Bounds.Max[0]-ps.Bounds.Min[0]), int(ps.Bounds.Max[1]-ps.Bounds.Min[1]))
	wr("\n")
	wr("<g fill=\"none\" stroke=\"black\" stroke-width=\"0.1\">\n")
	for _, p := range ps.P {
		if len(p.V) == 0 {
			continue
		}
		wr(`<path d="`)
		for i, v := range p.V {
			if i == 0 {
				wr("M %.2f, %.2f", v[0], v[1])
			} else {
				wr(" %.2f, %.2f", v[0], v[1])
			}
		}
		wr("\"/>\n")
	}
	wr("</g>")
	wr("</svg>")
	if werr == nil {
		werr = bi.Flush()
	}
	return werr
}
