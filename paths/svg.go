package paths

import (
	"fmt"
	"io"
	"log"
	"os"

	"github.com/rustyoz/svg"
)

func parseSVG(name string) (*svg.Svg, error) {
	f, err := os.Open(name)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return svg.ParseSvgFromReader(f, "", 1.0)
}

// FromSVG reads the paths from an SVG file.
func FromSVG(r io.Reader) (*Paths, error) {
	svgIn, err := svg.ParseSvgFromReader(r, "", 1.0)
	if err != nil {
		return nil, err
	}

	view, err := svgIn.ViewBoxValues()
	if err != nil {
		return nil, fmt.Errorf("failed to read viewbox values: %v", err)
	}

	p := &Paths{
		Bounds: Bounds{
			Min: Vec2{view[0], view[1]},
			Max: Vec2{view[0] + view[2], view[1] + view[3]},
		},
	}

	dic, erc := svgIn.ParseDrawingInstructions()

	ddrain := false
	edrain := false
	for !ddrain || !edrain {
		select {
		case ins, ok := <-dic:
			if !ok {
				dic = nil
				ddrain = true
				continue
			}
			switch ins.Kind {
			case svg.LineInstruction:
				pi := len(p.P) - 1
				if pi < 0 {
					return nil, fmt.Errorf("line issued before move")
				}
				p.P[pi].V = append(p.P[pi].V, Vec2{ins.M[0], ins.M[1]})
			case svg.MoveInstruction:
				p.P = append(p.P, Path{})
				pi := len(p.P) - 1
				p.P[pi].V = append(p.P[pi].V, Vec2{ins.M[0], ins.M[1]})
			case svg.PaintInstruction:
				// issued at the end of every path
			default:
				log.Fatalf("unhandled instruction type %v", ins.Kind)
			}

		case err, ok := <-erc:
			if !ok {
				edrain = true
				erc = nil
				continue
			}
			return nil, fmt.Errorf("svg parse failed: %v", err)
		}
	}
	return p, nil
}
