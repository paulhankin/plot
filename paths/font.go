package paths

import (
	"fmt"
	"io"
	"unicode"
	"unicode/utf8"
)

// FontGlyphConfig describes how to find a particular glyph in
// an SVG file.
type FontGlyphConfig struct {
	Dot          Vec2    // where in the svg the letter is to be found.
	ID           string  // the ID of the group/path in the SVG file
	DeltaAdvance float64 // if set, how much to fiddle the advance for this glyph
}

// FontConfig describes an SVG file that contains glyphs for a font.
type FontConfig struct {
	Glyph       map[rune]FontGlyphConfig // how to extract glyphs from SVG
	AdvanceRune rune                     // if set, multiply the advance by the width of this rune
	Advance     float64                  // how much advance to use (which is added to the width of the glyph).
	SpaceRune   rune                     // if set, multiply the space advance by the width of this rune
	Space       float64                  // how much advance to use for a space
	LineRune    rune                     // if set, multiply the line advance by the height of this rune
	Line        float64                  // how much to advance the y coord to start a new line
}

// FontGlyph describes paths of a single letter from a font.
type FontGlyph struct {
	Width, Height float64
	Advance       float64
	Paths         *Paths
}

// Font describes a typeface made up of paths.
type Font struct {
	LineAdvance float64
	Glyph       map[rune]*FontGlyph
}

// NewFont creates a font from the given SVG file.
// It's not an official SVG font; it's an SVG file
// with glyphs named groups or paths (named using the ID).
func NewFont(svg io.Reader, fc *FontConfig) (*Font, error) {
	var ids []string
	for _, g := range fc.Glyph {
		ids = append(ids, g.ID)
	}
	paths, err := IDsFromSVG(svg, ids)
	if err != nil {
		return nil, err
	}
	f := &Font{
		Glyph: map[rune]*FontGlyph{},
	}
	for r, g := range fc.Glyph {
		ps := paths[g.ID]
		ps.TightenBounds()
		ps.Translate(Vec2{-g.Dot[0], -g.Dot[1]})
		if len(ps.P) == 0 {
			return nil, fmt.Errorf("no paths found for glyph %q", g.ID)
		}
		w := ps.Bounds.Max[0]
		h := -ps.Bounds.Min[1]
		f.Glyph[r] = &FontGlyph{
			Width:  w,
			Height: h,
			Paths:  ps,
		}
	}
	line := 1.0
	if fc.LineRune != 0 {
		if f.Glyph[fc.LineRune] == nil {
			return nil, fmt.Errorf("line rune %c not found", fc.LineRune)
		}
		line = f.Glyph[fc.LineRune].Height
	}
	line *= fc.Line
	f.LineAdvance = line

	advance := 1.0
	if fc.AdvanceRune != 0 {
		if f.Glyph[fc.AdvanceRune] == nil {
			return nil, fmt.Errorf("advance rune %c not found", fc.AdvanceRune)
		}
		advance = f.Glyph[fc.AdvanceRune].Width
	}
	advance *= fc.Advance
	for r := range fc.Glyph {
		f.Glyph[r].Advance = f.Glyph[r].Width + advance + fc.Glyph[r].DeltaAdvance
	}
	space := 1.0
	if fc.SpaceRune != 0 {
		if f.Glyph[fc.SpaceRune] == nil {
			return nil, fmt.Errorf("space rune %c not found", fc.SpaceRune)
		}
		space = f.Glyph[fc.SpaceRune].Width
	}
	space *= fc.Space

	f.Glyph[' '] = &FontGlyph{
		Width:   0,
		Advance: space,
		Paths:   &Paths{},
	}
	return f, nil
}

type PositionedGlyph struct {
	Pos   Vec2
	Scale float64
	G     *FontGlyph
}

func LayoutText(f *Font, s string, scale float64, w float64) ([]PositionedGlyph, error) {
	var point Vec2
	var pgs []PositionedGlyph
	line := 0 // glyphs output on current line

	var words [][]rune
	data := []byte(s)
	start := 0
	chrtype := func(r rune) rune {
		if r == '\n' {
			return '\n'
		}
		if unicode.IsSpace(r) {
			return ' '
		}
		return 'a'
	}
	for start < len(data) {
		// skip spaces, except \n
		for width := 0; start < len(data); start += width {
			var r rune
			r, width = utf8.DecodeRune(data[start:])
			if r == '\n' || !unicode.IsSpace(r) {
				break
			}
		}
		// read either a sequence of \n, or a sequence of any other character
		var word []rune
		for width := 0; start < len(data); start += width {
			var r rune
			r, width = utf8.DecodeRune(data[start:])
			if len(word) > 0 && chrtype(r) != chrtype(word[0]) {
				break
			}
			word = append(word, r)
		}
		if len(word) > 0 {
			words = append(words, word)
		}
	}

	newline := func() {
		line = 0
		point[0] = 0
		point[1] += f.LineAdvance * scale
	}

	for _, word := range words {
		if line > 0 {
			point[0] += f.Glyph[' '].Advance * scale
		}
		wl := 0.0
		for i, r := range word {
			if r == '\n' {
				continue
			}
			if i+1 == len(word) {
				wl += f.Glyph[r].Width * scale
			} else {
				wl += f.Glyph[r].Advance * scale
			}
		}
		if line > 0 && point[0]+wl > w {
			newline()
		}
		for _, r := range word {
			if r == '\n' {
				newline()
				continue
			}
			g, ok := f.Glyph[r]
			if !ok {
				return nil, fmt.Errorf("no glyph for rune %c", r)
			}
			pgs = append(pgs, PositionedGlyph{Pos: point, Scale: scale, G: g})
			point[0] += g.Advance * scale
			line++
		}
	}
	return pgs, nil
}

func (f *Font) ScaleFromRuneHeight(r rune, height float64) (float64, error) {
	g, ok := f.Glyph[r]
	if !ok {
		return 0, fmt.Errorf("can't find rune %c", r)
	}
	if g.Height == 0 {
		return 0, fmt.Errorf("rune %c has 0 height")
	}
	return height / g.Height, nil
}

func (g *FontGlyph) TransformMatrixCopy(m *svgXform) []Path {
	ps := make([]Path, 0, len(g.Paths.P))
	for _, p := range g.Paths.P {
		pc := make([]Vec2, len(p.V))
		for i, v := range p.V {
			pc[i] = m.Apply(v)
		}
		ps = append(ps, Path{V: pc})
	}
	return ps
}

func GlyphsToPaths(offset Vec2, pgs []PositionedGlyph) *Paths {
	ps := &Paths{}
	for _, pg := range pgs {
		m := svgXform{M: [3][3]float64{
			{pg.Scale, 0, pg.Pos[0] + offset[0]},
			{0, pg.Scale, pg.Pos[1] + offset[1]},
			{0, 0, 1},
		}}
		ps.P = append(ps.P, pg.G.TransformMatrixCopy(&m)...)
	}
	ps.TightenBounds()
	return ps
}
