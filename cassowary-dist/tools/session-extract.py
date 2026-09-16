#!/usr/bin/env python3
# session-extract -- find sessions by name or id, and split one into readable files.
#
# Splits a session into your typed turns and the assistant's prose, written to
# separate files so a transcript can be read by a person without passing
# through an agent's context. Prints counts only, never content.
#
#   session-extract.py --list              every session that ever had a name
#   session-extract.py --list albert       only names containing "albert"
#   session-extract.py albert              extract the session named albert
#   session-extract.py <session-id>        extract by id
#   session-extract.py albert <outdir>     write somewhere other than the default
#   session-extract.py --summaries albert  write out the compaction summaries only
#   session-extract.py --exchanges N albert  the last N typed turns with the
#                                            replies that followed each one
#
# Default outdir is ~/temp/session-<first 8 chars of id>/.
#
# A session can be renamed, so a name is matched against every name the session
# ever answered to, and the row reports the one it answers to now. Names are
# not unique: if more than one session matches, the matches are listed and
# nothing is extracted.

import json
import os
import re
import subprocess
import sys
from pathlib import Path

PROJECTS = Path(os.path.realpath(Path.home() / ".claude" / "projects"))
UUID = re.compile(r"^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$", re.I)
NAME_IN_LINE = re.compile(r'^([^:]+):"agentName":"([^"]*)"')


# one grep over every transcript is far cheaper than opening them in python;
# returns {path: [names in the order they were set]}
def scan_names():
    proc = subprocess.run(
        ["grep", "-RHo", '"agentName":"[^"]*"', str(PROJECTS)],
        capture_output=True, text=True,
    )
    found = {}
    for line in proc.stdout.splitlines():
        m = NAME_IN_LINE.match(line)
        if not m:
            continue
        path, name = m.group(1), m.group(2)
        found.setdefault(path, []).append(name)
    return found


# first and last timestamp in a transcript, without reading it all into memory
def span(path):
    first = last = ""
    pat = re.compile(r'"timestamp":"([^"]*)"')
    with open(path, errors="replace") as fh:
        for line in fh:
            m = pat.search(line)
            if m:
                if not first:
                    first = m.group(1)
                last = m.group(1)
    return first[:19], last[:19]


# how many turns the person actually typed, as opposed to tool traffic
def typed_count(path):
    n = 0
    with open(path, errors="replace") as fh:
        for line in fh:
            if '"promptSource":"typed"' in line:
                n += 1
    return n


# a summary row for one transcript: id, current name, aliases, span, counts
def describe(path, names):
    p = Path(path)
    first, last = span(p)
    return {
        "id": p.stem,
        "now": names[-1] if names else "-",
        "aliases": sorted({n for n in names[:-1]} - {names[-1]}) if names else [],
        "first": first,
        "last": last,
        "typed": typed_count(p),
        "records": sum(1 for _ in open(p, errors="replace")),
        "project": p.parent.name,
        "path": p,
    }


# print the listing table
def show(rows):
    for r in sorted(rows, key=lambda r: r["first"]):
        alias = f"  (was {', '.join(r['aliases'])})" if r["aliases"] else ""
        print(f"{r['now']:<18} {r['id']}  typed={r['typed']:<4} rec={r['records']:<6} "
              f"{r['first']} -> {r['last']}  {r['project']}{alias}")


# a record's content is sometimes a bare string and sometimes a block list;
# return only human-readable text blocks, never tool traffic or thinking
def text_of(message):
    content = message.get("content")
    if isinstance(content, str):
        return content
    if not isinstance(content, list):
        return ""
    return "\n".join(
        b.get("text", "") for b in content
        if isinstance(b, dict) and b.get("type") == "text" and b.get("text")
    )


# read the transcript once and bucket every record by what it is
def split(transcript):
    typed, assistant, synthetic, other = [], [], 0, 0
    with open(transcript, errors="replace") as fh:
        for line in fh:
            line = line.strip()
            if not line:
                continue
            try:
                rec = json.loads(line)
            except json.JSONDecodeError:
                other += 1
                continue
            stamp, kind = rec.get("timestamp", "?"), rec.get("type")
            msg = rec.get("message") or {}
            if kind == "user" and rec.get("promptSource") == "typed":
                body = text_of(msg)
                if body.strip():
                    typed.append((stamp, body))
            elif kind == "assistant":
                if msg.get("model") == "<synthetic>":
                    synthetic += 1
                elif text_of(msg).strip():
                    assistant.append((stamp, text_of(msg)))
            else:
                other += 1
    return typed, assistant, synthetic, other


# write one section file and report how many entries and bytes it holds
def write_section(path, entries, heading):
    with open(path, "w") as fh:
        fh.write(f"# {heading}\n\n")
        for stamp, body in entries:
            fh.write(f"## {stamp}\n\n{body.rstrip()}\n\n")
    return len(entries), path.stat().st_size


# do the extraction and print counts, never content
def extract(transcript, outdir):
    outdir.mkdir(parents=True, exist_ok=True)
    typed, assistant, synthetic, other = split(transcript)
    n_you, b_you = write_section(outdir / "your-turns.md", typed, "Typed turns")
    n_ai, b_ai = write_section(outdir / "assistant-prose.md", assistant, "Assistant prose")
    print(f"transcript   {transcript}")
    print(f"project      {transcript.parent.name}")
    print(f"your turns   {n_you}  ({b_you} bytes)  -> {outdir/'your-turns.md'}")
    print(f"assistant    {n_ai}  ({b_ai} bytes)  -> {outdir/'assistant-prose.md'}")
    print(f"synthetic    {synthetic} (skipped)")
    print(f"other        {other} records (tool traffic, meta)")
    if typed:
        print(f"first typed  {typed[0][0]}")
        print(f"last typed   {typed[-1][0]}")


# pull the compaction summaries out of a transcript. they are stored as
# user-role records flagged isCompactSummary, which is why they are invisible
# in the session itself -- they look like something the person said.
def summaries(transcript, outdir):
    outdir.mkdir(parents=True, exist_ok=True)
    found = []
    with open(transcript, errors="replace") as fh:
        for line in fh:
            if '"isCompactSummary"' not in line:
                continue
            try:
                rec = json.loads(line)
            except json.JSONDecodeError:
                continue
            if not rec.get("isCompactSummary"):
                continue
            found.append((rec.get("timestamp", "?"), text_of(rec.get("message") or {})))
    dest = outdir / "compaction-summaries.md"
    n, b = write_section(dest, found, "Compaction summaries")
    print(f"transcript   {transcript}")
    print(f"summaries    {n}  ({b} bytes)  -> {dest}")
    for stamp, body in found:
        print(f"  {stamp[:19]}  {len(body)} chars")


# pair each typed turn with the assistant prose that followed it, up to the next
# typed turn. an exchange is the unit a person remembers; a turn on its own is
# not, because the reply is what they were reacting to
def exchanges(transcript, outdir, n):
    outdir.mkdir(parents=True, exist_ok=True)
    pairs, current = [], None
    with open(transcript, errors="replace") as fh:
        for line in fh:
            line = line.strip()
            if not line:
                continue
            try:
                rec = json.loads(line)
            except json.JSONDecodeError:
                continue
            msg = rec.get("message") or {}
            stamp = rec.get("timestamp", "?")
            if rec.get("promptSource") == "typed":
                body = text_of(msg)
                if body.strip():
                    current = {"ts": stamp, "prompt": body, "replies": []}
                    pairs.append(current)
            elif rec.get("type") == "assistant" and current is not None:
                if msg.get("model") == "<synthetic>":
                    continue
                body = text_of(msg)
                if body.strip():
                    current["replies"].append((stamp, body))
    tail = pairs[-n:]
    dest = outdir / f"last{n}-exchanges.md"
    with open(dest, "w") as fh:
        fh.write(f"# Last {len(tail)} exchanges\n\n")
        for ex in tail:
            fh.write(f"## {ex['ts']}\n\n### typed\n\n{ex['prompt'].rstrip()}\n\n")
            for ts, body in ex["replies"]:
                fh.write(f"### reply {ts}\n\n{body.rstrip()}\n\n")
    print(f"transcript   {transcript}")
    print(f"exchanges    {len(pairs)} total, wrote last {len(tail)}")
    print(f"replies      {sum(len(e['replies']) for e in tail)} in those")
    print(f"written to   {dest} ({dest.stat().st_size} bytes)")
    for ex in tail:
        print(f"  {ex['ts'][:19]}  {len(ex['prompt']):>6} chars typed, "
              f"{len(ex['replies'])} replies")


# resolve an id to its transcript path
def by_id(session_id):
    hits = sorted(PROJECTS.glob(f"*/{session_id}.jsonl"))
    return hits or sorted(PROJECTS.glob(f"*/{session_id}*.jsonl"))


def main():
    args = sys.argv[1:]
    if not args:
        print(__doc__ or "usage: session-extract.py [--list] <name|id> [outdir]", file=sys.stderr)
        return 2

    if args[0] == "--list":
        pattern = args[1].lower() if len(args) > 1 else ""
        named = scan_names()
        rows = [describe(p, n) for p, n in named.items()
                if not pattern or any(pattern in x.lower() for x in n)]
        if not rows:
            print(f"no session name matches {pattern!r}", file=sys.stderr)
            return 1
        show(rows)
        return 0

    want_exchanges = 0
    if args[0] == "--exchanges":
        want_exchanges = int(args[1])
        args = args[2:]
        if not args:
            print("usage: session-extract.py --exchanges N <name|id> [outdir]", file=sys.stderr)
            return 2

    want_summaries = args[0] == "--summaries"
    if want_summaries:
        args = args[1:]
        if not args:
            print("usage: session-extract.py --summaries <name|id> [outdir]", file=sys.stderr)
            return 2

    target = args[0]
    outdir = Path(args[1]) if len(args) > 1 else None

    # a bare hex prefix is an id fragment, not a name; try the filesystem first
    if UUID.match(target) or re.fullmatch(r"[0-9a-f]{6,}", target, re.I):
        hits = by_id(target)
    else:
        named = scan_names()
        hits = [Path(p) for p, n in named.items()
                if any(x.lower() == target.lower() for x in n)]

    if not hits:
        print(f"no session matches {target!r}. try --list", file=sys.stderr)
        return 1
    if len(hits) > 1:
        print(f"{len(hits)} sessions match {target!r}; extract one by id:", file=sys.stderr)
        named = scan_names()
        show([describe(str(h), named.get(str(h), [])) for h in hits])
        return 1

    transcript = hits[0]
    if outdir is None:
        outdir = Path.home() / "temp" / f"session-{transcript.stem[:8]}"
    if want_exchanges:
        exchanges(transcript, outdir, want_exchanges)
    elif want_summaries:
        summaries(transcript, outdir)
    else:
        extract(transcript, outdir)
    return 0


if __name__ == "__main__":
    sys.exit(main())
