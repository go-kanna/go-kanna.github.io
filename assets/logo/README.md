# Logo

## The mark

The mark is a *kanna*, a Japanese hand plane, seen from the side: the wooden block (the *dai*), the blade set through it
with its edge on the sole and its head standing proud of the top, and the cap iron lying along its face. A kanna is
pulled, not pushed, so the blade leans back toward the user.

It is drawn as three flat shapes and nothing finer, so that it still reads as a plane at 20px: a dark slab, and a light
bar cutting through it at an angle. The blade beds at 47° rather than the ~38° of a real kanna, which keeps the head
inside the square without flattening the slab. The cap iron is the only detail below that size, and losing it costs
nothing.

There is no wood brown anywhere: drawn in Go's cyan, what the blade works is not timber but Go itself.

## Files

| File                     | Use                                                                                         |
|--------------------------|---------------------------------------------------------------------------------------------|
| `avatar-blade.png`       | **Org avatar.** 1000×1000, square corners                                                   |
| `avatar-ink.png`         | Same, dark ground                                                                           |
| `avatar-white.png`       | Same, white ground                                                                          |
| `mark-blade-rounded.svg` | Rounded, for READMEs and docs                                                               |
| `mark-ink-rounded.svg`   | Same, dark ground                                                                           |
| `mark-transparent.svg`   | No ground and one color, `currentColor`; a gap cut around the blade tells it from the block |
| `generate.py`            | Regenerates every SVG above                                                                 |

Prefer the blade ground for anything that lands on GitHub. GitHub's dark theme sits at `#0d1117`, close enough to the
ink ground that the avatar's edge disappears into the page and the plane is left floating. The cyan ground holds its edge
in both themes.

## Palette

|       | Hex       |                                                             |
|-------|-----------|-------------------------------------------------------------|
| ink   | `#0B1A21` | the block on cyan and white; the ground of the dark variant |
| blade | `#00ADD8` | Go cyan; the ground, the block on ink, the blade on white   |
| white | `#FFFFFF` | the blade on cyan and ink                                   |
| steel | `#7F98A3` | the cap iron                                                |
| paper | `#F2F6F7` | cool off-white, for surrounding surfaces                    |

## Regenerating

```sh
python3 generate.py
```

Geometry lives in the constants at the top of the script — the block, the bedding angle, the blade's taper, and the
padding that keeps the mark inside GitHub's circular crop. Nothing else needs editing.

Rasterizing is left out of the script so it stays dependency-free. With `rsvg-convert`:

```sh
for n in avatar-blade avatar-ink avatar-white; do
  rsvg-convert -w 1000 -h 1000 "$n.svg" -o "$n.png"
done
```

Or with headless Chrome, which needs no extra install on macOS:

```sh
for n in avatar-blade avatar-ink avatar-white; do
  "/Applications/Google Chrome.app/Contents/MacOS/Google Chrome" \
    --headless --disable-gpu --hide-scrollbars \
    --window-size=1000,1000 --screenshot="$n.png" "file://$PWD/$n.svg"
done
```

## Why the avatars have square corners

GitHub rounds an org avatar itself. Uploading an image with the radius already baked in leaves a sliver of the ground
color outside the curve but inside GitHub's own crop, which reads as a dirty edge. The avatars are therefore square to
the edge; the rounded variants exist only for places that do no cropping of their own.
