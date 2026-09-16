#!/usr/bin/env python3
# interleave.py -- every agent's activity for a day, in one time-ordered stream.
#
# daylog.py records what a day produced: commits and issues, attributed. This
# records what a day did. A session that spent four hours reading and reached a
# conclusion leaves nothing in daylog and everything here.
#
# The transcripts are one jsonl per session under ~/.claude/projects, eighty
# megabytes each, with tool payloads inline. Nobody can read them and nothing
# merges them, so the order of work across sessions has existed only in the
# operator's head.
#
# every row carries its own referent.
#
# Each line is prefixed with the cite that addresses it, <session8>:<line>, the
# same form daylog prints and tcite resolves. So a row that looks wrong can be
# opened at full width without searching for it:
#
#     interleave.py --day 2026-09-03 | grep something
#     tcite.py 4b732342:47280 -c 20
#
# what it does not do.
#
# It does not attribute, rank or summarise. It is a record, and the reason to
# have one is that every other instrument here infers, and an inference with no
# underlying record cannot be checked.
#
# Reads transcripts. Writes only where told.

import argparse
import json
import sys
from datetime import datetime
from pathlib import Path

sys.path.insert(0, str(Path(__file__).parent))
from tcite import text_of, local_time    # one renderer, not two

PROJECTS = Path.home() / ".claude" / "projects"
_NAMES = {}


# session_name recovers what a session is called. Claude Code stores the name
# the operator gave it as a custom-title record and its own generated one as an
# ai-title, both repeated through the file; the last of each wins. An operator
# addresses a session by name, so a record that says "assistant" names the one
# thing every row already has in common and none of what distinguishes them.
def session_name(path, short):
    if short in _NAMES:
        return _NAMES[short]
    custom = ai = ""
    try:
        with path.open() as fh:
            for line in fh:
                if '"custom-title"' in line:
                    try:
                        custom = json.loads(line).get("customTitle") or custom
                    except json.JSONDecodeError:
                        pass
                elif '"ai-title"' in line:
                    try:
                        ai = json.loads(line).get("aiTitle") or ai
                    except json.JSONDecodeError:
                        pass
    except OSError:
        pass
    _NAMES[short] = custom or ai or short
    return _NAMES[short]


# unsanitise turns a project directory name back into the path it encodes.
def unsanitise(name):
    return "/" + name.strip("-").replace("-", "/")


# day_of returns the local calendar date of a record, which is what a person
# means by "today" even though every stored stamp is UTC.
def day_of(stamp):
    try:
        return (datetime.fromisoformat(stamp.replace("Z", "+00:00"))
                .astimezone().strftime("%Y-%m-%d"))
    except (ValueError, AttributeError):
        return ""


# collect walks every transcript and returns the records for one day, each
# tagged with the session and line that produced it.
def collect(day, want_thinking):
    rows = []
    if not PROJECTS.is_dir():
        return rows
    for d in PROJECTS.iterdir():
        if not d.is_dir():
            continue
        repo = unsanitise(d.name).rstrip("/").split("/")[-1]
        for f in d.glob("*.jsonl"):
            short = f.stem[:8]
            name = session_name(f, short)
            try:
                with f.open() as fh:
                    for n, line in enumerate(fh, 1):
                        if day not in line:
                            continue          # cheap reject before parsing
                        try:
                            obj = json.loads(line)
                        except json.JSONDecodeError:
                            continue
                        if obj.get("type") not in ("user", "assistant"):
                            continue
                        stamp = obj.get("timestamp") or ""
                        if day_of(stamp) != day:
                            continue
                        body = " ".join(text_of(obj).split())
                        if not body:
                            continue
                        if not want_thinking and body == "[thinking]":
                            continue
                        # A tool result and a typed prompt are both "user" in
                        # the transcript and are not the same thing at all.
                        if obj["type"] == "assistant":
                            who = name
                        else:
                            content = (obj.get("message") or {}).get("content")
                            blocks = content if isinstance(content, list) else []
                            kinds = {b.get("type") for b in blocks
                                     if isinstance(b, dict)}
                            who = "result" if "tool_result" in kinds else "typed"
                        rows.append({"stamp": stamp, "cite": f"{short}:{n}",
                                     "repo": repo, "role": who, "body": body})
            except OSError:
                continue
    rows.sort(key=lambda r: r["stamp"])
    return rows


# render writes the stream. The cite goes first because it is the column a
# reader acts on; everything else is there to decide whether to.
def render(rows, width, out):
    print(f"{'cite':<16}{'time':<10}{'repo':<14}{'who':<17}what", file=out)
    print("-" * (50 + width), file=out)
    for r in rows:
        print(f"{r['cite']:<16}{local_time(r['stamp']):<10}"
              f"{r['repo'][:13]:<14}{r['role'][:15]:<17}{r['body'][:width]}", file=out)


# summarise counts the day by session, so the scale of the record is visible
# before anyone reads it.
def summarise(rows, out):
    by = {}
    for r in rows:
        k = (r["cite"].split(":")[0], r["repo"])
        by[k] = by.get(k, 0) + 1
    print(f"\n{len(rows)} records across {len(by)} sessions", file=out)
    for (short, repo), n in sorted(by.items(), key=lambda kv: -kv[1]):
        print(f"  {n:6}  {short}  {repo}", file=out)


# main assembles one day and writes it where asked.
def main():
    ap = argparse.ArgumentParser(
        description="Every agent's activity for a day, time-ordered.")
    ap.add_argument("--day", default=datetime.now().strftime("%Y-%m-%d"),
                    help="date to render, YYYY-MM-DD (default today)")
    ap.add_argument("-w", "--width", type=int, default=200,
                    help="truncate each record at this width (default 200)")
    ap.add_argument("--thinking", action="store_true",
                    help="include reasoning placeholders, normally dropped")
    ap.add_argument("-o", "--out", help="write here instead of stdout")
    args = ap.parse_args()

    rows = collect(args.day, args.thinking)
    if not rows:
        print(f"no records for {args.day}", file=sys.stderr)
        return 1
    fh = open(args.out, "w") if args.out else sys.stdout
    try:
        render(rows, args.width, fh)
        summarise(rows, fh)
    finally:
        if args.out:
            fh.close()
            print(f"{args.out}: {len(rows)} records", file=sys.stderr)
    return 0


if __name__ == "__main__":
    sys.exit(main())
