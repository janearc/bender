#!/usr/bin/env python3
# corpus -- read session transcripts into turns, once, for everything here.
#
# why it is in the library. Three tools were reading ~/.claude/projects with
# three slightly different ideas of what counts as a turn somebody typed, which
# means three tools could report different denominators for the same day and
# all be right. This is the one definition.
#
# what is excluded, and why each. Tool results are the machine answering
# itself. System reminders, command stdout and bash echoes are the harness.
# Compaction summaries are the assistant's own prose stored under a user role,
# so counting them credits a person with words they did not write. The gate's
# own room is excluded by name because an instrument that reads its own output
# is the failure this repository has already made three times.

import json
import re
from datetime import datetime
from pathlib import Path

PROJECTS = Path.home() / ".claude" / "projects"


# not_typed marks the user records no person typed.
def not_typed(t):
    s = t.lstrip()
    return (s.startswith("<system-reminder>") or "cross-session-message" in t
            or "<command-name>" in t or "<local-command-stdout>" in t
            or "<bash-input>" in t or "<bash-stdout>" in t
            or t.startswith("This session is being continued from a previous")
            or ("Analysis:" in t[:200] and "Summary:" in t[:3000]))


# harvest returns every turn in order as (time, who, text, session).
# The session id travels with the turn because several detectors need to know
# where a session ends, and a global ordering loses that.
def harvest(since=None, projects=PROJECTS):
    turns = []
    for d in projects.iterdir():
        if not d.is_dir() or "worktrees" in d.name or d.name.startswith("-home-"):
            continue
        if "gate-room" in d.name:
            continue
        for f in d.glob("*.jsonl"):
            sid = f.stem[:8]
            for line in f.open():
                if '"timestamp"' not in line:
                    continue
                try:
                    o = json.loads(line)
                except json.JSONDecodeError:
                    continue
                k = o.get("type")
                if k not in ("user", "assistant"):
                    continue
                c = (o.get("message") or {}).get("content")
                blocks = c if isinstance(c, list) else (
                    [{"type": "text", "text": c}] if isinstance(c, str) else [])
                kinds = {b.get("type") for b in blocks if isinstance(b, dict)}
                text = " ".join(b.get("text", "") for b in blocks
                                if isinstance(b, dict)
                                and b.get("type") == "text")
                if k == "user" and ("tool_result" in kinds or not_typed(text)):
                    continue
                if not text.strip():
                    continue
                try:
                    t = datetime.fromisoformat(
                        o["timestamp"].replace("Z", "+00:00")).astimezone()
                except (KeyError, ValueError):
                    continue
                if since and t.strftime("%Y-%m-%d") < since:
                    continue
                turns.append((t, k, text, sid))
    turns.sort(key=lambda r: r[0])
    return turns


# exchanges pairs each assistant turn with the user turn before it in the same
# session, which is what most of the detectors actually need: whether the
# assistant did something it was asked to do.
def exchanges(turns):
    last_user = {}
    out = []
    for t, who, text, sid in turns:
        if who == "user":
            last_user[sid] = (t, text)
        else:
            prev = last_user.get(sid)
            out.append((t, text, sid, prev[1] if prev else None,
                        (t - prev[0]).total_seconds() / 60 if prev else None))
    return out


# session_tails returns the last assistant turn of each session, which is where
# an exit-timed pattern would have to show up if it showed up anywhere.
def session_tails(turns):
    last = {}
    for t, who, text, sid in turns:
        if who == "assistant":
            last[sid] = (t, text, sid)
    return list(last.values())


# sentences splits prose into sentences for the detectors that count sentence
# shapes rather than word occurrences. Deliberately crude: an over-split costs
# a small overcount and an under-split hides one, and the first is the safer
# error for something that must not overstate.
_SENT = re.compile(r"(?<=[.!?])\s+")


def sentences(text):
    return [s.strip() for s in _SENT.split(text) if s.strip()]
