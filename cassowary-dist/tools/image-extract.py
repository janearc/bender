#!/usr/bin/env python3
# image-extract -- recover every image from a session that is still on disk.
#
# Images reach a session three ways and survive for different lengths of time:
# pasted images are stored inline in the transcript and last as long as it
# does; the client also drops a copy in an image cache that is cleared without
# warning; and files the session read are only there while nobody deletes
# them. This pulls from all three, deduplicates by content, and writes the
# bytes out unchanged.
#
# This tool writes image files, which is the exception to the rule that
# nothing here emits what it reads: the point is to get a picture back.
# It prints a manifest -- source, size, dimensions -- and never the image.
#
#   image-extract.py --list albert          what is recoverable, write nothing
#   image-extract.py albert                 write them all
#   image-extract.py --from pasted albert   only images you pasted in
#   image-extract.py --from read albert     only images the session read
#   image-extract.py <session-id> <outdir>  somewhere other than the default
#
# Default outdir is ~/temp/session-<first 8 chars of id>-images/.
#
# Provenance is the useful column. An image you pasted sits in the user turn
# itself; one the session read comes back inside a tool result, which is a
# user-role record too, so role alone reports both as yours. They are told
# apart here by where in the record the block sits.

import hashlib
import json
import os
import re
import subprocess
import sys
from pathlib import Path

PROJECTS = Path(os.path.realpath(Path.home() / ".claude" / "projects"))
CACHE = Path.home() / ".claude" / "image-cache"
UUID = re.compile(r"^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$", re.I)
NAME_IN_LINE = re.compile(r'^([^:]+):"agentName":"([^"]*)"')
PATH_IN_LINE = re.compile(r'"file_path":"([^"]+\.(?:png|jpe?g|gif|webp))"', re.I)
SUFFIX = {"image/png": ".png", "image/jpeg": ".jpg", "image/gif": ".gif",
          "image/webp": ".webp"}


# one grep over every transcript, as session-extract does, so a session can be
# named rather than remembered as a uuid
def scan_names():
    proc = subprocess.run(
        ["grep", "-RHo", '"agentName":"[^"]*"', str(PROJECTS)],
        capture_output=True, text=True,
    )
    found = {}
    for line in proc.stdout.splitlines():
        m = NAME_IN_LINE.match(line)
        if m:
            found.setdefault(m.group(1), []).append(m.group(2))
    return found


# the transcript for a session id or a name it ever answered to
def find_session(want):
    if UUID.match(want):
        hits = list(PROJECTS.glob(f"*/{want}.jsonl"))
        if hits:
            return hits[0]
        sys.exit(f"image-extract: no transcript for {want}")
    matches = [p for p, names in scan_names().items()
               if any(want.lower() in n.lower() for n in names)]
    if not matches:
        sys.exit(f"image-extract: no session named {want!r}")
    if len(matches) > 1:
        print(f"image-extract: {len(matches)} sessions match {want!r}:")
        for p in sorted(matches):
            print("   ", Path(p).stem)
        sys.exit(1)
    return Path(matches[0])


# width and height straight out of the file's own header, so this stays
# stdlib: no imaging library for a number the format already states
def dimensions(raw):
    if raw[:8] == b"\x89PNG\r\n\x1a\n" and raw[12:16] == b"IHDR":
        return int.from_bytes(raw[16:20], "big"), int.from_bytes(raw[20:24], "big")
    if raw[:3] == b"GIF":
        return int.from_bytes(raw[6:8], "little"), int.from_bytes(raw[8:10], "little")
    if raw[:4] == b"RIFF" and raw[8:12] == b"WEBP":
        if raw[12:16] == b"VP8X":
            return (int.from_bytes(raw[24:27], "little") + 1,
                    int.from_bytes(raw[27:30], "little") + 1)
        if raw[12:16] == b"VP8 ":
            return (int.from_bytes(raw[26:28], "little") & 0x3FFF,
                    int.from_bytes(raw[28:30], "little") & 0x3FFF)
    if raw[:2] == b"\xff\xd8":
        i = 2
        while i + 9 < len(raw):
            if raw[i] != 0xFF:
                i += 1
                continue
            marker, size = raw[i + 1], int.from_bytes(raw[i + 2:i + 4], "big")
            if marker in (0xC0, 0xC1, 0xC2, 0xC3, 0xC5, 0xC6, 0xC7, 0xC9, 0xCA, 0xCB):
                return (int.from_bytes(raw[i + 7:i + 9], "big"),
                        int.from_bytes(raw[i + 5:i + 7], "big"))
            i += 2 + size
    return 0, 0


# every image block in one record, however deep it sits, carrying whether it
# was found inside a tool result. that is the whole difference between an
# image the operator pasted and one the session went and read.
def blocks(node, out, in_tool=False):
    if isinstance(node, dict):
        here = in_tool or node.get("type") in ("tool_result", "tool_use")
        src = node.get("source")
        if node.get("type") == "image" and isinstance(src, dict) and src.get("data"):
            out.append((src.get("media_type", ""), src["data"], here))
        for v in node.values():
            blocks(v, out, here)
    elif isinstance(node, list):
        for v in node:
            blocks(v, out, in_tool)


# everything recoverable, deduplicated by content, newest source last so the
# manifest reads in the order the session met them
def gather(path, want_role):
    import base64
    found = {}

    def keep(raw, role, when, where, note):
        h = hashlib.sha256(raw).hexdigest()
        if h in found:
            found[h]["seen"] += 1
            # the same bytes often survive in more than one place; say so
            if where not in found[h]["where"]:
                found[h]["where"] += "+" + where
            return
        w, ht = dimensions(raw)
        found[h] = {"sha": h, "raw": raw, "role": role, "when": when,
                    "where": where, "note": note, "w": w, "h": ht, "seen": 1}

    with open(path, errors="replace") as fh:
        for line in fh:
            if '"image"' not in line and "file_path" not in line:
                continue
            try:
                rec = json.loads(line)
            except ValueError:
                continue
            role = (rec.get("message") or {}).get("role") or rec.get("type") or "?"
            when = (rec.get("timestamp") or "")[:19]
            found_blocks = []
            blocks(rec.get("message", rec), found_blocks)
            for media, data, in_tool in found_blocks:
                kind = "read" if in_tool else ("pasted" if role == "user" else role)
                if want_role not in ("all", kind):
                    continue
                try:
                    keep(base64.b64decode(data), kind, when, "transcript", media)
                except Exception:
                    pass
            # files the session read, if nobody has deleted them since
            for m in PATH_IN_LINE.finditer(line):
                p = Path(m.group(1))
                if p.is_file() and want_role in ("all", "read"):
                    try:
                        keep(p.read_bytes(), "read", when, "on disk", str(p))
                    except OSError:
                        pass

    # the client's own cache, which is cleared without warning and is the
    # first thing to disappear
    sid = path.stem
    cdir = CACHE / sid
    if cdir.is_dir():
        for p in sorted(cdir.iterdir()):
            if p.is_file() and want_role in ("all", "pasted"):
                try:
                    keep(p.read_bytes(), "pasted", "", "image cache", p.name)
                except OSError:
                    pass
    return list(found.values())


def main():
    args = [a for a in sys.argv[1:]]
    listing = "--list" in args
    want_role = "all"
    if "--from" in args:
        i = args.index("--from")
        want_role = args[i + 1]
        del args[i:i + 2]
    args = [a for a in args if a != "--list"]
    if not args:
        sys.exit(__doc__ or "usage: image-extract.py [--list] [--from ROLE] "
                            "SESSION [OUTDIR]")

    path = find_session(args[0])
    sid = path.stem
    out = Path(args[1]) if len(args) > 1 else Path.home() / "temp" / f"session-{sid[:8]}-images"

    items = gather(path, want_role)
    items.sort(key=lambda d: (d["when"], d["sha"]))
    print(f"session {sid}")
    print(f"{len(items)} distinct images, from {path}")
    print()
    print(f"{'#':>3}  {'what':<7} {'kept':<18} {'when':<19} {'pixels':>11} "
          f"{'KB':>7}  sha")
    for n, d in enumerate(items, 1):
        size = f"{d['w']}x{d['h']}" if d["w"] else "?"
        print(f"{n:3d}  {d['role']:<7} {d['where']:<18} {d['when']:<19} "
              f"{size:>11} {len(d['raw']) / 1024:7.0f}  {d['sha'][:12]}")
    pasted = sum(1 for d in items if d["role"] == "pasted")
    print(f"\n{pasted} pasted, {len(items) - pasted} read by the session")

    if listing:
        print("\nnothing written (--list)")
        return
    out.mkdir(parents=True, exist_ok=True)
    for n, d in enumerate(items, 1):
        ext = SUFFIX.get(d["note"], "")
        if not ext:
            ext = {b"\x89P": ".png", b"\xff\xd8": ".jpg", b"GI": ".gif",
                   b"RI": ".webp"}.get(d["raw"][:2], ".bin")
        size = f"{d['w']}x{d['h']}" if d["w"] else "unknown"
        name = f"{n:03d}-{d['role']}-{size}-{d['sha'][:8]}{ext}"
        (out / name).write_bytes(d["raw"])
    print(f"\nwrote {len(items)} files to {out}")


if __name__ == "__main__":
    main()
