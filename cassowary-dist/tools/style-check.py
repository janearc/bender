#!/usr/bin/env python3
# style-check -- a pretooluse hook on what is about to be written.
#
# Reads one hook payload as json on stdin, inspects the text the tool is about
# to produce, and blocks it when a rule the evidence supports is broken. Three
# rules:
#
#   unbounded fault   a first-person fault claim that names nothing checkable.
#                     a claim with a referent, a file, a commit, a line, a
#                     quoted phrase, is ordinary error correction and passes.
#                     one without a referent cannot be checked, fixed or
#                     closed, so it accumulates, and it is refused. limit 0.
#   emphasis density  bold pairs and emphatic capitals per hundred words.
#                     documents in this estate run about 0.5 capitals per
#                     hundred words; drifted handoffs and peer messages ran
#                     2.2 to 7.2. the limit is 2.0 for each channel, applied
#                     only past a fifty word floor so a short note cannot trip
#                     on one word. rfc 2119 keywords and common acronyms are
#                     not counted.
#   session trailer   a claude-session line or session url in a commit
#                     message, which this estate has turned off. limit 0.
#
# Scope follows the file: comment lines only in source files, the whole text
# in markdown and in commit messages, nothing in formats with no prose. code
# fences and inline code are stripped before counting. the denial text states
# the count and the limit and stops.
#
# The regexes mirror fault-scan.py and style-metrics.py on purpose: the hook
# refuses what the corpus tools measure. it stays self-contained so it runs
# from any directory with nothing on the path.
#
# Blocks by exiting 2 with the counts on stderr; anything it does not
# understand passes with exit 0. this file installs nothing. the operator
# wires it into settings.json themselves, shaped like:
#
#   "hooks": {"PreToolUse": [{"matcher": "Write|Edit|MultiEdit|Bash",
#     "hooks": [{"type": "command",
#       "command": "python3 /Users/jane/mesh/dev/cassowary/tools/style-check.py"}]}]}

import json
import os
import re
import shlex
import sys

UNBOUNDED_LIMIT = 0
CAPS_LIMIT = 2.0
BOLD_LIMIT = 2.0
TRAILER_LIMIT = 0
WORD_FLOOR = 50

# a first-person subject attached to a fault word, within a short span
FAULT = re.compile(
    r"\b(i|my|i'm|i've|i'd)\b[^.\n]{0,60}?\b("
    r"wrong|mistaken|mistake|error|fault|failed|fail|broke|broken|"
    r"should have|shouldn't have|sorry|apolog|overstated|misread|"
    r"got that|missed|conflated|invented|fabricat)", re.I)

# something a reader could go and look at
REFERENT = re.compile(
    r"`[^`\n]+`"
    r"|\b[\w./-]+\.(?:md|go|ts|tsx|js|py|sh|json|jsonl|yaml|yml|sql|css|html)\b"
    r"|\b[0-9a-f]{7,40}\b"
    r"|\bline \d+|\b\d+\b"
    r"|\"[^\"\n]{3,}\""
    r"|\b(?:commit|file|function|test|endpoint|flag|field|column|table|branch|"
    r"issue|PR|pod|script|hook)\b", re.I)

# markdown emphasis, counted as pairs
BOLD = re.compile(r"\*\*[^*\n]+\*\*")

# all-caps words of three or more letters, minus working vocabulary
CAPS = re.compile(r"\b[A-Z]{3,}\b")
ACRONYMS = {
    "API", "URL", "URI", "HTTP", "HTTPS", "JSON", "YAML", "HTML", "CSS", "SQL",
    "CPU", "GPU", "RAM", "SSH", "TLS", "DNS", "TCP", "UDP", "PID", "UID", "GID",
    "CLI", "GUI", "SDK", "IDE", "CI", "PR", "MR", "VM", "OS", "IO", "UTC", "ISO",
    "RFC", "MUST", "SHALL", "SHOULD", "MAY", "NOT", "TODO", "FIXME", "NOTE",
    "README", "MD", "GET", "POST", "PUT", "DELETE", "HEAD", "OK", "ID", "IDS",
    "K8S", "AWS", "GCP", "DOC", "EOF", "STDIN", "STDOUT", "STDERR",
}

# the session line this estate has turned off, in either of its shapes
SESSION_TRAILER = re.compile(
    r"^claude-session:|claude\.ai/code/session_", re.I | re.M)

# comment markers by file extension. formats absent here carry no prose the
# hook should read, and are passed through
HASH_EXT = {".py", ".sh", ".bash", ".zsh", ".rb", ".yaml", ".yml", ".toml",
            ".tf", ".pl", ".r"}
SLASH_EXT = {".go", ".js", ".ts", ".tsx", ".jsx", ".rs", ".c", ".h", ".cc",
             ".cpp", ".hpp", ".java", ".kt", ".swift", ".proto"}
DASH_EXT = {".sql", ".lua", ".hs"}
PROSE_EXT = {".md", ".markdown", ".txt", ".rst"}


# strip fenced code blocks and inline code so examples do not skew the counts
def prose_only(text):
    text = re.sub(r"```.*?```", " ", text, flags=re.S)
    text = re.sub(r"`[^`\n]+`", " ", text)
    return text


# pull the comment text out of a source file, line comments plus c-style
# blocks. a marker mid-line counts only after whitespace, which keeps a url
# in a string literal from reading as a comment
def comment_text(text, marker):
    out = []
    esc = re.escape(marker)
    for line in text.splitlines():
        if line.lstrip().startswith("#!"):
            continue
        m = re.search(rf"(?:^|\s){esc}(.*)$", line)
        if m:
            out.append(m.group(1).strip())
    if marker == "//":
        out.extend(re.findall(r"/\*(.*?)\*/", text, re.S))
    return "\n".join(out)


# split prose into sentences, cheaply and good enough for counting
def sentences(text):
    return [s for s in re.split(r"(?<=[.!?])\s+|\n{2,}", text) if s.strip()]


# count fault claims with no referent in the claim or its neighbours. the
# artifact is often named in the sentence before the admission, so the window
# is three sentences wide
def unbounded_faults(text):
    sents = sentences(text)
    n = 0
    for i, s in enumerate(sents):
        if not FAULT.search(s):
            continue
        window = " ".join(sents[max(0, i - 1):i + 2])
        if not REFERENT.search(window):
            n += 1
    return n


# emphasis densities per hundred words, and the word count they rest on
def emphasis(text):
    words = re.findall(r"[A-Za-z']+", text)
    n = len(words) or 1
    caps = [c for c in CAPS.findall(text) if c not in ACRONYMS]
    return (round(100.0 * len(BOLD.findall(text)) / n, 2),
            round(100.0 * len(caps) / n, 2), len(words))


# commit message text from a bash command, or empty when the command is not a
# commit. covers -m and --message in their forms, -F files that already
# exist, and a heredoc body, which is how a long message usually arrives
def commit_message(command):
    if "commit" not in command:
        return ""
    try:
        toks = shlex.split(command)
    except ValueError:
        toks = command.split()
    if "git" not in toks or "commit" not in toks:
        return ""
    parts = []
    i = 0
    while i < len(toks):
        t = toks[i]
        if t in ("-m", "--message") or re.fullmatch(r"-[a-z]*m", t):
            if i + 1 < len(toks):
                parts.append(toks[i + 1])
                i += 1
        elif t.startswith("--message="):
            parts.append(t.split("=", 1)[1])
        elif t in ("-F", "--file") and i + 1 < len(toks):
            if os.path.isfile(toks[i + 1]):
                try:
                    parts.append(open(toks[i + 1], errors="replace").read())
                except OSError:
                    pass
            i += 1
        elif t.startswith("--file=") and os.path.isfile(t.split("=", 1)[1]):
            try:
                parts.append(open(t.split("=", 1)[1], errors="replace").read())
            except OSError:
                pass
        i += 1
    for _, body in re.findall(r"<<-?\s*'?(\w+)'?\n(.*?)\n\1", command, re.S):
        parts.append(body)
    return "\n\n".join(parts)


# what the payload is about to write, as (text, scope) pairs. scope is
# 'prose', 'comments' or 'commit'; a payload this function does not
# understand yields nothing and passes
def texts_of(tool, tool_input):
    if tool == "Bash":
        msg = commit_message(tool_input.get("command", ""))
        return [(msg, "commit")] if msg else []
    if tool == "Write":
        bodies = [tool_input.get("content", "")]
    elif tool in ("Edit", "MultiEdit"):
        bodies = [e.get("new_string", "") for e in tool_input.get("edits", [])]
        if tool_input.get("new_string"):
            bodies.append(tool_input["new_string"])
    else:
        return []
    ext = os.path.splitext(tool_input.get("file_path", ""))[1].lower()
    if ext in PROSE_EXT:
        return [(b, "prose") for b in bodies]
    for exts, marker in ((HASH_EXT, "#"), (SLASH_EXT, "//"), (DASH_EXT, "--")):
        if ext in exts:
            return [(comment_text(b, marker), "comments") for b in bodies]
    return []


# apply every rule that fits the scope, returning denial lines
def check(text, scope):
    out = []
    body = prose_only(text)
    faults = unbounded_faults(body)
    if faults > UNBOUNDED_LIMIT:
        out.append(f"unbounded fault claims {faults}, limit {UNBOUNDED_LIMIT}")
    bold, caps, words = emphasis(body)
    if words >= WORD_FLOOR:
        if caps > CAPS_LIMIT:
            out.append(f"capitals {caps} per 100 words, limit {CAPS_LIMIT}")
        if bold > BOLD_LIMIT:
            out.append(f"bold {bold} per 100 words, limit {BOLD_LIMIT}")
    if scope == "commit":
        trailers = sum(1 for ln in text.splitlines() if SESSION_TRAILER.search(ln))
        if trailers > TRAILER_LIMIT:
            out.append(f"session trailer lines {trailers}, limit {TRAILER_LIMIT}")
    return out


# read the payload, check it, exit 2 with counts when a rule is broken
def main():
    try:
        payload = json.load(sys.stdin)
    except (json.JSONDecodeError, ValueError):
        return 0
    tool = payload.get("tool_name", "")
    tool_input = payload.get("tool_input") or {}
    denials = []
    for text, scope in texts_of(tool, tool_input):
        denials.extend(check(text, scope))
    if denials:
        for d in denials:
            print(f"style-check: {d}", file=sys.stderr)
        return 2
    return 0


if __name__ == "__main__":
    sys.exit(main())
