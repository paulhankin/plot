// Binary svgtogcode converts an input svg file into gcode for
// an xy plotter.
// Apart from the file format conversion, it can scale the image,
// simplify paths, and sorts them to reduce pen movement.
//
// An example use is:
//   svgtocode -in drawing.svg -size 270,180 -paper 297,210 -center -penup 35 -out out.gcode -simplify 0.1
// Vector arguments, like -size and -paper take a pair of comma-separated values (no spaces).
// If the -out <file> ends in .svg, the output is in svg format rather than gcode format.
// All distance measurements are in millimeters.
package main

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/paulhankin/plot/cmd/svgtogcode/svgtogcode"
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

var config svgtogcode.Config

func init() {
	flag.StringVar(&config.In, "in", "", "svg input file")
	flag.StringVar(&config.Out, "out", "out.gcode", "gcode or svg output file")
	flag.Var((*flagSizeValue)(&config.Delta), "offset", "displacement x,y of image origin from pen origin (mm)")
	flag.Var((*flagSizeValue)(&config.Size), "size", "target size x,y of image (mm)")
	flag.Var((*flagSizeValue)(&config.PaperSize), "paper", "target size x,y of paper (mm)")
	flag.BoolVar(&config.Center, "center", false, "if set, center image on paper")
	flag.IntVar(&config.PenUp, "penup", 40, "how much to lift pen when moving")
	flag.IntVar(&config.FeedRate, "feed", 800, "feed rate when drawing (mm/min)")
	flag.BoolVar(&config.Split, "split", true, "allow paths to be split to reduce pen movement")
	flag.BoolVar(&config.Reverse, "reverse", true, "allow paths to be drawn backwards to reduce pen movement")
	flag.Float64Var(&config.Simplify, "simplify", 0.1, "simplify paths within this tolerance (0=disabled)")
	flag.Float64Var(&config.RotateDegrees, "rotate", 0, "rotate input by this number of degrees about its center")
}

func usageMessage() {
	var w = func(f string, args ...interface{}) {
		fmt.Fprintf(flag.CommandLine.Output(), f, args...)
	}
	w("%s converts an input svg file into gcode for an xy plotter.\n", os.Args[0])
	w("Apart from the file format conversion, it can scale the image,\n")
	w("simplify paths, and sorts them to reduce pen movement.\n\n")
	w("An example use is:\n\n")
	w("    svgtocode -in drawing.svg -size 270,180 -paper 297,210 -center -penup 35 -out out.gcode -simplify 0.1\n\n")
	w("Vector arguments, like -size and -paper take a pair of comma-separated values (no spaces).\n")
	w("If the -out <file> ends in .svg, the output is in svg format rather than gcode format.\n")
	w("All distance measurements are in millimeters.\n\n")
	w("Usage:\n")
	flag.PrintDefaults()
}

func main() {
	flag.Usage = usageMessage
	flag.Parse()
	fail := func(s string, args ...interface{}) {
		fmt.Fprintf(os.Stderr, s+"\n", args...)
		os.Exit(2)

	}
	if config.In == "" {
		fail("must specify -in <svg file>")
	}

	if err := svgtogcode.Convert(&config); err != nil {
		fail("%v", err)
	}
}
