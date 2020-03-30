package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"unicode"

	"github.com/paulhankin/plot/paths"
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

type Config struct {
	Out                                string
	BorderLeft, BorderRight, BorderTop float64
	XSize                              float64
	PaperSize                          paths.Vec2
	Text                               string
}

var config Config

func init() {
	flag.StringVar(&config.Out, "out", "out.svg", "svg output file")
	flag.Float64Var(&config.BorderLeft, "border_left", 10, "border left (mm)")
	flag.Float64Var(&config.BorderRight, "border_right", 10, "border right (mm)")
	flag.Float64Var(&config.BorderTop, "border_top", 10, "border top (mm)")
	flag.Float64Var(&config.XSize, "xsize", 8, "height of x character (mm)")
	flag.Var((*flagSizeValue)(&config.PaperSize), "paper", "target size x,y of paper (mm)")
	flag.StringVar(&config.Text, "text", "The quick brown fox jumps over the lazy dog", "text to render")
}

func loadFont() (*paths.Font, error) {
	f, err := os.Open("data/blockscript.svg")
	if err != nil {
		return nil, err
	}
	defer f.Close()
	fc := &paths.FontConfig{
		Glyph:       map[rune]paths.FontGlyphConfig{},
		AdvanceRune: 'x',
		Advance:     0.1,
		SpaceRune:   'x',
		Space:       1.3,
		LineRune:    'I',
		Line:        1.9,
	}
	layout := [][]rune{
		[]rune("ABCDEFGHIJKLM"),
		[]rune("NOPQRSTUVWXYZ"),
		[]rune("abcdefghijklm"),
		[]rune("nopqrstuvwxyz"),
		[]rune("012345679"),
		[]rune(".,"),
	}
	sym := map[rune]string{
		'.': "stop",
		',': "comma",
	}
	rn := func(r rune) string {
		if unicode.IsUpper(r) {
			return fmt.Sprintf("capital_%c", r)
		} else if unicode.IsLower(r) {
			return fmt.Sprintf("lower_%c", r)
		} else if unicode.IsDigit(r) {
			return fmt.Sprintf("digit_%c", r)
		}
		n, ok := sym[r]
		if !ok || n == "" {
			log.Fatalf("failed to find rune name for %c", r)
		}
		return n
	}
	for i, row := range layout {
		for j, r := range row {
			fc.Glyph[r] = paths.FontGlyphConfig{
				Dot: paths.Vec2{float64(j)*10 + 2, float64(i+1)*10 - 2},
				ID:  rn(r),
			}
		}
	}
	return paths.NewFont(f, fc)
}

func renderSVG(n string, ps *paths.Paths) error {
	f, err := os.Create(n)
	if err != nil {
		return err
	}
	if err := ps.SVG(f); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}

func main() {
	failf := func(s string, args ...interface{}) {
		fmt.Fprintf(os.Stderr, s+"\n", args...)
		os.Exit(1)
	}
	flag.Parse()
	font, err := loadFont()
	if err != nil {
		failf("failed to load font: %v", err)
	}

	if false {
		for r := rune('A'); r <= rune('Z'); r++ {
			g := font.Glyph[r]
			fmt.Printf("%c: width: %f, height: %f, advance: %f\n", r, g.Width, g.Height, g.Advance)
		}
	}

	scale, err := font.ScaleFromRuneHeight('x', config.XSize)
	if err != nil {
		failf("failed to get font scale: %v", err)
	}
	pgs, err := paths.LayoutText(font, config.Text, scale, config.PaperSize[0]-config.BorderLeft-config.BorderRight)
	if err != nil {
		failf("failed to render text: %v", err)
	}
	ps := paths.GlyphsToPaths(paths.Vec2{config.BorderLeft, config.BorderTop}, pgs)
	ps.Bounds = paths.Bounds{
		Min: paths.Vec2{0, 0},
		Max: config.PaperSize,
	}
	if err := renderSVG(config.Out, ps); err != nil {
		failf("failed to save svg: %v", err)
	}

}
