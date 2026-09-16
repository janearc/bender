#!/usr/bin/env python3
# phrase-recur -- find distinctive phrases that appear in more than one document.
#
# The propagation signal is a phrase that is not a technical term recurring
# across documents written at different times. Code, identifiers and paths are
# stripped first so that shared vocabulary about the same software does not
# read as shared rhetoric.
#
#   phrase-recur.py FILE [FILE...]

import re
import sys
from collections import defaultdict
from pathlib import Path

STOP_ONLY = re.compile(r"^(?:the|a|an|and|or|of|to|in|is|it|that|this|for|on|as|be|by|with|at|from|but|not|if|so|we|you|i|they|he|she)(?:\s|$)")


# remove code, identifiers, paths and numbers so only prose remains
def prose(text):
    text = re.sub(r"```.*?```", " ", text, flags=re.S)
    text = re.sub(r"`[^`\n]+`", " ", text)
    text = re.sub(r"https?://\S+|[~./][\w./-]+|\b[\w-]+\.(?:md|go|ts|py|js|sh|json|yaml)\b", " ", text)
    text = re.sub(r"\b\w*[_/#]\w*\b|\b\d[\w.]*\b", " ", text)
    return re.sub(r"[^a-z' ]+", " ", text.lower())


# every n-word window, as a normalised phrase
def ngrams(words, n):
    return [" ".join(words[i:i + n]) for i in range(len(words) - n + 1)]


def main():
    files = [Path(p) for p in sys.argv[1:]]
    if len(files) < 2:
        print("usage: phrase-recur.py FILE FILE [FILE...]", file=sys.stderr)
        return 2

    seen = defaultdict(set)
    for f in files:
        words = prose(f.read_text(errors="replace")).split()
        for n in (4, 5, 6):
            for g in set(ngrams(words, n)):
                seen[g].add(f.name)

    shared = {g: fs for g, fs in seen.items() if len(fs) >= 2 and not STOP_ONLY.match(g)}
    # drop a phrase wholly contained in a longer shared phrase from the same set
    longer = sorted(shared, key=len, reverse=True)
    keep = []
    for g in longer:
        if not any(g in k and shared[g] <= shared[k] for k in keep):
            keep.append(g)

    print(f"{len(files)} documents, {len(keep)} distinct shared phrases (4-6 words)\n")
    for g in sorted(keep, key=lambda x: (-len(shared[x]), -len(x.split()))):
        names = " ".join(sorted(n.split("-")[0] for n in shared[g]))
        print(f"  [{len(shared[g])}]  {g!r}   {names}")
    return 0


if __name__ == "__main__":
    sys.exit(main())
