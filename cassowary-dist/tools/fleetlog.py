#!/usr/bin/env python3
# fleetlog.py -- read container logs that outlived the container.
#
# vector-collector has been writing every pod's stdout to the host since 2026-08-28,
# one file per day at ~/var/log/whoops/containers-YYYY.MM.DD.jsonl. Nothing sat in
# front of it, so on 2026-09-03 an operator and an agent both ran `kubectl logs
# --tail` against pods that had already replaced the ones they wanted, three separate
# times, while the history they needed was on disk the whole time.
#
# this is the only way to read a dead pod's logs. `kubectl logs` can show the
# previous container of a running pod and nothing older. Once a pod is replaced its
# output exists here or nowhere.
#
# what it cannot do.
#
# Seven days. A retention CronJob globs containers-*.jsonl and deletes at +7d, so the
# window is fixed and older days are gone rather than archived. The sink path is a
# contract with that job, not a preference.
#
# It reads what vector collected, which is stdout and stderr as the CRI wrote them.
# A service that logs to a file inside its container, or not at all, appears here
# empty, and an empty result is not evidence that nothing happened.
#
# Files are one to one and a half gigabytes a day, so everything here streams and
# prefilters on the raw line before parsing. Do not load one into memory.

import argparse
import json
import os
import re
import sys
from datetime import datetime, timedelta, timezone
from pathlib import Path

LOGDIR = Path.home() / "var" / "log" / "whoops"


# day_files returns the log files covering a set of days, newest last so output
# reads forward in time.
def day_files(days):
    out = []
    for d in days:
        p = LOGDIR / f"containers-{d}.jsonl"
        if p.is_file():
            out.append(p)
    return out


# days_in_range expands a start and end date into the day strings the files use.
def days_in_range(start, end):
    out, cur = [], start
    while cur <= end:
        out.append(cur.strftime("%Y.%m.%d"))
        cur += timedelta(days=1)
    return out


# available lists the days actually on disk, so a caller can see the retention
# window rather than guessing at it.
def available():
    out = []
    for p in sorted(LOGDIR.glob("containers-*.jsonl")):
        m = re.search(r"containers-(\d{4}\.\d{2}\.\d{2})\.jsonl", p.name)
        if m:
            out.append((m.group(1), p.stat().st_size))
    return out


# parse_when accepts an ISO timestamp, or a bare HH:MM which is taken as local time
# on the day being read. Returns an aware UTC datetime.
def parse_when(text, day):
    if not text:
        return None
    if re.fullmatch(r"\d{1,2}:\d{2}(:\d{2})?", text):
        parts = [int(x) for x in text.split(":")]
        while len(parts) < 3:
            parts.append(0)
        local = datetime.strptime(day, "%Y.%m.%d").replace(
            hour=parts[0], minute=parts[1], second=parts[2])
        return local.astimezone().astimezone(timezone.utc)
    t = text.replace("Z", "+00:00")
    try:
        d = datetime.fromisoformat(t)
    except ValueError:
        return None
    return d if d.tzinfo else d.astimezone()


# record_time returns a record's own timestamp, preferring the CRI stamp the
# container emitted over the time vector wrote it.
def record_time(rec):
    for k in ("cri_ts", "timestamp"):
        v = rec.get(k)
        if not v:
            continue
        try:
            return datetime.fromisoformat(str(v).replace("Z", "+00:00"))
        except ValueError:
            continue
    return None


# matches decides whether a record is wanted. The name test is deliberately loose --
# a pod name carries a replicaset hash nobody remembers -- so a service name matches
# every pod of that service across restarts, which is the usual question.
def matches(rec, name, pattern):
    if name:
        hay = f"{rec.get('pod','')} {rec.get('container','')} {rec.get('namespace','')}"
        if name not in hay:
            return False
    if pattern and not pattern.search(rec.get("payload") or rec.get("message") or ""):
        return False
    return True


# render_line prints one record. The payload is preferred over the raw message,
# because the raw form carries the CRI prefix the operator did not ask for.
def render_line(rec, show_pod):
    t = record_time(rec)
    ts = t.astimezone().strftime("%H:%M:%S") if t else "--:--:--"
    body = rec.get("payload") or rec.get("message") or ""
    pod = f"{rec.get('pod','?')[:34]:<34} " if show_pod else ""
    print(f"{ts}  {pod}{body.rstrip()}")


# scan streams one file, prefiltering on the raw line before parsing, because these
# files are gigabytes and json.loads on every line is the whole cost.
def scan(path, name, pattern, since, until, show_pod, limit, counter):
    with open(path, "r", errors="replace") as fh:
        for line in fh:
            if name and name not in line:
                continue
            try:
                rec = json.loads(line)
            except json.JSONDecodeError:
                continue
            if not matches(rec, name, pattern):
                continue
            t = record_time(rec)
            if since and (not t or t < since):
                continue
            if until and (not t or t > until):
                continue
            counter[0] += 1
            if limit and counter[0] > limit:
                return True
            if not counter[1]:
                render_line(rec, show_pod)
    return False


# main parses arguments and streams the matching days.
def main():
    ap = argparse.ArgumentParser(
        description="Read container logs from the host collection, including dead pods.")
    ap.add_argument("name", nargs="?", help="pod, container or namespace substring")
    ap.add_argument("--day", help="day to read, YYYY.MM.DD (default today)")
    ap.add_argument("--days", type=int, default=1, help="how many days back from --day")
    ap.add_argument("--since", help="start time, HH:MM local or an ISO timestamp")
    ap.add_argument("--until", help="end time, same forms")
    ap.add_argument("--grep", help="regex the payload must match")
    ap.add_argument("--count", action="store_true", help="count matches, print nothing")
    ap.add_argument("--limit", type=int, default=2000, help="stop after N (default 2000)")
    ap.add_argument("--list", action="store_true", help="show the retention window and exit")
    args = ap.parse_args()

    if not LOGDIR.is_dir():
        print(f"no collection at {LOGDIR}", file=sys.stderr)
        return 2

    if args.list:
        rows = available()
        print(f"{len(rows)} days collected at {LOGDIR}")
        for d, size in rows:
            print(f"  {d}   {size/1_048_576:.0f} MiB")
        print("\nA retention CronJob deletes at +7d. Older days are gone, not archived.")
        return 0

    end_day = args.day or datetime.now().strftime("%Y.%m.%d")
    try:
        end = datetime.strptime(end_day, "%Y.%m.%d")
    except ValueError:
        print("--day wants YYYY.MM.DD", file=sys.stderr)
        return 2
    start = end - timedelta(days=max(0, args.days - 1))
    files = day_files(days_in_range(start, end))
    if not files:
        print(f"no collected logs for {end_day}; try --list", file=sys.stderr)
        return 1

    since = parse_when(args.since, end_day)
    until = parse_when(args.until, end_day)
    pattern = re.compile(args.grep) if args.grep else None
    counter = [0, args.count]

    for p in files:
        if scan(p, args.name, pattern, since, until, True, args.limit, counter):
            print(f"\n... stopped at --limit {args.limit}; narrow with --since/--until",
                  file=sys.stderr)
            break

    if args.count:
        print(f"{counter[0]} matching records")
    elif counter[0] == 0:
        print("no matching records. An empty result is not evidence nothing happened:",
              file=sys.stderr)
        print("a service logging to a file inside its container appears here empty.",
              file=sys.stderr)
    return 0


if __name__ == "__main__":
    sys.exit(main())
