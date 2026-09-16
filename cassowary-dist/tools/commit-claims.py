#!/usr/bin/env python3
# commit-claims -- check whether commits that claim a verification actually ran one.
#
# A commit message is prose written by the thing making the claim, so "tests
# pass" is an assertion, not evidence. This links each claiming commit back to
# the session that made it and looks for the tool call that would have produced
# the claim.
#
#   commit-claims.py [repo-glob] [--window minutes] [--show]
#
# Defaults to every git repository under ~/mesh/dev, a 90 minute lookback, and
# counts only. Prints no commit bodies unless --show.
#
# Only claims with a specific tool signature are checked. "Verified" and
# "measured" are excluded on purpose: verified what, by what means, is a
# judgement, and a checker that guesses is worse than no checker.

import json
import os
import re
import subprocess
import sys
from datetime import datetime, timedelta, timezone

PROJECTS = os.path.realpath(os.path.expanduser("~/.claude/projects"))

# first and last timestamp of every transcript, read cheaply: the first line for
# the start, the tail of the file for the end. a session cannot have made a
# commit that predates it or postdates it, and without this check a session is
# matched merely for having printed the commit's subject in a git log
_SPANS = {}


def span_of(path):
    if path in _SPANS:
        return _SPANS[path]
    first = last = None
    try:
        with open(path, "rb") as fh:
            head = fh.read(262144).decode("utf-8", "replace")
            m = re.search(r'"timestamp":"([^"]*)"', head)
            if m:
                first = m.group(1)
            fh.seek(0, os.SEEK_END)
            size = fh.tell()
            fh.seek(max(0, size - 65536))
            tail = fh.read().decode("utf-8", "replace")
            hits = re.findall(r'"timestamp":"([^"]*)"', tail)
            if hits:
                last = hits[-1]
    except OSError:
        pass
    _SPANS[path] = (first, last)
    return _SPANS[path]


# was this session alive when the commit was made? a generous margin either side
# covers clock skew and a commit written just after the last recorded turn
def alive_at(path, iso):
    first, last = span_of(path)
    if not first or not last:
        return True
    # Compare instants, not strings. Commit dates carry an offset (%aI gives
    # -07:00) and transcript stamps are UTC with a Z, so a lexical comparison
    # is wrong by the offset -- which excluded every session started less than
    # seven hours before its own commit, and mislinked four of the eight
    # unsupported claims to whichever session had merely quoted the sha.
    def at(t):
        t = t.replace("Z", "+00:00")
        try:
            d = datetime.fromisoformat(t)
        except ValueError:
            return None
        return d if d.tzinfo else d.replace(tzinfo=timezone.utc)
    a, b, c = at(first), at(last), at(iso)
    if not (a and b and c):
        return True
    return a <= c <= (b + timedelta(minutes=10))

# claim kind -> (pattern in the commit message, pattern the tool call must match)
CHECKS = {
    "tests-pass": (
        r"\b(tests? (?:pass|passing|green)|all green|ci green|"
        r"\d+ tests? (?:pass|green)|suite green)\b",
        r"\b(go test|npm test|npm run test|pytest|cargo test|bun test|"
        r"make test|go tool test2json|python -m pytest|tsx --test|vitest|"
        r"npx tsx|npx vitest|jest)\b"),
    "lint-clean": (
        r"\b(gofmt clean|lint clean|vet clean|no lint)\b",
        r"\b(gofmt|go vet|golangci-lint|eslint|ruff|tsc)\b"),
    "coverage": (
        r"\bcoverage (?:is|at|now)? ?\d",
        r"-cover|coverprofile|--cov|c8 |nyc "),
    "mutation": (
        r"\bmutation[- ]check",
        r"\b(go test|npm test|pytest|cargo test|bun test)\b"),
}


# every repository with a git directory under the given root
def repos(root):
    import glob
    return sorted(d for d in glob.glob(os.path.expanduser(root))
                  if os.path.isdir(os.path.join(d, ".git")))


# commits with their date, subject and body, oldest last
def commits(repo):
    out = subprocess.run(
        ["git", "-C", repo, "log", "--all",
         "--format=%H%x01%h%x01%aI%x01%s%x01%b%x02"],
        capture_output=True, text=True).stdout
    for rec in out.split("\x02"):
        if not rec.strip():
            continue
        parts = rec.strip("\n").split("\x01")
        if len(parts) >= 4:
            yield parts[0], parts[1], parts[2], parts[3], (parts[4] if len(parts) > 4 else "")


# which transcript made this commit. the short sha appears in git's own output
# after a successful commit; the subject appears in the command that made it.
# a rebase or amend rewrites the sha, so the subject is the more durable key.
#
# A subject can also appear in a session that merely quoted it -- an audit of the
# corpus contaminates the corpus -- so candidates are ordered with sessions whose
# project directory names the repository first, and every candidate is searched
# rather than the first two.
ALIVE_AT = [""]


def find_session(short, subject, repo_name):
    # Collect candidates from both needles before deciding. Short-circuiting on
    # the sha lets a session that merely printed the sha beat the session that
    # wrote the commit, which is how an audit of the corpus matches itself.
    found = {}
    for needle, kind in ((short, "sha"), (subject, "subject")):
        if not needle or len(needle) < 6:
            continue
        r = subprocess.run(["grep", "-rlF", needle, PROJECTS],
                           capture_output=True, text=True)
        for h in r.stdout.split():
            if h.endswith(".jsonl"):
                found.setdefault(h, kind)
    hits = [h for h in found if alive_at(h, ALIVE_AT[0])]
    if not hits:
        return [], None
    key = repo_name.lower().replace("-", "")

    def rank(h):
        proj = os.path.basename(os.path.dirname(h)).lower().replace("-", "")
        return (0 if key in proj else 1, found[h] != "subject")

    hits.sort(key=rank)
    return hits, found[hits[0]]


# every tool call in a session within the window before a moment
def tool_calls_before(path, when, minutes):
    lo = when - timedelta(minutes=minutes)
    out = []
    with open(path, errors="replace") as fh:
        for line in fh:
            if '"tool_use"' not in line:
                continue
            try:
                r = json.loads(line)
            except json.JSONDecodeError:
                continue
            ts = r.get("timestamp", "")
            if not ts:
                continue
            try:
                t = datetime.fromisoformat(ts.replace("Z", "+00:00"))
            except ValueError:
                continue
            if not (lo <= t <= when):
                continue
            for b in (r.get("message") or {}).get("content") or []:
                if isinstance(b, dict) and b.get("type") == "tool_use":
                    out.append(json.dumps(b.get("input", {})))
    return out


def main():
    args = [a for a in sys.argv[1:] if not a.startswith("--")]
    # no implicit estate root: a default that only resolves on one machine
    # reads someone else's tree or nothing at all, and says neither.
    root = args[0] if args else os.environ.get("CASSOWARY_ROOT", "")
    if not root:
        sys.stderr.write(
            "commit-claims: give a path or a glob, or set CASSOWARY_ROOT\n")
        return 2
    window = 90
    for a in sys.argv[1:]:
        if a.startswith("--window"):
            window = int(a.split("=", 1)[1]) if "=" in a else window
    show = "--show" in sys.argv

    rows = []
    for repo in repos(root):
        name = os.path.basename(repo)
        for full, short, iso, subj, body in commits(repo):
            text = f"{subj}\n{body}".lower()
            kinds = [k for k, (claim, _) in CHECKS.items() if re.search(claim, text)]
            if not kinds:
                continue
            try:
                when = datetime.fromisoformat(iso)
            except ValueError:
                continue
            ALIVE_AT[0] = iso
            sessions, how = find_session(short, subj, name)
            if not sessions:
                rows.append((name, short, iso[:16], kinds, "unlinked", ""))
                continue
            calls = []
            for s in sessions[:6]:
                calls += tool_calls_before(s, when, window)
            blob = "\n".join(calls).lower()
            for k in kinds:
                ok = bool(re.search(CHECKS[k][1], blob))
                rows.append((name, short, iso[:16], [k],
                             "supported" if ok else "unsupported",
                             os.path.basename(sessions[0])[:8]))

    from collections import Counter
    by_kind = Counter((r[3][0], r[4]) for r in rows)
    kinds = sorted({k for k, _ in by_kind})
    print(f"{'claim':<14}{'supported':>10}{'unsupported':>13}{'unlinked':>10}")
    for k in kinds:
        s = by_kind[(k, "supported")]
        u = by_kind[(k, "unsupported")]
        n = by_kind[(k, "unlinked")]
        print(f"{k:<14}{s:>10}{u:>13}{n:>10}")
    tot_s = sum(by_kind[(k, "supported")] for k in kinds)
    tot_u = sum(by_kind[(k, "unsupported")] for k in kinds)
    tot_n = sum(by_kind[(k, "unlinked")] for k in kinds)
    linked = tot_s + tot_u
    print(f"\n{linked} claims linked to a session, {tot_n} could not be linked")
    if linked:
        print(f"supported: {tot_s}/{linked} = {100*tot_s/linked:.0f}%")
    if show:
        print("\nunsupported:")
        for r in rows:
            if r[4] == "unsupported":
                print(f"  {r[0]:<12}{r[1]:<9}{r[2]:<18}{r[3][0]:<12}{r[5]}")
    return 0


if __name__ == "__main__":
    sys.exit(main())
