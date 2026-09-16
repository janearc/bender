#!/usr/bin/env python3
# style-metrics -- count the surface features that carry emotional valence.
#
# Three channels, all countable without a model: typographic emphasis,
# capitalisation, and vocabulary. Plus first-person self-assessment, which is
# the specific thing that propagates through handoffs.
#
#   style-metrics.py FILE [FILE...]
#
# Prints one row per file, densities per 100 words. Reads files, prints no
# content beyond matched single words in the vocabulary breakdown.

import re
import sys
from pathlib import Path

# markdown emphasis. bold is counted as pairs, not markers
BOLD = re.compile(r"\*\*[^*\n]+\*\*")
ITAL = re.compile(r"(?<!\*)\*[^*\n]+\*(?!\*)|(?<!_)_[^_\n]+_(?!_)")

# all-caps words used for emphasis. identifiers and short acronyms are excluded
# by requiring three or more letters and no digits or underscores
CAPS = re.compile(r"\b[A-Z]{3,}\b")
ACRONYMS = {
    "API", "URL", "URI", "HTTP", "HTTPS", "JSON", "YAML", "HTML", "CSS", "SQL",
    "CPU", "GPU", "RAM", "SSH", "TLS", "DNS", "TCP", "UDP", "PID", "UID", "GID",
    "CLI", "GUI", "SDK", "IDE", "CI", "PR", "MR", "VM", "OS", "IO", "UTC", "ISO",
    "RFC", "MUST", "SHALL", "SHOULD", "MAY", "NOT", "TODO", "FIXME", "NOTE",
    "README", "MD", "GET", "POST", "PUT", "DELETE", "HEAD", "OK", "ID", "IDS",
    "K8S", "AWS", "GCP", "DOC", "EOF", "STDIN", "STDOUT", "STDERR",
}

# intensifiers and absolutes. the register signal is density, not any one word
INTENSIFIER = {
    "critical", "crucial", "essential", "vital", "severe", "catastrophic",
    "disastrous", "dangerous", "alarming", "urgent", "dire", "grave",
    "completely", "entirely", "utterly", "absolutely", "totally", "wholly",
    "never", "always", "every", "all", "none", "nothing", "everything",
    "must", "cannot", "impossible", "unacceptable", "worthless", "useless",
    "broken", "wrong", "failure", "failed", "fails", "disease", "rot",
    "deeply", "profoundly", "fundamentally", "systematically", "pervasive",
}

# first person singular, the grammatical basis of self-assessment
FIRST_PERSON = re.compile(r"\b(i|i'm|i've|i'd|i'll|me|my|myself|mine)\b", re.I)

# explicit self-assessment: a first-person subject attached to a fault
SELF_FAULT = re.compile(
    r"\b(i|my)\b[^.\n]{0,40}\b(wrong|failed|fail|error|mistake|fault|"
    r"should have|shouldn't have|didn't|broke|broken|sorry|apolog)", re.I)

# a heading that frames the document around what went wrong
FAULT_HEADING = re.compile(
    r"^#{1,6} .*\b(fail|failed|failure|wrong|mistake|error|problem|"
    r"went wrong|disease|apolog)\b.*$", re.I | re.M)


# strip fenced code blocks and inline code so examples do not skew the counts
def prose_only(text):
    text = re.sub(r"```.*?```", " ", text, flags=re.S)
    text = re.sub(r"`[^`\n]+`", " ", text)
    return text


# every metric for one file, as counts and as density per 100 words
def measure(path):
    raw = Path(path).read_text(errors="replace")
    text = prose_only(raw)
    words = re.findall(r"[A-Za-z']+", text)
    n = len(words) or 1
    caps = [c for c in CAPS.findall(text) if c not in ACRONYMS]
    intens = [w for w in (x.lower() for x in words) if w in INTENSIFIER]
    per100 = lambda k: round(100.0 * k / n, 2)
    return {
        "file": Path(path).name,
        "words": n,
        "bold": per100(len(BOLD.findall(text))),
        "ital": per100(len(ITAL.findall(text))),
        "caps": per100(len(caps)),
        "intens": per100(len(intens)),
        "first": per100(len(FIRST_PERSON.findall(text))),
        "fault": len(SELF_FAULT.findall(text)),
        "headings": len(FAULT_HEADING.findall(raw)),
        "caps_words": sorted(set(caps))[:8],
    }


def main():
    if len(sys.argv) < 2:
        print("usage: style-metrics.py FILE [FILE...]", file=sys.stderr)
        return 2
    rows = [measure(p) for p in sys.argv[1:]]
    hdr = f"{'file':<32}{'words':>7}{'bold':>7}{'ital':>7}{'caps':>7}{'intens':>8}{'first':>7}{'fault':>7}{'hdgs':>6}"
    print(hdr)
    print("-" * len(hdr))
    for r in rows:
        print(f"{r['file']:<32}{r['words']:>7}{r['bold']:>7}{r['ital']:>7}"
              f"{r['caps']:>7}{r['intens']:>8}{r['first']:>7}{r['fault']:>7}{r['headings']:>6}")
    print("\ndensities are per 100 words; fault and hdgs are raw counts")
    for r in rows:
        if r["caps_words"]:
            print(f"  {r['file']}: caps {' '.join(r['caps_words'])}")
    return 0


if __name__ == "__main__":
    sys.exit(main())
