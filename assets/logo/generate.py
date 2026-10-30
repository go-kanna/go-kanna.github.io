#!/usr/bin/env python3
"""Generate the go-kanna mark.

The mark is a shaving -- kanna-kuzu, what a Japanese hand plane lifts off a
board: one ribbon, flat where it lay on the board and curled where it came off.
It is drawn as a single stroke, a straight run and then a spiral of quarter
turns, each tighter than the one before.

Run with no arguments to rewrite every SVG in this directory:

    python3 generate.py

PNGs are not produced here -- rasterizing needs a renderer this script should
not depend on. See README.md for the one-liner.
"""
import os

# Geometry. All values are in the 256x256 design canvas, before the mark is
# fitted inside the padding.
CANVAS = 256
BOARD = 186  # the height of the flat run
FROM, TO = 30, 148  # where the run starts, and where it starts to curl
RADII = (56, 56, 40, 26, 13)  # one per quarter turn of the curl, outermost first
STROKE = 24  # the width of the ribbon
PAD = 34  # keeps the mark clear of the corners GitHub rounds avatars to
CORNER = 56  # radius of the rounded variants

KON = "#1F2C5C"
HINOKI = "#F3DCA4"
WHITE = "#FFFFFF"


def curl():
    """The path of the ribbon: the run along the board, then each quarter turn
    to the left, so that it rolls back over itself. Returns the path data and
    every point it passes through, for fitting."""
    x, y = TO, BOARD
    dx, dy = 1, 0  # heading along the board
    d = ["M %g %g H %g" % (FROM, BOARD, TO)]
    pts = [(FROM, BOARD), (x, y)]
    for r in RADII:
        lx, ly = dy, -dx  # left of the heading, on a canvas whose y grows down
        x, y = x + r * (dx + lx), y + r * (dy + ly)
        d.append("A %g %g 0 0 0 %g %g" % (r, r, x, y))
        pts.append((x, y))
        dx, dy = lx, ly
    return " ".join(d), pts


def fit(pts):
    """Scale and centre the mark, ribbon width included, inside the padded canvas."""
    half = STROKE / 2
    x0, x1 = min(x for x, _ in pts) - half, max(x for x, _ in pts) + half
    y0, y1 = min(y for _, y in pts) - half, max(y for _, y in pts) + half
    avail = CANVAS - 2 * PAD
    s = min(avail / (x1 - x0), avail / (y1 - y0))
    return s, CANVAS / 2 - s * (x0 + x1) / 2, CANVAS / 2 - s * (y0 + y1) / 2


def svg(size, ground, ribbon, corner):
    d, pts = curl()
    s, tx, ty = fit(pts)
    rect = ""
    if ground:
        rect = '<rect width="%d" height="%d" rx="%d" fill="%s"/>' % (CANVAS, CANVAS, corner, ground)
    return (
        '<svg xmlns="http://www.w3.org/2000/svg" width="%d" height="%d" viewBox="0 0 %d %d">%s'
        '<path transform="translate(%.2f %.2f) scale(%.4f)" d="%s" fill="none" stroke="%s" stroke-width="%d"'
        ' stroke-linecap="round" stroke-linejoin="round"/></svg>\n'
        % (size, size, CANVAS, CANVAS, rect, tx, ty, s, d, ribbon, STROKE)
    )


# (filename, ground, ribbon, size, corner radius)
#
# The avatars carry no corner radius: GitHub rounds an org avatar itself, and a
# radius baked into the upload leaves a sliver of the ground colour inside
# GitHub's own crop.
OUTPUTS = [
    ("avatar-kon", KON, HINOKI, 1000, 0),
    ("avatar-hinoki", HINOKI, KON, 1000, 0),
    ("avatar-white", WHITE, KON, 1000, 0),
    ("mark-kon-rounded", KON, HINOKI, CANVAS, CORNER),
    ("mark-hinoki-rounded", HINOKI, KON, CANVAS, CORNER),
    ("mark-transparent", None, "currentColor", CANVAS, 0),
]


def main():
    here = os.path.dirname(os.path.abspath(__file__))
    for name, ground, ribbon, size, corner in OUTPUTS:
        path = os.path.join(here, name + ".svg")
        with open(path, "w") as f:
            f.write(svg(size, ground, ribbon, corner))
        print("wrote", name + ".svg")


if __name__ == "__main__":
    main()
