#!/usr/bin/env python3
"""Cut MPLUS1-ja-full.woff2, which pages in Japanese are set in, from M PLUS 1:
its kana, its punctuation, the kanji of JIS X 0208 level 1, nearly 3000 of them,
and the few the site needs beyond those, so that a README can change without the
font having to.

    python3 assets/fonts/subset.py 'MPLUS1[wght].ttf'

The source is ofl/mplus1 in https://github.com/google/fonts. Needs fonttools and
brotli. MPLUS1-ja.woff2 is the other cut, with only the few kanji that pages in
English show, so that those pages stay light.
"""
import sys
from pathlib import Path

from fontTools import subset
from fontTools.ttLib import TTFont

OUT = Path(__file__).resolve().parent / "MPLUS1-ja-full.woff2"
CJK_SYMBOLS_AND_KANA = range(0x3000, 0x3100)
FULLWIDTH_FORMS = range(0xFF01, 0xFF5F)
# Level 1 is the rows 16 to 47 of JIS X 0208, 0xB0 to 0xCF in EUC-JP.
LEVEL_1 = (bytes((row, cell)).decode("euc_jp", "ignore") for row in range(0xB0, 0xD0) for cell in range(0xA1, 0xFF))
# In level 2: the plane, the wood, and the 曖 of 曖昧.
BEYOND = "鉋檜曖"


def main():
    options = subset.Options(hinting=False, notdef_outline=True, flavor="woff2")
    options.layout_features = ["kern", "ccmp", "locl", "mark", "mkmk"]
    subsetter = subset.Subsetter(options)
    subsetter.populate(unicodes=[*CJK_SYMBOLS_AND_KANA, *FULLWIDTH_FORMS], text="".join(LEVEL_1) + BEYOND)
    font = TTFont(sys.argv[1])
    subsetter.subset(font)
    font.flavor = "woff2"
    font.save(OUT)
    print(f"{OUT.name}: {len(font.getBestCmap())} characters, {OUT.stat().st_size // 1024} KB")


if __name__ == "__main__":
    main()
