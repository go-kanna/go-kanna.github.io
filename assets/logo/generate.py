#!/usr/bin/env python3
"""Generate the go-kanna mark.

The mark is a kanna -- a Japanese hand plane -- seen from the side: a wooden
block (the dai) with the blade set through it, its edge on the sole and its head
standing proud of the top, the cap iron lying along its face. The user pulls a
kanna toward themselves, so the blade leans back toward the right, where they
stand.

Run with no arguments to rewrite every SVG in this directory:

    python3 generate.py

PNGs are not produced here -- rasterizing needs a renderer this script should
not depend on. See README.md for the one-liner.
"""
import math
import os

# Geometry. All values are in the 256x256 design canvas, before the mark is
# fitted inside the padding.
CANVAS = 256
SOLE, TOP = 184, 124  # the bottom and top of the dai
FRONT, REAR = 30, 226  # its ends; the user stands at the rear
CHAMFER = 13  # the top edges of the dai are taken off
FOOT = 4  # and the bottom ones eased
ANGLE = 47  # degrees between the blade and the sole; a real kanna beds at ~38
EDGE_X = 96  # where the cutting edge meets the sole
LENGTH = 150  # the blade, from the sole to its head
W_EDGE, W_HEAD = 15, 27  # the blade thickens toward its head
HEAD_BEVEL = 6  # the rear corner of the head is struck off
CAP = 7  # thickness of the cap iron
CAP_FROM, CAP_SHORT = 30, 22  # where it starts up the blade, and how far short of the head it stops
GAP = 5  # the knockout around the blade in the one-colour mark
PAD = 34  # keeps the mark clear of the circle GitHub crops avatars to
CORNER = 56  # radius of the rounded variants

INK = "#0B1A21"
BLADE = "#00ADD8"
WHITE = "#FFFFFF"
STEEL = "#7F98A3"

U = (math.cos(math.radians(ANGLE)), -math.sin(math.radians(ANGLE)))  # up the blade
N = (math.sin(math.radians(ANGLE)), math.cos(math.radians(ANGLE)))  # across it, toward the rear


def along(p, d, t):
    return (p[0] + d[0] * t, p[1] + d[1] * t)


def dai():
    return [
        (FRONT + FOOT, SOLE), (REAR - FOOT, SOLE), (REAR, SOLE - FOOT),
        (REAR, TOP + CHAMFER), (REAR - CHAMFER, TOP),
        (FRONT + CHAMFER, TOP), (FRONT, TOP + CHAMFER), (FRONT, SOLE - FOOT),
    ]


def blade():
    """The blade as a wedge: its front face (the ura) runs straight from the
    edge, its back widens toward a head square to its length."""
    edge = (EDGE_X, SOLE)
    head = along(edge, U, LENGTH)
    heel = (EDGE_X + W_EDGE / math.sin(math.radians(ANGLE)), SOLE)
    back = along(head, N, W_HEAD)
    return [
        edge, heel,
        along(back, U, -HEAD_BEVEL),
        along(head, N, W_HEAD - HEAD_BEVEL),
        head,
    ]


def cap_iron():
    """The cap iron, lying on the front face of the blade below its head."""
    start = along((EDGE_X, SOLE), U, CAP_FROM)
    end = along((EDGE_X, SOLE), U, LENGTH - CAP_SHORT)
    return [start, end, along(end, N, -CAP), along(start, N, -CAP)]


def fit(shapes):
    """Scale and centre the mark inside the padded canvas."""
    xs = [x for s in shapes for x, _ in s]
    ys = [y for s in shapes for _, y in s]
    avail = CANVAS - 2 * PAD
    s = min(avail / (max(xs) - min(xs)), avail / (max(ys) - min(ys)))
    return (s,
            CANVAS / 2 - s * (min(xs) + max(xs)) / 2,
            CANVAS / 2 - s * (min(ys) + max(ys)) / 2)


def points(ps):
    return " ".join("%.1f,%.1f" % p for p in ps)


def svg(size, ground, body, steel, corner):
    shapes = {"dai": dai(), "blade": blade(), "cap": cap_iron()}
    s, tx, ty = fit(shapes.values())
    rect = ""
    if ground:
        rect = '<rect width="%d" height="%d" rx="%d" fill="%s"/>' % (CANVAS, CANVAS, corner, ground)

    if steel:
        mark = (
            '<polygon points="%s" fill="%s"/>' % (points(shapes["dai"]), body)
            + '<polygon points="%s" fill="%s"/>' % (points(shapes["blade"]), steel)
            + '<polygon points="%s" fill="%s"/>' % (points(shapes["cap"]), STEEL)
        )
    else:
        # One colour: the blade is told from the dai by a gap cut around it.
        mark = (
            '<mask id="blade-gap" maskUnits="userSpaceOnUse" x="0" y="0" width="%d" height="%d">'
            '<rect width="%d" height="%d" fill="white"/>'
            '<polygon points="%s" fill="black" stroke="black" stroke-width="%d" stroke-linejoin="round"/>'
            '</mask>' % (CANVAS, CANVAS, CANVAS, CANVAS, points(shapes["blade"]), 2 * GAP)
            + '<polygon points="%s" fill="%s" mask="url(#blade-gap)"/>' % (points(shapes["dai"]), body)
            + '<polygon points="%s" fill="%s"/>' % (points(shapes["blade"]), body)
        )

    return (
        '<svg xmlns="http://www.w3.org/2000/svg" width="%d" height="%d" viewBox="0 0 %d %d">%s'
        '<g transform="translate(%.2f %.2f) scale(%.4f)">%s</g></svg>\n'
        % (size, size, CANVAS, CANVAS, rect, tx, ty, s, mark)
    )


# (filename, ground, body, blade, size, corner radius)
#
# The avatars carry no corner radius: GitHub rounds an org avatar itself, and a
# radius baked into the upload leaves a sliver of the ground colour inside
# GitHub's own crop.
OUTPUTS = [
    ("avatar-blade", BLADE, INK, WHITE, 1000, 0),
    ("avatar-ink", INK, BLADE, WHITE, 1000, 0),
    ("avatar-white", WHITE, INK, BLADE, 1000, 0),
    ("mark-blade-rounded", BLADE, INK, WHITE, CANVAS, CORNER),
    ("mark-ink-rounded", INK, BLADE, WHITE, CANVAS, CORNER),
    ("mark-transparent", None, "currentColor", None, CANVAS, 0),
]


def main():
    here = os.path.dirname(os.path.abspath(__file__))
    for name, ground, body, steel, size, corner in OUTPUTS:
        path = os.path.join(here, name + ".svg")
        with open(path, "w") as f:
            f.write(svg(size, ground, body, steel, corner))
        print("wrote", name + ".svg")


if __name__ == "__main__":
    main()
