package paths

import (
	"bufio"
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"io/ioutil"
	"math"
	"os"
	"strconv"
	"strings"
	"text/scanner"
	"unicode"

	"github.com/JoshVarga/svgparser"
	"golang.org/x/net/html/charset"
)

var isSVGUnit = map[string]bool{
	"mm": true,
}

func parseDist(s string) (float64, string, error) {
	fp, up := s, ""
	for i, c := range s {
		if !unicode.IsDigit(c) && c != '.' && c != '-' && c != '+' {
			fp, up = s[:i], s[i:]
			break
		}
	}
	f, err := strconv.ParseFloat(fp, 64)
	if err != nil {
		return 0, "", err
	}
	if up != "" && !isSVGUnit[up] {
		return 0, "", fmt.Errorf("%q is not understood by this program as an SVG unit", up)
	}
	return f, up, nil
}

func parseBounds(e *svgparser.Element) (Bounds, error) {
	width, _, werr := parseDist(e.Attributes["width"])
	height, _, herr := parseDist(e.Attributes["height"])
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

func svgXformRotate(theta float64) *svgXform {
	c, s := math.Cos(theta), math.Sin(theta)
	return &svgXform{
		M: [3][3]float64{
			{c, s, 0},
			{-s, c, 0},
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
	case "matrix":
		fa, err := parseFloats(args)
		if err != nil {
			return nil, err
		}
		if len(fa) != 6 {
			return nil, fmt.Errorf("matrix transform should have 6 parameters: got %s", args)
		}
		return &svgXform{
			M: [3][3]float64{
				{fa[0], fa[2], fa[4]},
				{fa[1], fa[3], fa[5]},
				{0, 0, 1},
			},
		}, nil
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
	var neg bool
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
				if neg {
					neg = false
					return nil, fmt.Errorf("don't understand -)")
				}
				newxform, err := parseSingleXform(fname, args)
				if err != nil {
					return nil, err
				}
				xf = xf.Compose(newxform)
				state = xfsName
				args = nil
			} else if tok == '-' {
				neg = !neg
			} else if tok == scanner.Float || tok == scanner.Int {
				if neg {
					args = append(args, "-"+s.TokenText())
				} else {
					args = append(args, s.TokenText())
				}
				state = xfsMaybeComma
				neg = false
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

type cmdType int

const (
	cmdNone cmdType = iota
	cmdMove
	cmdLine
	cmdHorLine
	cmdVerLine
	cmdCurve
)

func (c cmdType) Args() int {
	switch c {
	case cmdNone:
		return 0
	case cmdMove, cmdLine:
		return 2
	case cmdHorLine, cmdVerLine:
		return 1
	case cmdCurve:
		return 6
	default:
		return 0
	}
}

type pathTokenizer struct {
	b *bytes.Buffer
}

const (
	eofRune   = rune(-1)
	floatRune = rune(-2)
)

type pathToken struct {
	r rune
	f float64
}

func (pt pathToken) String() string {
	if pt.r == eofRune {
		return "tokEOF{}"
	} else if pt.r == floatRune {
		return fmt.Sprintf("tokFloat{%g}", pt.f)
	}
	return fmt.Sprintf("tokRune{%c}", pt.r)
}

func (pt *pathTokenizer) nextFloat() (pathToken, error) {
	var b bytes.Buffer
	for {
		r, _, err := pt.b.ReadRune()
		if err == io.EOF {
			break
		}
		if err != nil {
			return pathToken{}, err
		}
		if (r >= '0' && r <= '9') || r == '.' || r == '-' {
			b.WriteRune(r)
			continue
		}
		break
	}
	f, err := strconv.ParseFloat(b.String(), 64)
	return pathToken{r: floatRune, f: f}, err
}

func (pt *pathTokenizer) Next() (pathToken, error) {
	for {
		r, _, err := pt.b.ReadRune()
		if err == io.EOF {
			return pathToken{r: eofRune}, nil
		}
		if err != nil {
			return pathToken{}, nil
		}
		if r == ',' || r == ' ' || unicode.IsSpace(r) {
			continue
		}
		if r >= '0' && r <= '9' || r == '-' {
			if err := pt.b.UnreadRune(); err != nil {
				return pathToken{}, err
			}
			return pt.nextFloat()
		}
		return pathToken{r: r}, nil
	}
}

func vec2AddVec2(a, b Vec2) Vec2 {
	return Vec2{a[0] + b[0], a[1] + b[1]}
}

func bez1(p0, p1, p2, p3 Vec2, t float64) Vec2 {
	a, b, c, d := (1-t)*(1-t)*(1-t), 3*(1-t)*(1-t)*t, 3*(1-t)*t*t, t*t*t
	return Vec2{a*p0[0] + b*p1[0] + c*p2[0] + d*p3[0], a*p0[1] + b*p1[1] + c*p2[1] + d*p3[1]}
}

func bezierInterpolate(target []Vec2, p0, p1, p2, p3 Vec2, start, end, d float64) []Vec2 {
	vs := bez1(p0, p1, p2, p3, start)
	ve := bez1(p0, p1, p2, p3, end)
	if end-start < 0.5 && vec2dist(vs, ve) < d {
		target = append(target, ve)
		return target
	}
	target = bezierInterpolate(target, p0, p1, p2, p3, start, (start+end)/2, d)
	return bezierInterpolate(target, p0, p1, p2, p3, (start+end)/2, end, d)
}

func parsePath(ps *Paths, xf *svgXform, e *svgparser.Element) error {
	bb := &pathTokenizer{bytes.NewBufferString(e.Attributes["d"])}
	var xy [6]float64
	var xyp int
	var rel bool
	var first, last Vec2
	var firstSet bool
	cmd := cmdNone
	for {
		token, err := bb.Next()
		if err != nil {
			return err
		}
		if token.r == eofRune {
			if xyp != 0 {
				return fmt.Errorf("got stray component in path")
			}
			return nil
		}
		p := token.r
		lp := unicode.ToLower(p)
		if lp == 'm' {
			// Move
			if xyp != 0 {
				return fmt.Errorf("got stray components before %c", p)
			}
			cmd, rel = cmdMove, (p == lp)
		} else if lp == 'l' {
			// Line To
			if xyp != 0 {
				return fmt.Errorf("got stray components before %c", p)
			}
			cmd, rel = cmdLine, (p == lp)
		} else if lp == 'v' || lp == 'h' {
			if xyp != 0 {
				return fmt.Errorf("got stray components before %c", p)
			}
			cmd, rel = cmdHorLine, (p == lp)
			if lp == 'v' {
				cmd = cmdVerLine
			}
		} else if lp == 'z' {
			// Close Path
			if !firstSet {
				return fmt.Errorf("got close path %c before any points", p)
			}
			if xyp != 0 {
				return fmt.Errorf("got stray components before %c", p)
			}
			ps.P[len(ps.P)-1].V = append(ps.P[len(ps.P)-1].V, xf.Apply(first))
			last = first
		} else if lp == 'c' {
			// Curve To
			cmd, rel = cmdCurve, (p == lp)
		} else if p == floatRune {
			xy[xyp] = token.f
			xyp++
			if xyp > cmd.Args() {
				return fmt.Errorf("got unexpected float value %v", xy[xyp-1])
			}
			if xyp == cmd.Args() {
				if cmd == cmdMove {
					path := Path{}
					ps.P = append(ps.P, path)
				}
				var v Vec2
				switch cmd {
				case cmdHorLine:
					v = Vec2{xy[0], last[1]}
					if rel {
						v[0] += last[0]
					}
					ps.P[len(ps.P)-1].V = append(ps.P[len(ps.P)-1].V, xf.Apply(v))
				case cmdVerLine:
					v = Vec2{last[0], xy[1]}
					if rel {
						v[1] += last[1]
					}
					ps.P[len(ps.P)-1].V = append(ps.P[len(ps.P)-1].V, xf.Apply(v))
				case cmdCurve:
					p0 := last
					p1 := Vec2{xy[0], xy[1]}
					p2 := Vec2{xy[2], xy[3]}
					p3 := Vec2{xy[4], xy[5]}
					if rel {
						p1 = vec2AddVec2(p1, last)
						p2 = vec2AddVec2(p2, last)
						p3 = vec2AddVec2(p3, last)
					}
					v = p3
					for _, v := range bezierInterpolate([]Vec2{}, p0, p1, p2, p3, 0, 1, 0.5) {
						ps.P[len(ps.P)-1].V = append(ps.P[len(ps.P)-1].V, xf.Apply(v))
					}
				case cmdLine, cmdMove:
					v = Vec2{xy[xyp-2], xy[xyp-1]}
					if rel {
						v = vec2AddVec2(v, last)
					}
					ps.P[len(ps.P)-1].V = append(ps.P[len(ps.P)-1].V, xf.Apply(v))
				}
				if !firstSet {
					first = v
					firstSet = true
				}
				last = v
				if cmd == cmdMove {
					cmd = cmdLine
				}
				xyp = 0
			}
		} else {
			return fmt.Errorf("got unknown token %c", p)
		}
	}
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

func parsePaths(p *Paths, pm map[string]*Paths, xform *svgXform, e *svgparser.Element) error {
	for _, c := range e.Children {
		cp := p
		id := c.Attributes["id"]
		if namedP, ok := pm[id]; id != "" && ok {
			cp = namedP
		}
		switch c.Name {
		case "g":
			gxf, err := parseSVGXForm(c.Attributes["transform"])
			if err != nil {
				return err
			}
			xf2 := xform.Compose(gxf)
			if err := parsePaths(cp, pm, xf2, c); err != nil {
				return err
			}
		case "path":
			if err := parsePath(cp, xform, c); err != nil {
				return err
			}
		case "line":
			if err := parseLine(cp, xform, c); err != nil {
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

func IDsFromSVG(r io.Reader, ids []string) (map[string]*Paths, error) {
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
	pathMap := map[string]*Paths{
		"": &Paths{Bounds: bs},
	}

	for _, id := range ids {
		if _, ok := pathMap[id]; ok {
			return nil, fmt.Errorf("id %q appears twice or more", id)
		}
		pathMap[id] = &Paths{Bounds: bs}
	}
	return pathMap, parsePaths(pathMap[""], pathMap, svgIdentity, elt)
}

// FromSVG parses an SVG file, extracting paths.
// This provides only limited SVG parsing support, and
// will fail or produce incorrect results if the SVG file
// uses features that it doesn't understand.
func FromSVG(r io.Reader) (*Paths, error) {
	pm, err := IDsFromSVG(r, nil)
	return pm[""], err
}

var (
	svgh = `<svg height="%dmm" width="%dmm" viewBox="%d %d %d %d" version="1.1" xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink">`
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
				wr("M %.2f %.2f", v[0], v[1])
			} else {
				wr(" %.2f %.2f", v[0], v[1])
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
