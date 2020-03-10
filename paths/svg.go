package paths

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strconv"
	"strings"

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

func parsePath(ps *Paths, e *svgparser.Element) error {
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
			ps.P[len(ps.P)-1].V = append(ps.P[len(ps.P)-1].V, xy)
			move = false
			xyp = 0
		}
	}
	if xyp != 0 {
		return fmt.Errorf("got stray component in path")
	}
	return nil
}

func parsePaths(p *Paths, e *svgparser.Element) error {
	for _, c := range e.Children {
		switch c.Name {
		case "g":
			if err := parsePaths(p, c); err != nil {
				return err
			}
		case "path":
			if err := parsePath(p, c); err != nil {
				return err
			}
		default:
			fmt.Fprintf(os.Stderr, "unknown child node type %q\n", c.Name)
		}
	}
	return nil
}

// FromSVG reads the paths from an SVG file.
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
	return p, parsePaths(p, elt)
}
