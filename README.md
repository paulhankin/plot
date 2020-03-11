# *Plot* -- for gcode-based xy-plotters

This repository contains code for generating commands for an
xy-plotter.

The simplest way to use this code is to run the `svgtogcode`
command, which parses an svg file, and writes gcode, configured
by flags.

For example:

    svgtocode -in drawing.svg -size 270,180 -paper 297,210 -center -out out.gcode

This parses the file `drawing.svg`, resizes it to 270mm by 180mm,
clips all lines outside the image, reorders the paths to reduce
pen movement, and centers it on an A4 page of size 297mm by 210mm.
Finally, it writes the output file `out.gcode` which can be sent
to an xy-plotter which understands gcode.

Note that this code understands and parses only a small part of
the SVG standard.

The `paths` package contains code for loading and saving SVG
files, resizing, clipping, and sorting paths.

The `gcode` package contains code for writing gcode files.
