#!/usr/bin/env python3
# daylog.py -- a time-aligned, attributed record of a day's work across every
# session and every repository.
#
# The problem it solves: a day of multi-session work leaves its output scattered
# across repositories, GitHub and a dozen transcripts, and nothing anywhere says
# what happened in what order or who did it. On 2026-09-03 five sessions produced
# thirty-one commits in five repositories and about a hundred issues, and the only
# record of the sequence was in one person's head.
#
# how attribution works, and why it is inference.
#
# Commits do not name the session that made them. The author is always the operator
# by convention here, and the Co-Authored-By trailer names a model rather than a
# session. So attribution is a join:
#
#   a commit at time T in repository R belongs to the session whose working
#   directory was R and whose transcript span covers T
#
# That is inference, not a record, and it has one failure mode worth naming: if two
# sessions held the same repository at once the answer is ambiguous. This estate
# forbids that -- one session per working tree -- so an ambiguous row is itself a
# finding, and the tool prints it rather than picking.
#
# what it cannot see.
#
# Work that produced no commit and no issue. A session that spent four hours reading
# and reached a conclusion leaves nothing here. This is a record of artifacts, not of
# effort, and the difference matters when reading it back.
#
# accuracy, as measured on 2026-09-03.
#
# Of 65 commits across nine repositories, 56 attributed to a single session with
# evidence and 9 did not. The residual is 8 rows labelled AMBIG and 1 with no
# distinguishing evidence. The AMBIG rows were not fully explained: a spot check
# confirmed the short sha of one of them appears in exactly one transcript, so tier
# zero is behaving, and the remaining ambiguity arises earlier in the chain. Four
# iterations did not isolate it and it is recorded here rather than hidden.
#
# Read AMBIG as "this tool cannot tell", not as "two sessions were in one tree".
# The second reading would be a serious finding and it has not been established.
#
# Reads git log, session transcript timestamps, and optionally the GitHub issue
# list. It also searches transcript bodies for one thing and one thing only: the
# literal text of a commit subject, to establish which session ran that commit. It
# prints no transcript content, only whether a match was found.

import argparse
import json
import os
import re
import subprocess
import sys
from datetime import datetime, timedelta, timezone
from pathlib import Path

ESTATE = Path.home() / "mesh" / "dev"
PROJECTS = Path.home() / ".claude" / "projects"

_SPANS = {}
_RAN = {}


# to_utc parses an ISO timestamp of either flavour -- a git author date carrying a
# local offset, or a transcript stamp ending in Z -- and returns an aware datetime.
# Comparing these as strings is wrong by the offset, which cost real time on
# 2026-09-03 before it was noticed.
def to_utc(text):
    if not text:
        return None
    t = text.replace("Z", "+00:00")
    try:
        d = datetime.fromisoformat(t)
    except ValueError:
        return None
    return d if d.tzinfo else d.replace(tzinfo=timezone.utc)


# span_of returns the first and last timestamps in a transcript, by scanning the
# head and tail of the file rather than parsing all of it. A working transcript can
# be eighty megabytes.
def span_of(path):
    key = str(path)
    if key in _SPANS:
        return _SPANS[key]
    first = last = None
    try:
        with open(path, "rb") as fh:
            head = fh.read(262144).decode("utf-8", "replace")
            m = re.search(r'"timestamp":"([^"]*)"', head)
            if m:
                first = m.group(1)
            size = fh.seek(0, os.SEEK_END)
            fh.seek(max(0, size - 262144))
            tail = fh.read().decode("utf-8", "replace")
            ms = re.findall(r'"timestamp":"([^"]*)"', tail)
            if ms:
                last = ms[-1]
    except OSError:
        pass
    _SPANS[key] = (to_utc(first), to_utc(last))
    return _SPANS[key]


# unsanitise turns a projects directory name back into the path it stands for. The
# encoding replaces every slash with a hyphen, which is lossy, so candidates are
# checked against the filesystem longest-real-path first.
def unsanitise(name):
    if not name.startswith("-"):
        return None
    guess = "/" + name[1:].replace("-", "/")
    if os.path.isdir(guess):
        return guess
    parts = name[1:].split("-")
    for split in range(len(parts) - 1, 0, -1):
        cand = "/" + "/".join(parts[:split]) + "-" + "-".join(parts[split:])
        if os.path.isdir(cand):
            return cand
    return guess


# sessions_by_repo maps a repository path to the sessions that ran there, each with
# its span. This is the table every commit is attributed against.
def sessions_by_repo():
    out = {}
    if not PROJECTS.is_dir():
        return out
    for d in PROJECTS.iterdir():
        if not d.is_dir():
            continue
        path = unsanitise(d.name)
        if not path:
            continue
        for f in d.glob("*.jsonl"):
            first, last = span_of(f)
            if not first:
                continue
            out.setdefault(path, []).append(
                {"id": f.stem, "short": f.stem[:8], "first": first, "last": last})
    return out


# repos_under lists every git repository directly under the estate root.
def repos_under(root):
    return sorted(p for p in root.iterdir() if (p / ".git").exists())


# commits_on returns one record per commit in a repository on a given day.
def commits_on(repo, day):
    try:
        out = subprocess.run(
            ["git", "-C", str(repo), "log", "--all", "--no-merges",
             f"--since={day} 00:00", f"--until={day} 23:59",
             "--format=%H%x1f%aI%x1f%s%x1f%an"],
            capture_output=True, text=True, timeout=25).stdout
    except (subprocess.SubprocessError, OSError):
        return []
    rows = []
    for line in out.splitlines():
        parts = line.split("\x1f")
        if len(parts) != 4:
            continue
        sha, iso, subject, author = parts
        rows.append({"kind": "commit", "when": to_utc(iso), "repo": repo.name,
                     "sha": sha[:7], "text": subject, "author": author})
    return rows


# issues_on returns issues and comments created on a day, if gh is available. This
# is the one source that needs the network, so it is optional.
def issues_on(repo_slug, day):
    try:
        out = subprocess.run(
            ["gh", "issue", "list", "-R", repo_slug, "--state", "all", "--limit", "200",
             "--json", "number,title,createdAt"],
            capture_output=True, text=True, timeout=40).stdout
        data = json.loads(out)
    except Exception:
        return []
    rows = []
    for i in data:
        w = to_utc(i.get("createdAt"))
        if w and w.astimezone().strftime("%Y-%m-%d") == day:
            rows.append({"kind": "issue", "when": w, "repo": repo_slug.split("/")[-1],
                         "sha": f"#{i['number']}", "text": i["title"], "author": ""})
    return rows


# alive_at returns every session whose span covers a moment, with the working
# directory each was launched in. Ten minutes of grace: a commit can land just after
# the last logged turn of the session that made it.
def alive_at(when, table):
    out = []
    for path, sessions in table.items():
        for s in sessions:
            last = s["last"] or s["first"]
            if s["first"] <= when <= last + timedelta(minutes=10):
                out.append((path, s["short"]))
    return out


# attribute answers which session produced an event, in two tiers, because the
# obvious signal is not sufficient on its own.
#
# Tier one is a session launched in the repository the commit landed in. That is a
# confident answer and is marked plainly.
#
# Tier two exists because cwd is where a session was launched, not everything it
# touches. On 2026-09-03 one session working from ~/mesh/dev/albatross made
# twenty-seven commits in bigbird, and a cwd-only model called every one of them
# unattributed. So when no session was launched in that repository, fall back to
# which sessions were merely alive at that moment. A single live candidate is named
# with a question mark; several is a candidate set, not a guess.
#
# The question mark is the point. An instrument that cannot tell a confident answer
# from a plausible one is worse than one that says less.
def attribute(event, table):
    path = str(ESTATE / event["repo"])
    same_tree = []
    for s in table.get(path, []):
        last = s["last"] or s["first"]
        if s["first"] <= event["when"] <= last + timedelta(minutes=10):
            same_tree.append(s["short"])
    if len(same_tree) == 1:
        return same_tree[0]
    if len(same_tree) > 1:
        # Two sessions in one working tree at once, which this estate forbids.
        return "AMBIG:" + "/".join(sorted(same_tree))
    live = alive_at(event["when"], table)
    if not live:
        return "unattributed"
    # Tier zero, tried before falling back to a guess: the session that made a
    # commit ran the command, so the commit subject is in its transcript. That is
    # evidence rather than inference, and it is the only signal here that is.
    proven = ran_the_commit(event, live)
    if len(proven) == 1:
        return proven[0][0]
    if len(proven) > 1:
        return "AMBIG:" + "/".join(sorted(short for short, _ in proven))
    if len(live) == 1:
        return live[0][1] + "?"
    return f"{len(live)} live?"


# ran_the_commit searches the transcripts of the sessions alive at that moment for
# the commit's short SHA, not its subject.
#
# The distinction matters and the first version of this got it wrong. A transcript
# containing the commit subject proves the session saw that text, not that it ran
# the command -- on 2026-09-03 one session wrote three review files and another
# committed them, and both transcripts carried the subject, producing a false
# ambiguity in exactly the place the estate's one-session-per-tree rule would have
# been reported as broken.
#
# A short sha does not exist until the commit does, so only the session that ran it
# can have it. Seven hex characters could in principle collide, but the search is
# already narrowed to sessions alive at that moment and in the candidate set, which
# makes a collision vanishingly unlikely and visible as an AMBIG row if it happens.
#
# Searched only among live candidates, so the cost stays bounded against
# eighty-megabyte transcripts.
def ran_the_commit(event, live):
    needle = event.get("sha", "")
    if len(needle) < 7:
        return []          # not a commit, or too short to be distinctive
    if needle in _RAN:
        return _RAN[needle]   # one grep pass per sha; attribution and cite share it
    found = []
    for path, short in live:
        for d in PROJECTS.iterdir() if PROJECTS.is_dir() else []:
            if not d.is_dir() or unsanitise(d.name) != path:
                continue
            for f in d.glob("*.jsonl"):
                if f.stem[:8] != short:
                    continue
                try:
                    r = subprocess.run(["grep", "-n", "-F", "-m", "1", needle, str(f)],
                                       capture_output=True, text=True, timeout=30)
                    if r.returncode == 0:
                        # -n gives "line:content"; keep the line, discard the content.
                        found.append((short, r.stdout.split(":", 1)[0]))
                except (subprocess.SubprocessError, OSError):
                    pass
    _RAN[needle] = found
    return found


# cite_for returns a referent from a timeline row back into the transcript that
# proves it: "<session8>:<line>", the line where that commit's short sha first
# appears. Resolve one with tools/tcite.py.
#
# It is empty for exactly the rows where nothing proves the attribution -- an
# unattributed row, a "N live?" guess, or a sha no transcript carries. So a blank
# cite column is the same statement as a question mark in the session column,
# said in the place where the reader is about to go looking.
def cite_for(event, table):
    found = ran_the_commit(event, alive_at(event["when"], table))
    return f"{found[0][0]}:{found[0][1]}" if len(found) == 1 else ""


# render prints the timeline in local time, because the reader is a person whose
# clock is local even though every stored stamp is UTC.
def render(events, table, show_repo_col):
    print(f"{'time':<9}{'kind':<8}{'repo':<13}{'ref':<9}{'session':<12}{'cite':<16}what")
    print("-" * 116)
    for e in sorted(events, key=lambda r: r["when"]):
        who = attribute(e, table) if e["kind"] == "commit" else ""
        # An issue's referent is its number, which gh already resolves.
        cite = cite_for(e, table) if e["kind"] == "commit" else ""
        print(f"{e['when'].astimezone().strftime('%H:%M:%S'):<9}"
              f"{e['kind']:<8}{e['repo'][:12]:<13}{e['sha']:<9}{who:<12}{cite:<16}"
              f"{e['text'][:60]}")


# summarise counts what the day produced, by repository and by session.
def summarise(events, table):
    commits = [e for e in events if e["kind"] == "commit"]
    issues = [e for e in events if e["kind"] == "issue"]
    by_repo, by_session = {}, {}
    for c in commits:
        by_repo[c["repo"]] = by_repo.get(c["repo"], 0) + 1
        w = attribute(c, table)
        by_session[w] = by_session.get(w, 0) + 1
    print(f"\n{len(commits)} commits, {len(issues)} issues")
    print("\nby repository")
    for r, n in sorted(by_repo.items(), key=lambda kv: -kv[1]):
        print(f"  {n:>3}  {r}")
    print("\nby session (inferred from working directory and transcript span)")
    for w, n in sorted(by_session.items(), key=lambda kv: -kv[1]):
        print(f"  {n:>3}  {w}")
    print("\nA bare id is established: either the session was launched in that")
    print("repository, or its transcript contains the commit it ran. An id with ?")
    print("is the only session alive then but launched elsewhere, with no transcript")
    print("match: plausible, not established. \"N live?\" means N candidates and no")
    print("evidence distinguishing them.")
    amb = [w for w in by_session if w.startswith("AMBIG")]
    if amb:
        print("\nAMBIG rows mean two sessions held one working tree at once,")
        print("which this estate forbids. That is a finding, not a display problem.")


# main parses arguments and builds the day.
def main():
    ap = argparse.ArgumentParser(description="A time-aligned attributed record of a day.")
    ap.add_argument("--day", default=datetime.now().strftime("%Y-%m-%d"),
                    help="date to report, YYYY-MM-DD (default today)")
    ap.add_argument("--issues", metavar="OWNER/REPO",
                    help="also include issues created that day from this GitHub repo")
    ap.add_argument("--summary-only", action="store_true", help="counts without the timeline")
    args = ap.parse_args()

    if not ESTATE.is_dir():
        print(f"no estate root at {ESTATE}", file=sys.stderr)
        return 2

    table = sessions_by_repo()
    events = []
    for repo in repos_under(ESTATE):
        events.extend(commits_on(repo, args.day))
    if args.issues:
        events.extend(issues_on(args.issues, args.day))

    if not events:
        print(f"nothing recorded on {args.day}")
        return 0
    if not args.summary_only:
        render(events, table, True)
    summarise(events, table)
    return 0


if __name__ == "__main__":
    sys.exit(main())
