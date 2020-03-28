package paths

import (
	"bytes"
	"reflect"
	"strings"
	"testing"
)

// A simple test svg that contains paths and groups that have
// transforms applied to them.
var testSVG = `
<svg width="2000" height="1000">
   <path d="M 123, 456 321, 654"/>
   <g transform="translate(200, 100) scale(2)" stroke="black" fill="none">
	   <path d="M100,50 300, 200"/>
	   <g transform="translate(50,50)">
		   <path d="M 50, 50 250, 50 150, 100"/>
	   </g>
   </g>
</svg>`

func TestSVG(t *testing.T) {
	got, err := FromSVG(strings.NewReader(testSVG))
	if err != nil {
		t.Fatalf("failed to parse svg: %v", err)
	}
	want := &Paths{
		Bounds: Bounds{Max: Vec2{2000, 1000}},
		P: []Path{
			Path{V: []Vec2{{123, 456}, {321, 654}}},
			Path{V: []Vec2{{400, 200}, {800, 500}}},
			Path{V: []Vec2{{400, 300}, {800, 300}, {600, 400}}},
		},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("svg parse. Got:\n%v\nWant:\n%v\n", got, want)
	}
}

// TestSVGRoundTrip parses paths out of an svg, writes them back
// to a new svg file, parses the paths out of that, and then checks
// that the paths (or bounds) don't change.
func TestSVGRoundTrip(t *testing.T) {
	got, err := FromSVG(strings.NewReader(testSVG))
	if err != nil {
		t.Fatalf("failed to parse svg: %v", err)
	}
	if len(got.P) == 0 {
		t.Fatalf("expected some paths")
	}
	var bb bytes.Buffer
	if err := got.SVG(&bb); err != nil {
		t.Fatalf("failed to write back svg: %v", err)
	}
	got2, err := FromSVG(&bb)
	if err != nil {
		t.Fatalf("failed to re-parse svg: %v", err)
	}
	if !reflect.DeepEqual(got, got2) {
		t.Errorf("svg round-trip not identity. Started with:\n%v\nGot:\n%v", got, got2)
	}

}
