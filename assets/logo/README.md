# Logo

## The mark

The mark is a shaving: *kanna-kuzu*, what a Japanese hand plane lifts off a board. One ribbon, flat where it lay on the
board and curled where it came off.

It is one stroke: a straight run, then a spiral of quarter turns, each tighter than the one before. Nothing in it is
finer than the ribbon itself, so it reads the same at 16px as at 1000.

The ribbon is the color of planed hinoki, and on the site that color means the same thing: code kanna wrote. What you
write sits on indigo, what a generator writes sits on wood.

## Files

| File                      | Use                                       |
|---------------------------|-------------------------------------------|
| `avatar-kon.png`          | **Org avatar.** 1000×1000, square corners |
| `avatar-hinoki.png`       | Same, wood ground                         |
| `avatar-white.png`        | Same, white ground                        |
| `mark-kon-rounded.svg`    | Rounded, for READMEs and docs             |
| `mark-hinoki-rounded.svg` | Same, wood ground                         |
| `mark-transparent.svg`    | No ground and one color, `currentColor`   |
| `generate.py`             | Regenerates every SVG above               |

Prefer the indigo ground for anything that lands on GitHub: it holds its edge against GitHub's dark theme (`#0d1117`)
and its light one alike. Hinoki on white does not, so the white variant draws the ribbon in indigo.

## Palette

The mark uses the first three; the site uses all of them.

|         | Hex       |                                            |
|---------|-----------|--------------------------------------------|
| kon     | `#1F2C5C` | indigo; the ground, and the code you write |
| hinoki  | `#F3DCA4` | planed cypress; the ribbon                 |
| white   | `#FFFFFF` |                                            |
| kachi   | `#141B3D` | the darkest indigo; text                   |
| ruri    | `#3355D1` | lapis; links                               |
| steel   | `#8C94B0` | what stays quiet                           |
| shaving | `#FAF0D7` | the ground of the code kanna writes        |
| grain   | `#8A5A10` | text on shaving                            |
| mist    | `#F3F4F8` | cool off-white, for surrounding surfaces   |

## Regenerating

```sh
python3 generate.py
```

Geometry lives in the constants at the top of the script: the run, the radius of each quarter turn, the width of the
ribbon, and the padding that keeps the mark clear of the corners GitHub rounds. Nothing else needs editing.

Rasterizing is left out of the script so it stays dependency-free. With `rsvg-convert`:

```sh
for n in avatar-kon avatar-hinoki avatar-white; do
  rsvg-convert -w 1000 -h 1000 "$n.svg" -o "$n.png"
done
```

Or with headless Chrome, which needs no extra install on macOS:

```sh
for n in avatar-kon avatar-hinoki avatar-white; do
  "/Applications/Google Chrome.app/Contents/MacOS/Google Chrome" \
    --headless --disable-gpu --hide-scrollbars \
    --window-size=1000,1000 --screenshot="$n.png" "file://$PWD/$n.svg"
done
```

## Why the avatars have square corners

GitHub rounds an org avatar itself. Uploading an image with the radius already baked in leaves a sliver of the ground
color outside the curve but inside GitHub's own crop, which reads as a dirty edge. The avatars are therefore square to
the edge; the rounded variants exist only for places that do no cropping of their own.
