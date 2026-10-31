#!/usr/bin/env python3
"""Fail when a translation has fallen behind what it translates. A translation
keeps the code as it is and has the same headings, so each README.ja.md is held
to its README in both, and ja/index.html to index.html in its code.

    python3 pages/check_translations.py

Needs the gh CLI, logged in. The Pages workflow runs this on every push to main
and once a day.
"""
import re
import sys

from render import LANGS, ORG, ROOT, TOOLS, gh

REPOS = [repo for repo, _ in TOOLS] + ["go-kanna.github.io", "internal", "showcase"]
FENCE = re.compile(r"^[ \t]*```[^\n]*\n.*?^[ \t]*```[ \t]*$", re.M | re.S)
HEADING = re.compile(r"^(#{1,6}) ", re.M)
PRE = re.compile(r"<pre[^>]*>(.*?)</pre>", re.S)


def fetch(repo, name):
    return gh(f"repos/{ORG}/{repo}/contents/{name}", "-H", "Accept: application/vnd.github.raw+json")


def compare_readme(repo):
    english, japanese = (fetch(repo, LANGS[lang]["readme"]) for lang in ("en", "ja"))
    if FENCE.findall(english) != FENCE.findall(japanese):
        yield f"{repo}: the code blocks of README.ja.md differ from those of README.md"
    prose = [FENCE.sub("", text) for text in (english, japanese)]
    if HEADING.findall(prose[0]) != HEADING.findall(prose[1]):
        yield f"{repo}: the headings of README.ja.md differ from those of README.md"


def compare_landing():
    pages = [(ROOT / name).read_text() for name in ("index.html", "ja/index.html")]
    if PRE.findall(pages[0]) != PRE.findall(pages[1]):
        yield "kanna: the code of ja/index.html differs from that of index.html"


def main():
    problems = [*compare_landing()]
    for repo in REPOS:
        problems += compare_readme(repo)
    for problem in problems:
        print(problem, file=sys.stderr)
    sys.exit(1 if problems else 0)


if __name__ == "__main__":
    main()
