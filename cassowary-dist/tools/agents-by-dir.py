#!/usr/bin/env python3
# agents-by-dir.py -- which Claude sessions are in which working directory.
#
# The estate's rule is one session per working tree, and until now there was no
# way to check it. ListAgents gives names and tmux panes but not directories, so
# answering "is anyone already in this repo" meant asking around or guessing.
#
# Two independent sources, joined:
#
#   /tmp/cc-socks/<pid>.sock   one per session that has ever run. The name is the
#                              process id, so a live pid resolves to a real cwd
#                              through lsof. A dead pid means a stale socket.
#   ~/.claude/projects/<dir>/  one directory per working directory a session has
#                              ever run in, named by sanitising the path. The
#                              *.jsonl inside are transcripts, one per session.
#
# The socket side is authoritative for what is running now. The projects side is
# the only record of what ran before.
#
# what this does not answer. cwd is where a session was launched, not everything
# it touches. On 2026-09-03 a session with cwd in flipr was reviewing kingfisher,
# building a local instance and reading its mounts. So an empty result here is
# not proof that nobody is working in a tree -- it is proof that nobody launched
# there. Treat it as one input to the one-session-per-tree question, not the
# answer to it.
#
# which identifier this reports, and why.
#
# Three identify a session and they are not equally useful.
#
#   session id   the transcript's uuid. durable. It survives a resume, a restart and
#                a reboot -- on 2026-09-03 a session resumed after running overnight
#                kept its 6.8 MB transcript and got a new pid. This is what daylog
#                attributes commits against and what this tool leads with.
#   pid          useful for ending a process and for nothing else. Gone on restart,
#                and you cannot address a message to it.
#   name         what you actually type to message someone, and runtime-only.
#
# why this cannot report names.
#
# Sessions have names -- porter, gaggle-1c, dev-c6 -- and a name is what you address
# a message to. Those names are runtime-only. As of 2026-09-03 a session's name
# appears nowhere on disk: not beside its socket, not in its own transcript, not
# under ~/.claude. The only place a name is recorded is in the transcript of a
# session it messaged, as the sender.
#
# So a shell-reachable tool can offer a pid and a working directory and nothing more.
# Resolving a pid to a name needs ListAgents, which is a tool call. Run that, match on
# the working directory, and you have who to talk to.
#
# One consequence worth knowing: auto-generated names are ephemeral and change across
# restarts, so anything written down -- a prompt that says "ask the porter session",
# a runbook, a handoff -- should refer to a name that was deliberately assigned.
#
# Reads only metadata: file names, sizes and modification times. Never opens a
# transcript body, and prints no transcript content.

import argparse
import json
import os
import re
import subprocess
import sys
import time
from datetime import datetime, timezone
from pathlib import Path

SOCKDIR = Path("/tmp/cc-socks")
PROJECTS = Path.home() / ".claude" / "projects"


# unsanitise turns a projects directory name back into the path it stands for.
# The encoding replaces every "/" with "-", which is lossy: a real hyphen in a
# path is indistinguishable from a separator. So this is a best guess, and it is
# checked against the filesystem before being trusted.
def unsanitise(name):
    if not name.startswith("-"):
        return None
    guess = "/" + name[1:].replace("-", "/")
    if os.path.isdir(guess):
        return guess
    # Walk back through the ambiguous hyphens, longest real path first, so
    # "/Users/jane/mesh/dev/hall-monitor" resolves rather than failing.
    parts = name[1:].split("-")
    for split in range(len(parts) - 1, 0, -1):
        cand = "/" + "/".join(parts[:split]) + "-" + "-".join(parts[split:])
        if os.path.isdir(cand):
            return cand
    return guess  # unresolvable; return the naive form and let the caller see it


# alive reports whether a pid is a running process.
def alive(pid):
    try:
        subprocess.run(["ps", "-p", str(pid)], capture_output=True, check=True)
        return True
    except subprocess.CalledProcessError:
        return False


# cwd_of asks lsof for a running process's working directory. Returns None when
# the process is gone or lsof cannot see it, rather than guessing.
def cwd_of(pid):
    try:
        out = subprocess.run(["lsof", "-a", "-p", str(pid), "-d", "cwd", "-Fn"],
                             capture_output=True, text=True, timeout=8).stdout
    except (subprocess.SubprocessError, OSError):
        return None
    for line in out.splitlines():
        if line.startswith("n"):
            return line[1:]
    return None


# started_at returns a process's start time as a string, or None.
def started_at(pid):
    try:
        out = subprocess.run(["ps", "-o", "lstart=", "-p", str(pid)],
                             capture_output=True, text=True, timeout=8).stdout
        return out.strip() or None
    except (subprocess.SubprocessError, OSError):
        return None


# live_sessions walks the socket directory and returns one record per socket,
# marking each as running or stale. A stale socket is kept rather than dropped:
# it is the only evidence that a session existed there at all.
def live_sessions():
    out = []
    if not SOCKDIR.is_dir():
        return out
    for sock in sorted(SOCKDIR.glob("*.sock")):
        try:
            pid = int(sock.stem)
        except ValueError:
            continue
        running = alive(pid)
        out.append({
            "pid": pid,
            "running": running,
            "cwd": cwd_of(pid) if running else None,
            "started": started_at(pid) if running else None,
            "sock_mtime": sock.stat().st_mtime,
        })
    return out


# transcripts_for returns the session transcripts recorded for one working
# directory, newest first. Size and mtime only; the bodies are never read.
def transcripts_for(path):
    if not PROJECTS.is_dir():
        return []
    for d in PROJECTS.iterdir():
        if not d.is_dir():
            continue
        if unsanitise(d.name) == path:
            rows = []
            for f in d.glob("*.jsonl"):
                st = f.stat()
                rows.append({"id": f.stem, "bytes": st.st_size, "mtime": st.st_mtime})
            return sorted(rows, key=lambda r: r["mtime"], reverse=True)
    return []


# all_project_dirs returns every working directory that has ever hosted a
# session, with its transcript count and the time of the most recent one.
def all_project_dirs():
    rows = []
    if not PROJECTS.is_dir():
        return rows
    for d in sorted(PROJECTS.iterdir()):
        if not d.is_dir():
            continue
        files = list(d.glob("*.jsonl"))
        if not files:
            continue
        newest = max(f.stat().st_mtime for f in files)
        rows.append({
            "path": unsanitise(d.name),
            "dirname": d.name,
            "sessions": len(files),
            "newest": newest,
            "bytes": sum(f.stat().st_size for f in files),
        })
    return sorted(rows, key=lambda r: r["newest"], reverse=True)


# ago renders a timestamp as a compact relative age.
def ago(ts):
    d = time.time() - ts
    if d < 3600:
        return f"{int(d/60)}m"
    if d < 86400:
        return f"{int(d/3600)}h"
    return f"{int(d/86400)}d"


# human renders a timestamp in local time, to the minute.
def human(ts):
    return datetime.fromtimestamp(ts).strftime("%Y-%m-%d %H:%M")


# session_id_for finds a live session's durable identifier by matching its working
# directory to a projects directory and taking the most recently written transcript.
# The pid is what the socket gives us; the session id is what survives.
def session_id_for(s):
    path = transcript_path_for(s)
    return os.path.basename(path)[:-6] if path else None


# report_live prints the running sessions, optionally narrowed to one directory.
# This is the answer to "is anyone in this tree right now".
def report_live(sessions, only=None):
    running = [s for s in sessions if s["running"]]
    if only:
        running = [s for s in running if s["cwd"] == only]
    print(f"live sessions ({len(running)})")
    if not running:
        print("  none" + (f" launched in {only}" % () if only else ""))
        return
    for s in sorted(running, key=lambda r: r["cwd"] or ""):
        sid = session_id_for(s)
        print(f"  {sid or '(no transcript)':<38} {s['cwd']}")
        print(f"  {'':<38} pid {s['pid']}" + (f", started {s['started']}" if s["started"] else ""))
    print("\n  The first column is the session id and survives restarts and resumes.")
    print("  To message one, resolve it to a name with ListAgents; a pid is only")
    print("  useful for ending a process.")


# report_stale prints sockets whose process is gone. These accumulate and are
# harmless, but a stale socket in a tree is often mistaken for a live session.
def report_stale(sessions):
    stale = [s for s in sessions if not s["running"]]
    if not stale:
        return
    print(f"\nstale sockets ({len(stale)}) -- process gone, socket remains")
    for s in sorted(stale, key=lambda r: r["sock_mtime"], reverse=True):
        print(f"  pid {s['pid']:<7} last active {human(s['sock_mtime'])} ({ago(s['sock_mtime'])} ago)")


# writers_of reports which live sessions have committed to a repository recently,
# regardless of where they were launched.
#
# This exists because cwd is not scope and the estate's one-session-per-tree rule is
# about trees. On 2026-09-03 a session launched in albatross committed to bigbird,
# cassowary and whoops all day; a cwd-only answer would have told another session
# those trees were free. Nothing collided, but only because nobody tried.
#
# candidates, not proof. The signal is a commit's short sha appearing in a session's
# transcript, and that proves the session saw the sha, not that it made the commit.
# Sessions here report shas to each other in cross-session messages constantly, so a
# peer that was merely told about a commit matches too. On 2026-09-03 this reported
# three writers for bigbird when one had committed and two had been told.
#
# It is still worth having: three candidates beats "nobody was launched here", which
# was the previous answer and was actively misleading. Read it as "these sessions
# have handled this repository recently" and go ask them.
#
# An empty result is not proof a tree is unclaimed either: a session reading, or
# editing without committing, leaves no evidence here at all.
def writers_of(path, sessions, hours=6):
    if not os.path.isdir(os.path.join(path, ".git")):
        return []
    since = f"{hours} hours ago"
    try:
        out = subprocess.run(
            ["git", "-C", path, "log", "--all", f"--since={since}", "--format=%h"],
            capture_output=True, text=True, timeout=20).stdout.split()
    except (subprocess.SubprocessError, OSError):
        return []
    if not out:
        return []
    found = []
    for s in sessions:
        if not s["running"]:
            continue
        tpath = transcript_path_for(s)
        if not tpath:
            continue
        for sha in out[:40]:
            try:
                r = subprocess.run(["grep", "-c", "-F", "-m", "1", sha, tpath],
                                   capture_output=True, timeout=25)
                if r.returncode == 0:
                    found.append((s["pid"], sha))
                    break
            except (subprocess.SubprocessError, OSError):
                pass
    return found


# transcript_path_for finds the transcript file for a live session, by matching its
# working directory to a projects directory and taking the most recently written.
def transcript_path_for(s):
    if not s.get("cwd") or not PROJECTS.is_dir():
        return None
    for d in PROJECTS.iterdir():
        if d.is_dir() and unsanitise(d.name) == s["cwd"]:
            files = sorted(d.glob("*.jsonl"), key=lambda f: f.stat().st_mtime, reverse=True)
            return str(files[0]) if files else None
    return None


# under returns whether a path is at or beneath a root, so asking about ~/mesh
# answers for everything in the estate rather than only for that exact directory.
# Compares resolved paths with a trailing separator, so /a/bc is not read as being
# under /a/b.
def under(path, root):
    path, root = os.path.realpath(path), os.path.realpath(root)
    return path == root or path.startswith(root.rstrip("/") + os.sep)


# report_tree prints every live session at or beneath a root, and every directory
# beneath it that has ever hosted one. Asking this of ~/mesh answers the question
# that matters most -- whether anything is live in prod -- which no per-directory
# query would surface unless you already suspected it.
def report_tree(root, sessions):
    print(f"everything under  {root}\n")
    live = [s for s in sessions if s["running"] and s["cwd"] and under(s["cwd"], root)]
    print(f"live sessions ({len(live)})")
    if not live:
        print("  none")
    for s in sorted(live, key=lambda r: r["cwd"]):
        sid = session_id_for(s)
        print(f"  {sid or '(no transcript)':<38} {s['cwd']}")
        print(f"  {'':<38} pid {s['pid']}" + (f", started {s['started']}" if s["started"] else ""))

    rows = [r for r in all_project_dirs() if r["path"] and under(r["path"], root)]
    live_paths = {s["cwd"] for s in live}
    print(f"\ndirectories that have hosted a session ({len(rows)}), newest first")
    print(f"  {'live':<6}{'sess':>4}  {'newest':<17}{'age':>5}  path")
    for r in sorted(rows, key=lambda r: r["newest"], reverse=True):
        mark = "live" if r["path"] in live_paths else ""
        print(f"  {mark:<6}{r['sessions']:>4}  {human(r['newest']):<17}{ago(r['newest']):>5}  {r['path']}")
    if not rows:
        print("  none recorded")
    print("\n  Session ids survive restarts and resumes. Resolve one to a name with")
    print("  ListAgents to message it. cwd is where a session was launched, not")
    print("  everything it touches, so this is not a complete picture of who is")
    print("  working where.")


# report_dir prints everything known about one working directory: whether a
# session is live there now, and what has run there before.
def report_dir(path, sessions):
    print(f"working directory  {path}")
    if not os.path.isdir(path):
        print("  (does not exist on disk)")
    print()
    report_live(sessions, only=path)
    w = writers_of(path, sessions)
    print(f"\nsessions that have handled this repo ({len(w)}), last 6h")
    print("  Candidates, not proof. The signal is a commit sha appearing in a")
    print("  session's transcript, which a peer that was merely told also matches.")
    if w:
        for pid, sha in w:
            print(f"  pid {pid:<7} knows {sha}")
        print("  Ask them before working here. cwd is not scope.")
    else:
        print("  none. Not proof the tree is unclaimed: reading, or editing without")
        print("  committing, leaves no evidence here.")

    ts = transcripts_for(path)
    print(f"\ntranscripts ({len(ts)})")
    if not ts:
        print("  none recorded -- no session has run here")
        return
    for t in ts[:20]:
        print(f"  {t['id']}  {human(t['mtime'])}  ({ago(t['mtime'])} ago)  {t['bytes']//1024} KiB")
    if len(ts) > 20:
        print(f"  ... and {len(ts)-20} older")


# report_all prints every directory that has hosted a session, newest first,
# with live sessions marked. This is the overview.
def report_all(sessions, limit):
    live_by_cwd = {s["cwd"] for s in sessions if s["running"] and s["cwd"]}
    rows = all_project_dirs()
    print(f"working directories ({len(rows)}), newest first")
    print(f"{'live':<6} {'sess':>4} {'newest':>16} {'age':>5}  path")
    for r in rows[:limit]:
        mark = "live" if r["path"] in live_by_cwd else ""
        print(f"{mark:<6} {r['sessions']:>4} {human(r['newest']):>16} {ago(r['newest']):>5}  {r['path']}")
    if len(rows) > limit:
        print(f"... and {len(rows)-limit} more; pass --limit to see them")
    orphans = live_by_cwd - {r["path"] for r in rows}
    if orphans:
        print(f"\nlive with no transcript directory ({len(orphans)})")
        for o in sorted(orphans):
            print(f"  {o}")


# main parses arguments and dispatches. A bare invocation is the overview; a
# directory argument narrows to that tree.
def main():
    ap = argparse.ArgumentParser(
        description="Which Claude sessions are in which working directory.")
    ap.add_argument("dirname", nargs="?",
                    help="a working directory to report on; omit for the overview")
    ap.add_argument("--limit", type=int, default=25,
                    help="rows in the overview (default 25)")
    ap.add_argument("-r", "--tree", action="store_true",
                    help="descend: report every session at or beneath the directory")
    ap.add_argument("--live-only", action="store_true",
                    help="print only running sessions and exit")
    args = ap.parse_args()

    sessions = live_sessions()

    if args.live_only:
        report_live(sessions)
        report_stale(sessions)
        return 0

    if args.dirname:
        path = os.path.realpath(os.path.expanduser(args.dirname))
        if args.tree:
            report_tree(path, sessions)
        else:
            report_dir(path, sessions)
        return 0

    report_all(sessions, args.limit)
    print()
    report_live(sessions)
    report_stale(sessions)
    return 0


if __name__ == "__main__":
    sys.exit(main())
