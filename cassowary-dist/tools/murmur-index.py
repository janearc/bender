#!/usr/bin/env python3
# murmur-index -- index a corpus of agent-directed messages, and pull them out
# on request. The index prints names, counts and timestamps and never text; the
# pull writes text to a file so a person reads it rather than an agent.
#
#   murmur-index.py FILE.jsonl
#       every agent named in the corpus, with counts and the last five
#       timestamps of messages addressed to it
#
#   murmur-index.py FILE.jsonl --origins AGENT
#       who sent the messages that mention that agent, resolved to the agent
#       name of the originating session where one exists
#
#   murmur-index.py FILE.jsonl --pull AGENT [--n 5] [--out PATH] [--role user]
#                              [--from-human]
#       write the last N messages mentioning that agent to a markdown file.
#
# Note on --role: the agents field is a mention index, not an address. A record
# tagged arbiter-2 is one whose text names arbiter-2, and role=user means it
# arrived as input to some session -- which may be the operator typing, or one
# agent relaying to another. --from-human keeps only records originating in a
# session that never registered an agent name.
#
# Records are one JSON object per line with ts, day, session, project, role,
# agents and text.

import argparse
import glob
import json
import os
import re
from collections import Counter, defaultdict
from pathlib import Path

PROJECTS = os.path.expanduser("~/.claude/projects")


# map every session id on disk to the last agent name it answered to. a session
# with no name is taken to be a person's, since agents in this fleet were named
def session_names():
    names = {}
    for path in glob.glob(f"{PROJECTS}/*/*.jsonl"):
        sid = os.path.basename(path)[:-6]
        last = None
        with open(path, errors="replace") as fh:
            for line in fh:
                m = re.search(r'"agentName":"([^"]*)"', line)
                if m:
                    last = m.group(1)
        if last:
            names[sid] = last
    return names


# read the corpus once, keeping records grouped by the agents they name
def load(path):
    by_agent = defaultdict(list)
    total = 0
    for line in open(path, errors="replace"):
        line = line.strip()
        if not line:
            continue
        try:
            rec = json.loads(line)
        except json.JSONDecodeError:
            continue
        total += 1
        for a in rec.get("agents") or []:
            by_agent[a].append(rec)
    for a in by_agent:
        by_agent[a].sort(key=lambda r: r.get("ts") or "")
    return by_agent, total


# counts and the tail of timestamps, per agent. no text is touched
def index(by_agent, total, tail=5):
    print(f"{'agent':<24}{'msgs':>6}{'to':>6}{'from':>6}  {'first':<20}{'last':<20}")
    for a in sorted(by_agent, key=lambda x: -len(by_agent[x])):
        recs = by_agent[a]
        to = sum(1 for r in recs if r.get("role") == "user")
        fro = sum(1 for r in recs if r.get("role") == "assistant")
        print(f"{a:<24}{len(recs):>6}{to:>6}{fro:>6}  "
              f"{(recs[0].get('ts') or '?')[:19]:<20}{(recs[-1].get('ts') or '?')[:19]:<20}")
    print(f"\n{total} records, {len(by_agent)} agents\n")
    print(f"last {tail} messages addressed to each agent (role=user):")
    for a in sorted(by_agent):
        to = [r for r in by_agent[a] if r.get("role") == "user"]
        stamps = " ".join((r.get("ts") or "?")[:19] for r in to[-tail:])
        print(f"  {a:<24}{stamps}")


# who originated the records that mention this agent
def origins(by_agent, agent, names):
    recs = by_agent.get(agent) or []
    c = Counter()
    for r in recs:
        if r.get("role") != "user":
            continue
        sid = r.get("session") or "?"
        c[(sid[:8], names.get(sid, "(unnamed session)"))] += 1
    print(f"inbound records mentioning {agent}, by originating session:")
    for (sid, nm), n in c.most_common():
        print(f"  {sid}  {nm:<24}{n:>5}")
    human = sum(n for (sid, nm), n in c.items() if nm == "(unnamed session)")
    print(f"\n  from named agents : {sum(c.values()) - human}")
    print(f"  from unnamed      : {human}")


# write the tail of one agent's messages out for a person to read
def pull(by_agent, agent, n, out, role, names=None, from_human=False):
    recs = by_agent.get(agent)
    if not recs:
        print(f"no agent named {agent!r}")
        return 1
    if role:
        recs = [r for r in recs if r.get("role") == role]
    if from_human:
        recs = [r for r in recs if (names or {}).get(r.get("session")) is None]
    recs = recs[-n:]
    dest = Path(out or f"{agent}-last{n}.md")
    with open(dest, "w") as fh:
        fh.write(f"# {agent}: last {len(recs)} messages"
                 f"{f' (role={role})' if role else ''}\n\n")
        for r in recs:
            sid = r.get("session") or "?"
            who = (names or {}).get(sid, "unnamed")
            fh.write(f"## {r.get('ts')}  role={r.get('role')}  "
                     f"from={who}  session={sid[:8]}\n\n")
            fh.write((r.get("text") or "").rstrip() + "\n\n")
    print(f"{len(recs)} messages -> {dest} ({dest.stat().st_size} bytes)")
    return 0


def main():
    p = argparse.ArgumentParser()
    p.add_argument("corpus")
    p.add_argument("--pull")
    p.add_argument("--origins")
    p.add_argument("--from-human", action="store_true")
    p.add_argument("--n", type=int, default=5)
    p.add_argument("--out")
    p.add_argument("--role", default=None)
    p.add_argument("--tail", type=int, default=5)
    a = p.parse_args()

    by_agent, total = load(a.corpus)
    names = session_names() if (a.origins or a.from_human or a.pull) else {}
    if a.origins:
        origins(by_agent, a.origins, names)
        return 0
    if a.pull:
        return pull(by_agent, a.pull, a.n, a.out, a.role, names, a.from_human)
    index(by_agent, total, a.tail)
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
