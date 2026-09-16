#!/usr/bin/env python3
# drift-scan -- look for register drift across a whole corpus of session transcripts.
#
# For each session it takes the assistant's prose in order, splits it into three
# equal parts by turn index, and compares the last third with the first. A
# session whose emphasis rises across its own length is the signal; a session
# that is uniformly emphatic is a different thing and is reported separately.
#
#   drift-scan.py [minimum-bytes]
#
# Prints one row per session. Reads transcripts, prints no content.

import json
import re
import sys
import glob
import os

BOLD = re.compile(r"\*\*[^*\n]+\*\*")
CAPS = re.compile(r"\b[A-Z]{3,}\b")
ACRONYMS = {"API","URL","HTTP","HTTPS","JSON","YAML","HTML","CSS","SQL","CPU","GPU","RAM",
            "SSH","TLS","DNS","TCP","UDP","PID","UID","GID","CLI","GUI","SDK","IDE","VM",
            "OS","IO","UTC","ISO","RFC","MUST","SHALL","SHOULD","MAY","NOT","TODO","FIXME",
            "NOTE","README","GET","POST","PUT","DELETE","HEAD","OK","ID","IDS","K8S","AWS",
            "EOF","STDIN","STDOUT","STDERR","TUI","IPC","ANSI","TERM","LOCAL","CSV","PNG"}
SELF_FAULT = re.compile(r"\b(i|my)\b[^.\n]{0,40}\b(wrong|failed|fail|error|mistake|fault|"
                        r"should have|shouldn't have|broke|broken|sorry|apolog)", re.I)


# strip code so pasted source does not count as prose
def prose(t):
    t = re.sub(r"```.*?```", " ", t, flags=re.S)
    return re.sub(r"`[^`\n]+`", " ", t)


# emphasis and self-fault densities for one slice of a session
def score(chunks):
    t = prose("\n".join(chunks))
    n = len(re.findall(r"[A-Za-z']+", t)) or 1
    caps = [c for c in CAPS.findall(t) if c not in ACRONYMS]
    return (100.0 * len(BOLD.findall(t)) / n,
            100.0 * len(caps) / n,
            len(SELF_FAULT.findall(t)),
            n)


# ordered assistant prose blocks for one transcript
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
            msg = r.get("message") or {}
            if msg.get("model") == "<synthetic>":
                continue
            for b in msg.get("content") or []:
                if isinstance(b, dict) and b.get("type") == "text" and b.get("text"):
                    out.append(b["text"])
    return out


def main():
    floor = int(sys.argv[1]) if len(sys.argv) > 1 else 200000
    rows = []
    for p in glob.glob(os.path.expanduser("~/.claude/projects/*/*.jsonl")):
        if os.path.getsize(p) < floor:
            continue
        bs = blocks(p)
        if len(bs) < 30:
            continue
        k = len(bs) // 3
        b1, c1, f1, w1 = score(bs[:k])
        b3, c3, f3, w3 = score(bs[-k:])
        name = "-"
        with open(p, errors="replace") as fh:
            for line in fh:
                m = re.search(r'"agentName":"([^"]*)"', line)
                if m:
                    name = m.group(1)
        rows.append({
            "name": name, "id": os.path.basename(p)[:8], "blocks": len(bs),
            "b1": b1, "b3": b3, "c1": c1, "c3": c3,
            "f1": f1, "f3": f3,
            "rise": (b3 + c3) - (b1 + c1),
        })
    rows.sort(key=lambda r: -r["rise"])
    print(f"{'name':<20}{'id':<10}{'blks':>5}{'bold1':>7}{'bold3':>7}{'caps1':>7}{'caps3':>7}"
          f"{'flt1':>6}{'flt3':>6}{'rise':>7}")
    for r in rows:
        print(f"{r['name'][:19]:<20}{r['id']:<10}{r['blocks']:>5}{r['b1']:>7.2f}{r['b3']:>7.2f}"
              f"{r['c1']:>7.2f}{r['c3']:>7.2f}{r['f1']:>6}{r['f3']:>6}{r['rise']:>7.2f}")
    return 0


if __name__ == "__main__":
    sys.exit(main())
