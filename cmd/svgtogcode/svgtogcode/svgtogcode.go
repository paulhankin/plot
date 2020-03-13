// Package svgtogcode provides the functionality for the
// svgtocode binary as a library.
package svgtogcode

import (
	"fmt"
	"math"
	"os"
	"path/filepath"

	"github.com/paulhankin/plot/gcode"
	"github.com/paulhankin/plot/paths"
)

type Config struct {
	In  string
	Out string

	Delta     paths.Vec2
	Size      paths.Vec2
	PaperSize paths.Vec2
	Center    bool
	PenUp     int
	FeedRate  int

	Split   bool
	Reverse bool

	Simplify float64
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
		return paths.Bounds{}, fmt.Errorf("target image size %v not compatible with image size %g,%g", &sz, ow, oh)
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

func Convert(cfg *Config) error {
	if cfg.In == "" {
		return fmt.Errorf("input file must be specified")
	}

	ps, err := func() (*paths.Paths, error) {
		f, err := os.Open(cfg.In)
		if err != nil {
			return nil, err
		}
		defer f.Close()
		return paths.FromSVG(f)
	}()
	if err != nil {
		return err
	}

	bounds, err := adjustSize(cfg.Size, cfg.PaperSize, cfg.Delta, cfg.Center, ps.Bounds)
	if err != nil {
		return err
	}

	ps.Transform(bounds)
	ps.Clip(ps.Bounds)
	if cfg.Simplify > 0 {
		ps.Simplify(cfg.Simplify)
	}

	ps.Sort(&paths.SortConfig{
		Split:   cfg.Split,
		Reverse: cfg.Reverse,
	})

	gcodeOut, err := os.Create(cfg.Out)
	if err != nil {
		return fmt.Errorf("failed to open output file: %w", err)
	}

	if filepath.Ext(cfg.Out) == ".svg" {
		err := ps.SVG(gcodeOut)
		if err == nil {
			err = gcodeOut.Close()
		}
		if err != nil {
			return fmt.Errorf("failed to write svg file: %w", err)
		}
		return nil
	}

	gcodeWriter := gcode.NewWriter(gcodeOut, &gcode.Config{
		PenUp:    cfg.PenUp,
		FeedRate: cfg.FeedRate,
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
		return fmt.Errorf("failed to write gcode: %w", err)
	}

	if err := gcodeOut.Close(); err != nil {
		return fmt.Errorf("failed to write gcode: %w", err)
	}
	return nil
}
