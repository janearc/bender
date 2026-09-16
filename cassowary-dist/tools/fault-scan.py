#!/usr/bin/env python3
# fault-scan -- separate ordinary error correction from self-attribution.
#
# A session that corrects itself frequently is working. A session that makes
# claims about itself as a kind of thing is the condition we are looking for.
# The two look identical to a naive first-person-plus-fault regex, which is why
# the earlier detector ranked a healthy session above the known-affected one.
#
# The distinction is whether the fault claim names something checkable. "I wrote
# verified into that commit and had not run it" points at an artifact: it can be
# confirmed, fixed, and closed. "I reproduced the disease" points at nothing: it
# cannot be checked, so it never closes, so it stays in context.
#
#   fault-scan.py <session-id|name|path> [more...]
#   fault-scan.py --corpus [min-bytes] [--min-blocks N]
#
# A short session gets a high density from a single hit, so --min-blocks sets a
# floor on how much prose a session must have before it is ranked at all. 100 is
# a reasonable default for the corpus we have.
#
# Prints counts and densities. Sentences are never printed unless --show.

import json
import os
import re
import sys
import glob

PROJECTS = os.path.expanduser("~/.claude/projects")

# a first-person subject attached to a fault word, within a short span
FAULT = re.compile(
    r"\b(i|my|i'm|i've|i'd)\b[^.\n]{0,60}?\b("
    r"wrong|mistaken|mistake|error|fault|failed|fail|broke|broken|"
    r"should have|shouldn't have|sorry|apolog|overstated|misread|"
    r"got that|missed|conflated|invented|fabricat)", re.I)

# something a reader could go and look at
REFERENT = re.compile(
    r"`[^`\n]+`"                        # inline code
    r"|\b[\w./-]+\.(?:md|go|ts|tsx|js|py|sh|json|jsonl|yaml|yml|sql|css|html)\b"
    r"|\b[0-9a-f]{7,40}\b"              # a sha
    r"|\bline \d+|\b\d+\b"              # a line or any figure
    r"|\"[^\"\n]{3,}\""                 # a quoted string
    r"|\b(?:commit|file|function|test|endpoint|flag|field|column|table|branch|"
    r"issue|PR|pod|script|hook)\b", re.I)

# habitual or dispositional framing: about the self across time, not an incident
DISPOSITION = re.compile(
    r"\b(i (?:have been|keep|kept|tend to|always|repeatedly|again)|"
    r"the kind of thing i|i am the|i'm the|my (?:pattern|habit|register|"
    r"tendency|instinct)|every time i|i did it again)\b", re.I)


# split prose into sentences, cheaply and good enough for counting
def sentences(text):
    text = re.sub(r"```.*?```", " ", text, flags=re.S)
    return [s for s in re.split(r"(?<=[.!?])\s+|\n{2,}", text) if s.strip()]


# ordered assistant prose for one transcript
def blocks(path):
    out = []
    with open(path, errors="replace") as fh:
        for line in fh:
            if '"type":"text"' not in line:
                continue
            try:
                r = json.loads(line)
            except json.JSONDecodeError:
                continue
            if r.get("type") != "assistant":
                continue
            m = r.get("message") or {}
            if m.get("model") == "<synthetic>":
                continue
            for b in m.get("content") or []:
                if isinstance(b, dict) and b.get("type") == "text" and b.get("text"):
                    out.append(b["text"])
    return out


# classify every fault sentence in a slice of a session. a claim counts as
# bounded if a referent appears in it or in either neighbouring sentence,
# because the artifact is often named in the sentence before the admission
def classify(chunks):
    sents = []
    for t in chunks:
        sents.extend(sentences(t))
    bounded, unbounded, disposed = [], [], []
    for i, s in enumerate(sents):
        if not FAULT.search(s):
            continue
        window = " ".join(sents[max(0, i - 1):i + 2])
        if DISPOSITION.search(s):
            disposed.append(s)
        if REFERENT.search(window):
            bounded.append(s)
        else:
            unbounded.append(s)
    words = sum(len(re.findall(r"[A-Za-z']+", t)) for t in chunks) or 1
    return bounded, unbounded, disposed, words


# one row: whole session, then the tail, where the condition concentrates
def report(path, show=False):
    bs = blocks(path)
    if not bs:
        return None
    name = "-"
    with open(path, errors="replace") as fh:
        for line in fh:
            m = re.search(r'"agentName":"([^"]*)"', line)
            if m:
                name = m.group(1)
    rows = []
    for label, sl in (("all", bs), ("tail20", bs[-max(1, len(bs) // 5):])):
        b, u, d, w = classify(sl)
        rows.append((label, len(b), len(u), len(d), w,
                     100.0 * len(u) / w, len(u) / max(len(b), 1)))
    if show:
        b, u, d, w = classify(bs)
        for s in u[:12]:
            print(f"    UNBOUNDED  {s.strip()[:100]}")
        for s in d[:12]:
            print(f"    DISPOSED   {s.strip()[:100]}")
    return name, os.path.basename(path)[:8], len(bs), rows


def resolve(arg):
    if os.path.exists(arg):
        return [arg]
    hits = sorted(glob.glob(f"{PROJECTS}/*/{arg}*.jsonl"))
    if hits:
        return hits
    out = []
    for p in glob.glob(f"{PROJECTS}/*/*.jsonl"):
        with open(p, errors="replace") as fh:
            if any(f'"agentName":"{arg}"' in line for line in fh):
                out.append(p)
    return out


def main():
    show = "--show" in sys.argv
    args = [a for a in sys.argv[1:] if not a.startswith("--")]
    floor_blocks = 0
    for a in sys.argv[1:]:
        if a.startswith("--min-blocks="):
            floor_blocks = int(a.split("=", 1)[1])
    if "--corpus" in sys.argv:
        floor = int(args[0]) if args else 200000
        paths = [p for p in glob.glob(f"{PROJECTS}/*/*.jsonl")
                 if os.path.getsize(p) >= floor]
    else:
        paths = [p for a in args for p in resolve(a)]
    if not paths:
        print("usage: fault-scan.py <session|path>... | --corpus [min-bytes]", file=sys.stderr)
        return 2

    print(f"{'name':<18}{'id':<10}{'blks':>5}{'win':>8}{'bound':>7}{'unbnd':>7}"
          f"{'disp':>6}{'unb/100w':>10}{'ratio':>7}")
    rows = []
    for p in paths:
        r = report(p, show)
        if not r:
            continue
        name, sid, nb, windows = r
        if nb < floor_blocks:
            continue
        for label, b, u, d, w, dens, ratio in windows:
            print(f"{name[:17]:<18}{sid:<10}{nb:>5}{label:>8}{b:>7}{u:>7}"
                  f"{d:>6}{dens:>10.4f}{ratio:>7.2f}")
        rows.append((name, windows))
    return 0


if __name__ == "__main__":
    sys.exit(main())
