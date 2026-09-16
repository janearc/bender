#!/usr/bin/env python3
# exposure-report.py -- what the software does, against what is published.
#
# what it is for. Two halves. The patterns are designs known to work against
# the person using software, with what can be counted about each. --measure
# runs the counters over the session transcripts on this machine and puts the
# result next to the pattern it belongs to. Without --measure it is the
# reference: the patterns, the sources, and what each source cannot argue.
#
# the number to read first is how few thresholds there are. One pattern of
# eighteen carries a line from a randomised experiment. Every measured count
# below that line is a place to look and not a verdict, and the report says so
# on each one rather than in a preamble nobody reaches.
#
# what is not in it. No session data, no dates, no hours, no operator. This
# report is about the sources and nothing else. If a number about a person ever
# appears on this page, something has gone wrong upstream of it.
#
# THEMES are corrected, not as designed. Both palettes come from dodo, and
# neither passes the reference-dark-2026-09-03 readability profile as it ships:
# corvid's ink glares at 16.39:1 where the band ends at 12.5, and its dim text
# sits at 3.58:1 where the band starts at 10. The values below are what
# `libreadme fix` produced. The type rules are the profile's own.
#
# four FORMATS, and one of them is conditional. html and pdf are the report;
# csv and tsv are the same rows for a spreadsheet or a pipe. PDF needs an
# external renderer and this does not vendor one: if prince is not installed,
# --format pdf exits non-zero and says so rather than quietly writing the html
# and letting you find out later that the pdf you asked for is not there.
#
# The theme applies to html and pdf only. csv and tsv carry the same rows
# whichever theme is named, so asking for both themes in a tabular format
# writes one file rather than two identical ones.
#
#   standards-report.py [--theme corvid|vaporwave] [--format html] [-o PATH]
#   standards-report.py --all OUTDIR          every theme, every format
#   standards-report.py --format csv,tsv -o -  tabular, to stdout

import argparse
import csv
import html
import shutil
import subprocess
import sys
from datetime import date
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent.parent / "lib"))

from exposure import (ALL, DARK_PATTERNS, DISSENT, dissent_strength, OCCUPATIONAL,
                      PARTIAL, RECALLED, SLEEP, VERIFIED, patterns)
from exposure import detect

# Palettes as dodo defines them, with the corrections libreadme computed
# against the reference profile. Keeping both themes in one dict rather than
# two files means a new theme is one entry and cannot forget a token.
THEMES = {
    "corvid": {
        "ground": "#0b0e14", "pane": "rgba(20, 26, 36, 0.78)",
        "ink": "#b8cce8", "dim": "#c6cbd4", "accent": "#6fa8ff",
        "green": "#3fb950", "red": "#e65c54", "edge": "rgba(184,204,232,.16)",
    },
    "vaporwave": {
        "ground": "#140f1f", "pane": "rgba(28, 20, 44, 0.90)",
        "ink": "#ddbeff", "dim": "#d1c6e4", "accent": "#ff7ad9",
        "green": "#5fd38a", "red": "#ff6b6b", "edge": "rgba(221,190,255,.18)",
    },
}

# The profile's type rules, written once. Nothing under 16px, no all-caps for
# prose because capitals remove the word shape a reader navigates by, and
# monospace reserved for identifiers for the same reason.
TYPE_CSS = """
 * { box-sizing:border-box; text-transform:none }
 body { margin:0; background:var(--ground); color:var(--ink);
   font-family:system-ui,-apple-system,'Segoe UI',sans-serif;
   font-size:24px; font-weight:400; line-height:1.75; letter-spacing:.015em;
   font-variant-ligatures:none; padding:40px 34px 80px }
 .wrap { max-width:1180px; margin:0 auto }
 h1 { font-size:34px; font-weight:600; letter-spacing:.01em; margin:0 0 10px }
 h2 { font-size:27px; font-weight:600; letter-spacing:.01em;
   margin:56px 0 6px; padding-bottom:10px;
   border-bottom:1px solid var(--edge) }
 h3 { font-size:24px; font-weight:600; margin:0 0 4px }
 p { margin:0 0 16px; max-width:74ch }
 .lede { color:var(--dim); font-size:22px; max-width:74ch }
 .sub { color:var(--dim); font-size:18px; margin:0 0 34px }
 .note { color:var(--dim); font-size:20px; max-width:74ch; margin:0 0 22px }
 code, .mono { font-family:ui-monospace,SFMono-Regular,Menlo,monospace;
   font-size:20px; letter-spacing:0 }
 .card { background:var(--pane); border:1px solid var(--edge);
   border-radius:10px; padding:24px 26px; margin:0 0 18px }
 .card.against { border-left:5px solid var(--accent) }
 .key { color:var(--accent) }
 .row { display:flex; flex-wrap:wrap; gap:10px; align-items:center;
   margin:0 0 14px }
 .chip { font-size:16px; line-height:1.5; padding:3px 11px; border-radius:999px;
   border:1px solid var(--edge); color:var(--dim) }
 .chip.ok { color:var(--green); border-color:var(--green) }
 .chip.part { color:var(--accent); border-color:var(--accent) }
 .chip.no { color:var(--red); border-color:var(--red) }
 .chip.against { color:var(--red); border-color:var(--red) }
 .thr { color:var(--ink); font-size:20px }
 .label { color:var(--dim); font-size:18px; margin:16px 0 2px }
 .claim { margin:0 0 4px }
 .src { color:var(--dim); font-size:20px; margin:0 }
 a { color:var(--accent) }
 .counts { display:flex; flex-wrap:wrap; gap:26px; margin:0 0 10px;
   padding:20px 26px; background:var(--pane); border:1px solid var(--edge);
   border-radius:10px }
 .counts div { font-size:20px; color:var(--dim) }
 .why { color:var(--dim); font-size:19px; margin:0 0 14px;
   font-style:italic }
 .chart { display:block; width:100%; height:auto; margin:14px 0 6px }
 .counts b { display:block; font-size:30px; font-weight:600; color:var(--ink);
   font-variant-numeric:tabular-nums; letter-spacing:0 }
 footer { color:var(--dim); font-size:18px; margin:60px 0 0;
   border-top:1px solid var(--edge); padding-top:18px }
"""

# Print rules, applied by the pdf path and harmless on screen. A citation split
# across a page break is the one thing that would make this document worse than
# the terminal output it replaces, so a card never breaks.
PRINT_CSS = """
 @media print {
   @page { size:A4 portrait; margin:18mm 16mm 20mm }
   body { background:#fff; color:#111; font-size:11pt; padding:0 }
   h1 { font-size:20pt } h2 { font-size:14pt } h3 { font-size:11.5pt }
   .lede { font-size:11pt; color:#333 } .sub, .note, .src { color:#333 }
   .card { background:#fff; border:1pt solid #bbb; break-inside:avoid;
     page-break-inside:avoid }
   .card.against { border-left:3pt solid #444 }
   h2 { break-after:avoid; page-break-after:avoid }
   .counts { background:#fff; border:1pt solid #bbb; break-inside:avoid }
   .chip { border-color:#999; color:#333 }
   .key, a { color:#111 } a { text-decoration:none }
   code, .mono { font-size:10pt }
 }
"""

STATUS_CHIP = {VERIFIED: ("ok", "read at source"),
               PARTIAL: ("part", "partly checked"),
               RECALLED: ("no", "not read")}
# Transfer is the field a reader of this page most needs to see without
# hunting, so it gets a colour: far means somebody argued the transfer rather
# than measuring it.
TRANSFER_CHIP = {"near": "ok", "adjacent": "part", "far": "no"}


# esc keeps the source text intact through the render. Every field in the
# library is prose written for a person, so it is escaped rather than trusted.
def esc(s):
    return html.escape(s or "", quote=False)


# card renders one standard. The limits field is given the same weight as the
# claim on purpose: a threshold quoted without its limits is the failure this
# library exists to prevent, so the page must not make it easy to read one and
# skip the other.
def card(s):
    scls, stext = STATUS_CHIP[s.status]
    chips = ['<span class="chip %s">%s%s</span>'
             % (scls, esc(stext), " " + s.checked if s.checked else "")]
    chips.append('<span class="chip">%s</span>' % esc(s.evidence))
    chips.append('<span class="chip">%s</span>' % esc(s.interest))
    if s in DISSENT:
        chips.append('<span class="chip against">disputes</span>')
    if s.threshold:
        chips.append('<span class="chip"><span class="thr">%s %s</span></span>'
                     % (esc(str(s.threshold)), esc(s.unit or "")))
    link = ('<p class="src"><a href="%s">%s</a></p>'
            % (esc(s.url), esc(s.url))) if s.url else ""
    return (
        '<div class="card%s">'
        '<h3><span class="key mono">%s</span> &middot; %s</h3>'
        '<div class="row">%s</div>'
        '<p class="claim">%s</p>'
        '<p class="label">source</p><p class="src">%s</p>%s'
        '<p class="label">what it cannot be used to argue</p>'
        '<p class="src">%s</p>'
        '</div>'
    ) % (" against" if s in DISSENT else "",
         esc(s.key), esc(s.domain), "".join(chips),
         esc(s.claim), esc(s.source), link, esc(s.limits))


# svg_profile draws one detector's daily rate across the corpus. Inline SVG
# with theme tokens rather than an image, so the chart themes with the page and
# the file stays one document with nothing to fetch.
#
# the Y axis is per-chart, not shared. These rates differ by two orders of
# magnitude, so a common scale would flatten six of the eight into a flat line.
# The peak is printed on the axis so nobody reads height across charts.
def svg_profile(m, w=1020, h=120, pad=26):
    pts = m.daily_rates()
    if not pts:
        return ('<p class="src"><em>no daily series: this pattern is counted '
                'per session, not per turn</em></p>')
    top = max(r for _, r in pts) or 1.0
    n = len(pts)
    bw = max((w - pad * 2) / n - 2, 1.0)
    bars = []
    for i, (d, r) in enumerate(pts):
        x = pad + i * ((w - pad * 2) / n)
        bh = (r / top) * (h - pad * 2)
        bars.append(
            '<rect x="%.1f" y="%.1f" width="%.1f" height="%.1f" rx="1.5" '
            'fill="var(--accent)" opacity="%.2f"><title>%s  %.2f%%</title>'
            '</rect>' % (x, h - pad - bh, bw, max(bh, 0.6),
                         0.45 + 0.55 * (r / top), esc(d), r * 100))
    mean = m.rate
    my = h - pad - (mean / top) * (h - pad - pad)
    return (
        '<svg class="chart" viewBox="0 0 %d %d" role="img" '
        'aria-label="daily rate for %s">'
        '<line x1="%d" y1="%d" x2="%d" y2="%d" stroke="var(--edge)" '
        'stroke-width="1"/>'
        '%s'
        '<line x1="%d" y1="%.1f" x2="%d" y2="%.1f" stroke="var(--dim)" '
        'stroke-dasharray="4 4" stroke-width="1"/>'
        '<text x="%d" y="14" fill="var(--dim)" font-size="13">peak %.2f%% '
        'per day</text>'
        '<text x="%d" y="%.1f" fill="var(--dim)" font-size="13">mean %.2f%%'
        '</text>'
        '<text x="%d" y="%d" fill="var(--dim)" font-size="13">%s</text>'
        '<text x="%d" y="%d" fill="var(--dim)" font-size="13" '
        'text-anchor="end">%s</text>'
        '</svg>'
    ) % (w, h, esc(m.key),
         pad, h - pad, w - pad, h - pad,
         "".join(bars),
         pad, my, w - pad, my,
         pad, top * 100,
         pad + 4, max(my - 5, 12), mean * 100,
         pad, h - 8, esc(pts[0][0]),
         w - pad, h - 8, esc(pts[-1][0]))


# pcard renders one pattern. Threshold and look-for are given their own
# labelled rows even when empty, because "none published" is the finding here
# and a field that vanishes when empty reads as an oversight instead.
def pcard(p, measured=None, why=None):
    cls = TRANSFER_CHIP[p.transfer]
    srcs = ", ".join(p.backing)
    m = (measured or {}).get(p.key)
    meas = ""
    if m:
        meas = ('<p class="label">measured across the corpus</p>'
                '<p class="claim"><b class="thr">%s</b> of %s %s '
                '&mdash; %s, appearing in %s of sessions</p>'
                '%s<p class="src">%s</p><p class="src">%s</p>'
                % (f"{m.value:,}", f"{m.of:,}", esc(m.unit),
                   f"{m.rate:.2%}", f"{m.spread:.1%}",
                   svg_profile(m), esc(m.note),
                   esc(m.concentration.note())))
    return (
        '<div class="card%s">'
        '<h3><span class="key mono">%s</span> &middot; %s</h3>'
        '<div class="row">'
        '<span class="chip %s">%s</span>'
        '<span class="chip">%s</span>%s</div>'
        '<p class="why">%s</p>'
        '<p class="claim">%s</p>'
        '<p class="label">in a text or conversational interface</p>'
        '<p class="src">%s</p>'
        '<p class="label">what you could count</p><p class="src">%s</p>'
        '<p class="label">threshold</p><p class="src">%s</p>'
        '%s'
        '<p class="label">backing</p><p class="src mono">%s</p>'
        '</div>'
    ) % ("" if p.threshold else " against",
         esc(p.key), esc(p.name), cls, esc(p.transfer), esc(p.family),
         '<span class="chip ok">has a threshold</span>' if p.threshold else "",
         esc(why or ""),
         esc(p.in_software),
         esc(p.in_text) if p.in_text else "<em>does not transfer</em>",
         esc(p.look_for) if p.look_for
         else "<em>nothing measurable proposed</em>",
         esc(p.threshold) if p.threshold else "<em>none published</em>",
         meas, esc(srcs))


# psection groups patterns by the taxonomy that named them, in the order the
# families are declared, so the page never mixes two taxonomies under one
# heading.
def tiers(measured=None):
    rows = patterns.ordered(measured)
    out, current = [], None
    for p, tier, why in rows:
        if tier != current:
            current = tier
            out.append('<h2>%d &middot; %s</h2>'
                       % (tier + 1, esc(patterns.TIER_LABELS[tier])))
        out.append(pcard(p, measured, why))
    return "".join(out)


# section renders a titled group with its own standfirst, because a reader who
# lands mid-page needs to know why these entries are together.
def section(title, blurb, items):
    return ('<h2>%s</h2><p class="note">%s</p>%s'
            % (esc(title), esc(blurb), "".join(card(s) for s in items)))


# counts is the summary strip. It reports the dissent as a first-class figure
# rather than a footnote: it is the number that says whether the rest can be
# trusted, so it sits with the others and not below them.
def counts():
    ver = sum(1 for s in ALL if s.status == VERIFIED)
    part = sum(1 for s in ALL if s.status == PARTIAL)
    rec = sum(1 for s in ALL if s.status == RECALLED)
    against = len(DISSENT)
    cells = [("patterns", len(patterns.ALL)),
             ("with something to count", len(patterns.measurable())),
             ("with a published threshold", len(patterns.with_threshold())),
             ("standards behind them", len(ALL)),
             ("read at source", ver), ("partly checked", part),
             ("not read", rec),
             ("disputing (%.0f%% of the library)" % (100 * dissent_strength()),
              against)]
    return ('<div class="counts">%s</div>'
            % "".join('<div><b>%d</b>%s</div>' % (n, esc(t))
                      for t, n in cells))


# COLUMNS is the tabular contract. Written out rather than derived from the
# object so that adding a field to Standard does not silently change every
# spreadsheet anyone has already built on this output.
COLUMNS = ["key", "domain", "evidence", "interest", "disputes", "status",
           "checked", "threshold", "unit", "claim", "source", "url", "limits"]

# Patterns get their own table rather than sharing one. They answer a different
# question and a merged sheet would need half its cells empty in every row,
# which is how a spreadsheet stops being read.
PCOLUMNS = ["key", "name", "family", "transfer", "measurable", "threshold",
            "in_software", "in_text", "look_for", "backing"]


# rows flattens the library for csv and tsv. Every field a reader needs to
# judge a citation is present, including limits: a table that carried the
# thresholds and dropped what they cannot argue would be the most quotable and
# least honest artifact this repository produces.
def rows():
    out = []
    for s in ALL:
        out.append({"key": s.key, "domain": s.domain,
                    "evidence": s.evidence, "interest": s.interest,
                    "disputes": "yes" if s in DISSENT else "",
                    "status": s.status, "checked": s.checked or "",
                    "threshold": s.threshold if s.threshold else "",
                    "unit": s.unit or "", "claim": s.claim,
                    "source": s.source, "url": s.url, "limits": s.limits})
    return out


# tabular writes csv or tsv. QUOTE_ALL because the claim and limits fields are
# prose containing commas and colons, and a reader importing this should not
# have to think about the delimiter at all.
def prows():
    out = []
    for p in patterns.ALL:
        out.append({"key": p.key, "name": p.name, "family": p.family,
                    "transfer": p.transfer,
                    "measurable": "yes" if p.measurable() else "no",
                    "threshold": p.threshold or "",
                    "in_software": p.in_software, "in_text": p.in_text or "",
                    "look_for": p.look_for or "", "backing": " ".join(p.backing)})
    return out


def tabular(fh, delimiter, which="standards"):
    cols = PCOLUMNS if which == "patterns" else COLUMNS
    data = prows() if which == "patterns" else rows()
    w = csv.DictWriter(fh, fieldnames=cols, delimiter=delimiter,
                       quoting=csv.QUOTE_ALL, lineterminator="\n")
    w.writeheader()
    for r in data:
        w.writerow(r)


# to_pdf renders through prince, which is the only renderer this checks for --
# it is what the estate already uses and what the print rules above were
# written against. Missing renderer is a hard failure with the reason named,
# because a report that silently is not the format you asked for is worse than
# no report.
def to_pdf(html_text, dest):
    prince = shutil.which("prince")
    if not prince:
        raise RuntimeError(
            "pdf needs prince and it is not on PATH. Install it "
            "(brew install --cask prince) or ask for --format html. "
            "Nothing was written.")
    src_html = dest.with_suffix(".prince-input.html")
    src_html.write_text(html_text)
    try:
        r = subprocess.run([prince, str(src_html), "-o", str(dest)],
                           capture_output=True, text=True)
        if r.returncode != 0 or not dest.exists():
            raise RuntimeError("prince failed: %s"
                               % (r.stderr.strip() or "no output"))
    finally:
        src_html.unlink(missing_ok=True)
    return dest


# render builds the whole page for one theme. The theme only reaches the CSS
# variables; nothing in the content changes with it, so the two pages are the
# same report and a difference between them would be a bug.
def render(theme, measured=None):
    t = THEMES[theme]
    tokens = "\n".join(" --%s:%s;" % (k, v) for k, v in t.items())
    return (
        '<!doctype html><html lang="en" data-theme="%s"><meta charset="utf-8">'
        '<meta name="viewport" content="width=device-width,initial-scale=1">'
        '<title>exposure -- thresholds and their sources</title>'
        '<style>:root {\n%s\n}\n%s\n%s</style><body><div class="wrap">'
        '<h1>exposure</h1>'
        '<p class="sub mono">%s theme &middot; generated %s &middot; '
        'cassowary/lib/exposure</p>'
        '<p class="lede">Every published threshold this kit measures against, '
        'with the source for it, the claim in the source&rsquo;s own words, '
        'and what it cannot be used to argue. A number without the sentence it '
        'came from becomes somebody&rsquo;s opinion within a month and then '
        'gets adjusted to fit.</p>'
        '<p class="note">This is not a diagnostic and cannot be used as one. '
        'Every threshold here is a population figure or a legal entitlement, '
        'and neither transfers to a person. No session data appears on this '
        'page.</p>'
        '%s'
        '<h2>What to look for</h2><p class="note">%s</p>'
        '%s'
        '<h2>The evidence behind them</h2><p class="note">%s</p>'
        '%s%s%s%s'
        '<footer>Generated by <span class="mono">cassowary/tools/'
        'standards-report.py</span>. Palette from dodo, corrected by '
        '<span class="mono">libreadme fix</span> against '
        '<span class="mono">reference-dark-2026-09-03</span>: neither theme '
        'passes that profile as designed.</footer>'
        '</div></body></html>'
    ) % (theme, tokens, TYPE_CSS, PRINT_CSS, theme,
         date.today().isoformat(), counts(),
         ("Counted over the session transcripts on this machine. " if measured
          else "") +
         "Designs that work against the person using the software, and what "
         "if anything can be counted about each. Of the %d below, %d have "
         "something countable proposed and %d carries a threshold from a "
         "randomised experiment. A pattern reading none published is not a "
         "gap waiting to be filled by guessing -- it is the finding. Nearly "
         "all of this literature studied visual interfaces, so each pattern "
         "records how far it sits from a text one: far means somebody argued "
         "the transfer rather than measuring it. Ordered most significant "
         "first, by a stated rule rather than a score: what can be compared "
         "to a published line, then what was counted and found, then what was "
         "counted and came back null, then what applies here with no counter "
         "written, then what does not transfer at all. Each entry says why it "
         "sits where it does. There is no measured quantity that makes one of "
         "these more significant than another, so the ordering is an argument "
         "and is written out to be disagreed with."
         % (len(patterns.ALL), len(patterns.measurable()),
            len(patterns.with_threshold())),
         tiers(measured),
         "Each source carries what kind of claim it can support at all, and "
         "whether whoever wrote it had something to win. Sorting by the first "
         "before sorting by quality is the whole trick: a taxonomy cannot be "
         "wrong the way a measurement can be wrong, only useful or not.",
         section("Working hours and rest",
                 "Occupational thresholds and legal entitlements. None of "
                 "them is a clinical finding about an individual.",
                 OCCUPATIONAL),
         section("Sleep", "Experimental work on what restricted sleep does to "
                 "performance.", SLEEP),
         section("Interface and language model",
                 "Taxonomies of design that works against the person using "
                 "it, and the two studies that measure it in conversational "
                 "systems.", DARK_PATTERNS),
         section("The dissent",
                 "Sources that argue against thresholds elsewhere in this "
                 "library. They are here because a standards library "
                 "assembled only from agreement returns whatever it was built "
                 "to return, and the way to tell the difference is whether "
                 "anything in it can lose. They are %.0f%% of the library, "
                 "which is reported and not graded: there is no published "
                 "figure for how much a citation library ought to argue with "
                 "itself." % (100 * dissent_strength()), DISSENT))


FORMATS = ("html", "pdf", "csv", "tsv")

# THEMED says which formats a palette applies to. csv and tsv are the same rows
# whichever theme is named, so --all writes one of each rather than two
# identical files with different names.
THEMED = {"html", "pdf"}


# write_one produces a single artifact and returns the path. It is the only
# place that knows how a format reaches disk, so adding a format is one branch
# here and one entry in FORMATS.
def write_one(fmt, theme, dest, which="standards", measured=None):
    dest.parent.mkdir(parents=True, exist_ok=True)
    if fmt == "html":
        dest.write_text(render(theme, measured))
    elif fmt == "pdf":
        to_pdf(render(theme, measured), dest)
    else:
        with dest.open("w", newline="") as fh:
            tabular(fh, "," if fmt == "csv" else "\t", which)
    return dest


def main():
    ap = argparse.ArgumentParser(
        description="Render lib/exposure as a report.",
        epilog="pdf needs prince on PATH; without it --format pdf fails "
               "rather than writing something else.")
    ap.add_argument("--theme", choices=sorted(THEMES), default="corvid")
    ap.add_argument("--format", default="html",
                    help="comma-separated: %s" % ", ".join(FORMATS))
    ap.add_argument("-o", "--out", help="write here; '-' means stdout")
    ap.add_argument("--measure", action="store_true",
                    help="run the counters over this machine's transcripts "
                         "and put each result beside its pattern")
    ap.add_argument("--since", metavar="YYYY-MM-DD",
                    help="only transcripts from this date on")
    ap.add_argument("--table", choices=("patterns", "standards"),
                    default="standards",
                    help="which table csv/tsv writes to stdout")
    ap.add_argument("--all", metavar="OUTDIR",
                    help="every theme and every requested format into here")
    a = ap.parse_args()

    measured = detect.run(a.since) if a.measure else None

    fmts = [f.strip() for f in a.format.split(",") if f.strip()]
    bad = [f for f in fmts if f not in FORMATS]
    if bad:
        print("unknown format(s): %s; have %s"
              % (", ".join(bad), ", ".join(FORMATS)), file=sys.stderr)
        return 2

    # Check the renderer before writing anything. A run that emits three files
    # and then fails on the fourth leaves a half-made report that looks whole.
    if "pdf" in fmts and not shutil.which("prince"):
        print("pdf needs prince and it is not on PATH. Install it "
              "(brew install --cask prince) or drop pdf from --format. "
              "Nothing was written.", file=sys.stderr)
        return 3

    if a.all:
        d = Path(a.all).expanduser()
        for fmt in fmts:
            if fmt in THEMED:
                for name in sorted(THEMES):
                    print(write_one(fmt, name,
                                    d / ("exposure-%s.%s" % (name, fmt)),
                                    measured=measured))
            else:
                for which in ("patterns", "standards"):
                    print(write_one(fmt, a.theme,
                                    d / ("exposure-%s.%s" % (which, fmt)),
                                    which))
        return 0

    if len(fmts) > 1:
        print("more than one --format needs --all", file=sys.stderr)
        return 2
    fmt = fmts[0]

    if not a.out or a.out == "-":
        if fmt == "pdf":
            print("pdf cannot go to stdout; give -o PATH", file=sys.stderr)
            return 2
        if fmt == "html":
            sys.stdout.write(render(a.theme, measured))
        else:
            tabular(sys.stdout, "," if fmt == "csv" else "\t", a.table)
        return 0

    print(write_one(fmt, a.theme, Path(a.out).expanduser(),
                    measured=measured))
    return 0


if __name__ == "__main__":
    sys.exit(main())
