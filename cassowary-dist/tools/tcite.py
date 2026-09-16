#!/usr/bin/env python3
# tcite.py -- resolve a daylog cite back into the transcript it points at.
#
# daylog.py prints a cite column, "<session8>:<line>", naming the line in a
# session's transcript where a commit's short sha first appears. That is the
# evidence the attribution rests on. This turns the referent back into
# something a person can read.
#
# The transcripts are jsonl and are not readable as they sit: one JSON object
# per line, eighty megabytes, with tool payloads inline. This prints the
# messages around a cite as time, role and text, and nothing else.
#
# why a line number and not a message UUID.
#
# A uuid would be stabler and is the better referent in principle. A line
# number is what grep already produced during attribution, so it costs nothing
# and cannot disagree with the evidence. Transcripts are append-only, so a line
# number stays valid for a file that is still being written. It stops being
# valid if a transcript is ever rewritten, which nothing here does.
#
# Reads transcripts. Writes nothing.

import argparse
import json
import sys
from datetime import datetime
from pathlib import Path

PROJECTS = Path.home() / ".claude" / "projects"


# find_transcript locates the jsonl for a session by its first eight characters,
# searching every project directory because a session id does not say which
# project it belongs to.
def find_transcript(short):
    hits = []
    if not PROJECTS.is_dir():
        return hits
    for d in PROJECTS.iterdir():
        if not d.is_dir():
            continue
        for f in d.glob("*.jsonl"):
            if f.stem[:8] == short:
                hits.append(f)
    return hits


# text_of flattens a transcript message down to readable text. Content is either
# a bare string or a list of blocks, and only some blocks carry prose; a tool
# call is reported by name rather than by payload, because the payload is the
# reason the raw file is unreadable.
def text_of(obj):
    msg = obj.get("message") or {}
    content = msg.get("content")
    if isinstance(content, str):
        return content
    if not isinstance(content, list):
        return ""
    parts = []
    for block in content:
        if not isinstance(block, dict):
            continue
        kind = block.get("type")
        if kind == "text":
            parts.append(block.get("text", ""))
        elif kind == "thinking":
            parts.append("[thinking]")
        elif kind == "tool_use":
            parts.append(f"[tool_use {block.get('name', '?')}]")
        elif kind == "tool_result":
            # The result content is the evidence a cite points at. Printing a
            # bare marker here made the tool answer "where" and not "what",
            # which is the only question it exists to answer.
            parts.append(result_text(block.get("content")))
    return "\n".join(p for p in parts if p)


# result_text flattens a tool result, whose content is a string or a list of
# blocks, into the text it carried.
def result_text(content):
    if isinstance(content, str):
        return content
    if not isinstance(content, list):
        return ""
    out = []
    for b in content:
        if isinstance(b, dict) and b.get("type") == "text":
            out.append(b.get("text", ""))
        elif isinstance(b, dict) and b.get("type") == "image":
            out.append("[image]")
    return "\n".join(out)


# excerpt centres a long body on the first occurrence of needle, so a cite that
# matched deep inside an eighty-kilobyte tool result shows the match rather than
# the first hundred characters of something else.
def excerpt(body, needle, width):
    if needle:
        i = body.find(needle)
        if i > 0:
            start = max(0, i - width // 3)
            return ("..." if start else "") + body[start:start + width]
    return body[:width]


# read_window returns the transcript lines from start to end inclusive, parsed,
# skipping anything that does not parse rather than failing the whole read.
def read_window(path, start, end):
    out = []
    with path.open() as fh:
        for n, line in enumerate(fh, 1):
            if n < start:
                continue
            if n > end:
                break
            try:
                obj = json.loads(line)
            except json.JSONDecodeError:
                out.append((n, None))
                continue
            # Transcripts carry non-message records -- last-prompt, custom-title,
            # ai-title and others. They have no role and no prose and rendered as
            # empty rows, which read as damage rather than as "not a message".
            if obj.get("type") not in ("user", "assistant"):
                continue
            out.append((n, obj))
    return out


# local_time converts a transcript's UTC stamp to the reader's clock. daylog
# renders local and this rendered UTC, so the same moment appeared as two times
# seven hours apart and a cite looked like it predated the commit it proved.
def local_time(text):
    if not text:
        return ""
    try:
        return (datetime.fromisoformat(text.replace("Z", "+00:00"))
                .astimezone().strftime("%H:%M:%S"))
    except ValueError:
        return text[11:19]


# render prints one window, marking the cited line so it is findable in output
# that is otherwise uniform.
def render(rows, target, width, raw, needle=""):
    for n, obj in rows:
        mark = ">>" if n == target else "  "
        if obj is None:
            print(f"{mark} {n:<8} [unparseable line]")
            continue
        if raw:
            print(f"{mark} {n:<8} {json.dumps(obj)[:width]}")
            continue
        stamp = local_time(obj.get("timestamp"))
        role = obj.get("type", "?")
        body = " ".join(text_of(obj).split())
        print(f"{mark} {n:<8} {stamp:<9} {role:<10} {excerpt(body, needle, width)}")


# parse_cite splits the referent daylog prints. It is deliberately strict: a
# malformed cite is a typo, and guessing at one would resolve to the wrong
# evidence silently.
def parse_cite(text):
    if ":" not in text:
        raise ValueError("a cite is <session8>:<line>, for example 4b732342:14872")
    short, _, lineno = text.partition(":")
    if len(short) != 8 or not lineno.isdigit():
        raise ValueError("a cite is <session8>:<line>, for example 4b732342:14872")
    return short, int(lineno)


# main resolves one cite and prints the messages around it.
def main():
    ap = argparse.ArgumentParser(
        description="Resolve a daylog cite into readable transcript text.")
    ap.add_argument("cite", help="<session8>:<line>, from daylog.py's cite column")
    ap.add_argument("-c", "--context", type=int, default=3,
                    help="lines either side (default 3)")
    ap.add_argument("-w", "--width", type=int, default=160,
                    help="truncate each message at this width (default 160)")
    ap.add_argument("-m", "--match",
                    help="centre each excerpt on this text; use the commit sha "
                         "the cite was found by, to see what actually matched")
    ap.add_argument("--raw", action="store_true",
                    help="print the raw JSON instead of rendered text")
    args = ap.parse_args()

    try:
        short, lineno = parse_cite(args.cite)
    except ValueError as exc:
        print(exc, file=sys.stderr)
        return 2

    hits = find_transcript(short)
    if not hits:
        print(f"no transcript for session {short} under {PROJECTS}", file=sys.stderr)
        return 1
    if len(hits) > 1:
        # Two files for one session id would mean the id is not unique, which
        # would invalidate every cite. Report it rather than picking one.
        print(f"session {short} matches {len(hits)} transcripts:", file=sys.stderr)
        for h in hits:
            print(f"  {h}", file=sys.stderr)
        return 1

    path = hits[0]
    start = max(1, lineno - args.context)
    rows = read_window(path, start, lineno + args.context)
    if not any(n == lineno for n, _ in rows):
        print(f"{path} has no line {lineno}", file=sys.stderr)
        return 1

    print(f"{path}")
    print(f"lines {start}-{lineno + args.context}, cited line marked >>")
    print("-" * 100)
    render(rows, lineno, args.width, args.raw, args.match or "")
    return 0


if __name__ == "__main__":
    sys.exit(main())
