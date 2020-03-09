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

type flagSizeValue struct {
	X, Y float64
}

func (fs *flagSizeValue) String() string {
	return fmt.Sprintf("%.2f,%.2f", fs.X, fs.Y)
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
		fs.X, err = parseSizePart(parts[0])
		return err
	}
	if len(parts) > 2 {
		return fmt.Errorf("can't parse %q as size", s)
	}
	if fs.X, err = parseSizePart(parts[0]); err != nil {
		return err
	}
	if fs.Y, err = parseSizePart(parts[1]); err != nil {
		return err
	}
	return nil
}

// flags
var (
	flagIn  string
	flagOut string

	flagDelta     flagSizeValue
	flagSize      flagSizeValue
	flagPaperSize flagSizeValue
	flagCenter    bool
	flagPenUp     int
	flagFeedRate  int
)

func init() {
	flag.StringVar(&flagIn, "in", "", "svg input file")
	flag.StringVar(&flagOut, "out", "out.gcode", "gcode output file")
	flag.Var(&flagDelta, "offset", "displacement of 0,0 from pen origin")
	flag.Var(&flagSize, "size", "target size of image (mm)")
	flag.Var(&flagPaperSize, "paper", "target size of paper (mm)")
	flag.BoolVar(&flagCenter, "center", false, "if set, center image on paper")
	flag.IntVar(&flagPenUp, "penup", 40, "how much to lift pen when moving")
	flag.IntVar(&flagFeedRate, "feed", 800, "feed rate when drawing (mm/min)")
}

func parseSVG(name string) (*svg.Svg, error) {
	f, err := os.Open(name)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return svg.ParseSvgFromReader(f, "", 1.0)
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

	ow := ps.Bounds.Max[0] - ps.Bounds.Min[0]
	oh := ps.Bounds.Max[1] - ps.Bounds.Min[1]
	if flagSize.X == 0 && flagSize.Y == 0 {
		flagSize.X = ow
		flagSize.Y = oh
	} else if flagSize.Y == 0 {
		flagSize.Y = flagSize.Y * oh / ow
	} else if flagSize.X == 0 {
		flagSize.X = flagSize.X * ow / oh
	}

	if !(math.Abs(flagSize.X/flagSize.Y-ow/oh) < 1e-3) {
		fail("target image size %s not compatible with image size %g,%g", &flagSize, ow, oh)
	}

	if flagPaperSize.X != 0 || flagPaperSize.Y != 0 {
		if flagPaperSize.X == 0 || flagPaperSize.Y == 0 {
			fail("paper size %g,%g doesn't make sense", flagPaperSize.X, flagPaperSize.Y)
		}

		if flagSize.X > flagPaperSize.X || flagSize.Y > flagPaperSize.Y {
			fail("paper size %g,%g is smaller than image %g,%g", flagPaperSize.X, flagPaperSize.Y, flagSize.X, flagSize.Y)
		}

	}
	if flagCenter {
		if flagPaperSize.X == 0 {
			fail("must set -papersize to use -center")
		}
		flagDelta.X += (flagPaperSize.X - flagSize.X) / 2
		flagDelta.Y += (flagPaperSize.Y - flagSize.Y) / 2
	}

	ps.Transform(paths.Bounds{
		Min: paths.Vec2{flagDelta.X, flagDelta.Y},
		Max: paths.Vec2{flagSize.X + flagDelta.X, flagSize.Y + flagDelta.Y},
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
