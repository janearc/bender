#!/usr/bin/env python3
# comment-drift -- measure the register of a repository's comments over its history.
#
# Takes evenly spaced snapshots of a repo, extracts the comments from its source
# at each one, and reports the same densities used elsewhere. Read-only: it
# exports each commit to a temporary directory rather than touching the repo.
#
#   comment-drift.py /abs/path/to/repo [snapshots]

import re
import subprocess
import sys
import tempfile
import os
from pathlib import Path

LINE = re.compile(r"(?m)^\s*//\s?(.*)$")
BLOCK = re.compile(r"/\*(.*?)\*/", re.S)
CAPS = re.compile(r"\b[A-Z]{3,}\b")
AC = {"API","URL","HTTP","HTTPS","JSON","YAML","HTML","CSS","SQL","CLI","VM","OS","UTC","ISO",
      "RFC","MUST","SHALL","SHOULD","MAY","NOT","TODO","NOTE","README","GET","POST","OK","ID",
      "EOF","STDIN","STDOUT","STDERR","TUI","IPC","ANSI","TERM","LOCAL","PID","SSH","DNS","TCP"}
JUDGE = set("""critical crucial essential vital severe catastrophic disastrous dangerous alarming
urgent dire grave completely entirely utterly absolutely totally wholly impossible unacceptable
worthless useless broken wrong failure failed fails disease rot deeply profoundly fundamentally
pervasive dumb stupid awful terrible ugly bad nonsense garbage""".split())


# every commit id and date, oldest first
def history(repo):
    out = subprocess.run(["git", "-C", repo, "log", "--reverse", "--format=%H %ad",
                          "--date=short"], capture_output=True, text=True).stdout
    return [l.split(None, 1) for l in out.splitlines() if l.strip()]


# comments from every Go file in one exported tree
def comments_at(repo, commit, workdir):
    subprocess.run(f"git -C {repo} archive {commit} | tar -x -C {workdir}",
                   shell=True, capture_output=True)
    parts = []
    for p in Path(workdir).rglob("*.go"):
        try:
            s = p.read_text(errors="replace")
        except OSError:
            continue
        parts.append("\n".join(LINE.findall(s) + BLOCK.findall(s)))
    return "\n".join(parts)


# the densities for one snapshot
def measure(text):
    words = re.findall(r"[A-Za-z']+", text)
    n = len(words) or 1
    caps = [c for c in CAPS.findall(text) if c not in AC]
    judge = sum(1 for w in words if w.lower() in JUDGE)
    return n, 100.0 * len(caps) / n, 100.0 * judge / n


def main():
    repo = sys.argv[1]
    k = int(sys.argv[2]) if len(sys.argv) > 2 else 8
    h = history(repo)
    if not h:
        print("no history", file=sys.stderr)
        return 1
    step = max(1, len(h) // k)
    picks = h[::step][:k] + [h[-1]]
    print(f"{os.path.basename(repo)}: {len(h)} commits, {len(picks)} snapshots")
    print(f"  {'date':<12}{'commit':<10}{'words':>8}{'caps/100':>10}{'judge/100':>11}")
    seen = set()
    for commit, date in picks:
        if commit in seen:
            continue
        seen.add(commit)
        with tempfile.TemporaryDirectory() as wd:
            text = comments_at(repo, commit, wd)
        n, caps, judge = measure(text)
        print(f"  {date:<12}{commit[:8]:<10}{n:>8}{caps:>10.2f}{judge:>11.2f}")
    return 0


if __name__ == "__main__":
    sys.exit(main())
