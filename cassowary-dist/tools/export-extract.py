#!/usr/bin/env python3
# export-extract -- turn a claude.ai data export into shapes the analysis tools read.
#
# The export is one enormous single-line JSON array. This flattens it into one
# pretty-printed document per conversation, and optionally into the same
# two-file split that session-extract produces, so style-metrics, phrase-recur
# and the rest work on it unchanged.
#
#   export-extract.py conversations.json OUTDIR [--markdown]
#
# Prints counts only. No message text reaches stdout.

import json
import re
import sys
from pathlib import Path


# the message body, preferring the top-level text and falling back to the
# content blocks, which carry the same string in the exports seen so far
def body_of(msg):
    t = (msg.get("text") or "").strip()
    if t:
        return t
    parts = []
    for b in msg.get("content") or []:
        if isinstance(b, dict) and b.get("type") == "text" and b.get("text"):
            parts.append(b["text"])
    return "\n".join(parts).strip()


# the non-prose blocks: tool calls, their results, and whatever else the export
# carries. these are what an assertion can be checked against, so they are kept
# in full rather than counted
def actions_of(msg):
    out = []
    for b in msg.get("content") or []:
        if not isinstance(b, dict):
            continue
        k = b.get("type")
        if k == "tool_use":
            out.append({"kind": "tool_use", "name": b.get("name"),
                        "input": b.get("input"), "id": b.get("id")})
        elif k == "tool_result":
            out.append({"kind": "tool_result", "for": b.get("tool_use_id"),
                        "is_error": b.get("is_error"), "content": b.get("content")})
        elif k in ("flag", "token_budget"):
            out.append({"kind": k, "block": b})
    return out


# thinking is exported hollow, with only the summaries populated; keep those
# because they are the only trace of it that survives
def summaries_of(msg):
    out = []
    for b in msg.get("content") or []:
        if isinstance(b, dict) and b.get("type") == "thinking":
            for s in b.get("summaries") or []:
                if isinstance(s, dict) and s.get("summary"):
                    out.append(s["summary"])
    return out


# a filesystem-safe stem: date, short uuid, and a slug of the title
def stem_for(conv):
    date = (conv.get("created_at") or "")[:10] or "undated"
    uid = (conv.get("uuid") or "")[:8] or "nouuid"
    slug = re.sub(r"[^a-z0-9]+", "-", (conv.get("name") or "").lower()).strip("-")[:48]
    return f"{date}-{uid}" + (f"-{slug}" if slug else "")


# normalise one conversation into a flat, ordered, pretty-printable record
def normalise(conv):
    msgs = []
    for i, m in enumerate(conv.get("chat_messages") or []):
        msgs.append({
            "i": i,
            "sender": m.get("sender"),
            "created_at": m.get("created_at"),
            "uuid": m.get("uuid"),
            "text": body_of(m),
            "thinking_summaries": summaries_of(m),
            "actions": actions_of(m),
            "n_attachments": len(m.get("attachments") or []),
            "n_files": len(m.get("files") or []),
        })
    return {
        "uuid": conv.get("uuid"),
        "name": conv.get("name"),
        "created_at": conv.get("created_at"),
        "updated_at": conv.get("updated_at"),
        "n_messages": len(msgs),
        "messages": msgs,
    }


# the two-file split, matching what session-extract writes
def write_markdown(outdir, rec):
    outdir.mkdir(parents=True, exist_ok=True)
    for who, fname, heading in (("human", "your-turns.md", "Typed turns"),
                                ("assistant", "assistant-prose.md", "Assistant prose")):
        with open(outdir / fname, "w") as fh:
            fh.write(f"# {heading}\n\n")
            for m in rec["messages"]:
                if m["sender"] != who or not m["text"]:
                    continue
                fh.write(f"## {m['created_at']}\n\n{m['text'].rstrip()}\n\n")


# count how many words each side contributed, for the manifest
def words(rec, who):
    return sum(len(re.findall(r"[A-Za-z']+", m["text"]))
               for m in rec["messages"] if m["sender"] == who)


def main():
    args = [a for a in sys.argv[1:] if not a.startswith("--")]
    want_md = "--markdown" in sys.argv
    if len(args) < 2:
        print("usage: export-extract.py conversations.json OUTDIR [--markdown]", file=sys.stderr)
        return 2

    src, out = Path(args[0]), Path(args[1])
    data = json.loads(src.read_text(errors="replace"))
    if not isinstance(data, list):
        data = [data]

    (out / "json").mkdir(parents=True, exist_ok=True)
    kinds, senders = {}, {}
    rows = []

    for conv in data:
        for m in conv.get("chat_messages") or []:
            senders[m.get("sender")] = senders.get(m.get("sender"), 0) + 1
            for b in m.get("content") or []:
                if isinstance(b, dict):
                    k = b.get("type")
                    kinds[k] = kinds.get(k, 0) + 1
        rec = normalise(conv)
        stem = stem_for(conv)
        (out / "json" / f"{stem}.json").write_text(json.dumps(rec, indent=2, ensure_ascii=False))
        if want_md:
            write_markdown(out / "md" / stem, rec)
        n_actions = sum(len(m["actions"]) for m in rec["messages"])
        rows.append((stem, rec["created_at"], rec["n_messages"],
                     words(rec, "human"), words(rec, "assistant"), n_actions))

    rows.sort(key=lambda r: r[1] or "")
    with open(out / "manifest.tsv", "w") as fh:
        fh.write("stem\tcreated_at\tmessages\thuman_words\tassistant_words\tactions\n")
        for r in rows:
            fh.write("\t".join(str(x) for x in r) + "\n")

    print(f"conversations   {len(rows)}")
    print(f"messages        {sum(r[2] for r in rows)}")
    print(f"human words     {sum(r[3] for r in rows)}")
    print(f"assistant words {sum(r[4] for r in rows)}")
    print(f"action blocks   {sum(r[5] for r in rows)}")
    print(f"senders         {senders}")
    print(f"content types   {kinds}")
    print(f"date range      {rows[0][1]} -> {rows[-1][1]}" if rows else "empty")
    print(f"written to      {out}")
    return 0


if __name__ == "__main__":
    sys.exit(main())
