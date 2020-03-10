package main

import (
	"flag"
	"fmt"
	"math"
	"os"
	"strconv"
	"strings"

	"github.com/paulhankin/plot/gcode"
	"github.com/paulhankin/plot/paths"
	"github.com/rustyoz/svg"
)

type flagSizeValue paths.Vec2

func (fs *flagSizeValue) String() string {
	return fmt.Sprintf("%.2f,%.2f", fs[0], fs[1])
}

func parseSizePart(s string) (float64, error) {
	s = strings.TrimSpace(s)
	if len(s) == 0 {
		return 0, nil
	}
	return strconv.ParseFloat(s, 64)
}

func (fs *flagSizeValue) Set(s string) error {
	var err error
	parts := strings.Split(s, ",")
	if len(parts) == 1 {
		fs[0], err = parseSizePart(parts[0])
		return err
	}
	if len(parts) > 2 {
		return fmt.Errorf("can't parse %q as size", s)
	}
	if fs[0], err = parseSizePart(parts[0]); err != nil {
		return err
	}
	if fs[1], err = parseSizePart(parts[1]); err != nil {
		return err
	}
	return nil
}

// flags
var (
	flagIn  string
	flagOut string

	flagDelta     paths.Vec2
	flagSize      paths.Vec2
	flagPaperSize paths.Vec2
	flagCenter    bool
	flagPenUp     int
	flagFeedRate  int

	flagSplit   bool
	flagReverse bool

	flagSimplify float64
)

func init() {
	flag.StringVar(&flagIn, "in", "", "svg input file")
	flag.StringVar(&flagOut, "out", "out.gcode", "gcode output file")
	flag.Var((*flagSizeValue)(&flagDelta), "offset", "displacement of 0,0 from pen origin")
	flag.Var((*flagSizeValue)(&flagSize), "size", "target size of image (mm)")
	flag.Var((*flagSizeValue)(&flagPaperSize), "paper", "target size of paper (mm)")
	flag.BoolVar(&flagCenter, "center", false, "if set, center image on paper")
	flag.IntVar(&flagPenUp, "penup", 40, "how much to lift pen when moving")
	flag.IntVar(&flagFeedRate, "feed", 800, "feed rate when drawing (mm/min)")
	flag.BoolVar(&flagSplit, "split", true, "allow paths to be split to reduce pen movement")
	flag.BoolVar(&flagReverse, "reverse", true, "allow paths to be drawn backwards to reduce pen movement")
	flag.Float64Var(&flagSimplify, "simplify", 0.1, "simplify paths within this tolerance (0=disabled)")
}

func parseSVG(name string) (*svg.Svg, error) {
	f, err := os.Open(name)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return svg.ParseSvgFromReader(f, "", 1.0)
}

func adjustSize(sz, ps, delta paths.Vec2, center bool, b paths.Bounds) (paths.Bounds, error) {
	ow := b.Max[0] - b.Min[0]
	oh := b.Max[1] - b.Min[1]
	if sz[0] == 0 && sz[1] == 0 {
		sz[0] = ow
		sz[1] = oh
	} else if sz[1] == 0 {
		sz[1] = sz[1] * oh / ow
	} else if sz[0] == 0 {
		sz[0] = sz[0] * ow / oh
	}

	if !(math.Abs(sz[0]/sz[1]-ow/oh) < 1e-3) {
		return paths.Bounds{}, fmt.Errorf("target image size %s not compatible with image size %g,%g", &sz, ow, oh)
	}

	if ps[0] != 0 || ps[1] != 0 {
		if ps[0] == 0 || ps[1] == 0 {
			return paths.Bounds{}, fmt.Errorf("paper size %g,%g doesn't make sense", ps[0], ps[1])
		}

		if sz[0] > ps[0] || sz[1] > ps[1] {
			return paths.Bounds{}, fmt.Errorf("paper size %g,%g is smaller than image %g,%g", ps[0], ps[1], sz[0], sz[1])
		}
	}

	if center {
		if ps[0] == 0 {
			return paths.Bounds{}, fmt.Errorf("must set -papersize to use -center")
		}
		delta[0] += (ps[0] - sz[0]) / 2
		delta[1] += (ps[1] - sz[1]) / 2
	}

	return paths.Bounds{
		Min: paths.Vec2{delta[0], delta[1]},
		Max: paths.Vec2{sz[0] + delta[0], sz[1] + delta[1]},
	}, nil
}
func vec2lerp(x, y paths.Vec2, s float64) paths.Vec2 {
	return paths.Vec2{x[0]*(1-s) + y[0]*s, x[1]*(1-s) + y[1]*s}
}

func main() {
	fail := func(s string, args ...interface{}) {
		fmt.Fprintf(os.Stderr, s+"\n", args...)
		os.Exit(2)
	}

	flag.Parse()
	if flagIn == "" {
		fail("must specify -in <svg file>")
	}

	ps, err := func() (*paths.Paths, error) {
		f, err := os.Open(flagIn)
		if err != nil {
			return nil, err
		}
		defer f.Close()
		return paths.FromSVG(f)
	}()
	if err != nil {
		fail("%s", err)
	}

	bounds, err := adjustSize(flagSize, flagPaperSize, flagDelta, flagCenter, ps.Bounds)
	if err != nil {
		fail("%s", err)
	}

	ps.Transform(bounds)
	ps.Clip(ps.Bounds)
	if flagSimplify > 0 {
		ps.Simplify(flagSimplify)
	}

	ps.Sort(&paths.SortConfig{
		Split:   flagSplit,
		Reverse: flagReverse,
	})

	gcodeOut, err := os.Create(flagOut)
	if err != nil {
		fail("failed to open gcode output file: %v", err)
	}

	gcodeWriter := gcode.NewWriter(gcodeOut, &gcode.Config{
		PenUp:    flagPenUp,
		FeedRate: flagFeedRate,
	})

	gcodeWriter.Preamble()

	for _, p := range ps.P {
		for i, v := range p.V {
			if i == 0 {
				gcodeWriter.Move(v[0], v[1])
			} else {
				gcodeWriter.Line(v[0], v[1])
			}
		}
	}

	gcodeWriter.Postamble()

	if err := gcodeWriter.Flush(); err != nil {
		fail("failed to write gcode: %v", err)
	}

	if err := gcodeOut.Close(); err != nil {
		fail("failed to write gcode: %v", err)
	}
}
