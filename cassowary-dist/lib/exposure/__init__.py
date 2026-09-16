#!/usr/bin/env python3
# exposure -- the published thresholds this estate's instruments measure
# against, and the machinery for citing them in a report.
#
# why a library. A number like 55 is worthless without the sentence it came
# from. Sitting loose in a script it becomes somebody's opinion within a month,
# gets adjusted to fit, and the instrument stops meaning anything. Everything
# here carries its citation, the claim in the source's own words, what it
# cannot be used to argue, and which way it cuts.
#
# what this is not. Not a diagnostic, not a clinical instrument, and not a
# basis for telling anyone anything about their health. Every threshold in it
# is a population figure or a legal entitlement. The DISSENT list exists to
# keep that visible: it holds sources that argue against thresholds in the rest
# of the file, and the check fails without them.
#
#   python3 -m exposure.standards --check

from . import patterns
from .patterns import Pattern
from .standards import (
    Standard,
    ALL, BY_KEY, OCCUPATIONAL, SLEEP, DARK_PATTERNS, DISSENT,
    get,
    VERIFIED, PARTIAL, RECALLED,
    STATUTORY, DEFINITIONAL, DOCUMENTARY, ASSOCIATIONAL, CAUSAL,
    DISINTERESTED, MANDATED, ADVOCACY, ADVERSARIAL,
    SUPPORTS, QUALIFIES, DISPUTES,
    NEAR, ADJACENT, FAR,
    dissent_strength,
)

__all__ = [
    "Standard", "Pattern", "patterns",
    "ALL", "BY_KEY", "OCCUPATIONAL", "SLEEP", "DARK_PATTERNS",
    "DISSENT", "get", "Citation", "cited", "footnotes", "unsupported",
    "VERIFIED", "PARTIAL", "RECALLED",
    "STATUTORY", "DEFINITIONAL", "DOCUMENTARY", "ASSOCIATIONAL", "CAUSAL",
    "DISINTERESTED", "MANDATED", "ADVOCACY", "ADVERSARIAL",
    "SUPPORTS", "QUALIFIES", "DISPUTES",
    "NEAR", "ADJACENT", "FAR",
    "dissent_strength",
]


class Citation:
    """A measured value, the standard it is read against, and the relation.

    Stance and distance live here rather than on the Standard because they are
    relations, not attributes. The same source supports one claim and disputes
    another, and is near to one question and far from a second. Storing them on
    the Standard was correct only while every entry had exactly one consumer,
    and would have handed the second consumer the first one's reading.
    """

    def __init__(self, standard, value, unit, stance, distance, note=None):
        self.standard, self.value, self.unit = standard, value, unit
        self.stance, self.distance = stance, distance
        self.note = note

    # honest is the question a report should ask before printing the value: a
    # far, disputed or unverified citation is not forbidden, but a page that
    # renders it identically to a near, supported, verified one is lying by
    # typography.
    def honest(self):
        return (self.standard.status == VERIFIED
                and self.stance == SUPPORTS
                and self.distance == NEAR)

    def __repr__(self):
        return f"<Citation {self.standard.key} {self.stance}/{self.distance}>"


# cited pairs a measured value with the standard it is read against. stance and
# distance are required arguments: a caller that has not decided which way its
# source cuts, or how far the source's subject is from its own, has not
# finished thinking about the citation, and a default would let it skip that.
def cited(key, value, stance, distance, unit=None, note=None):
    s = get(key)
    return Citation(s, value, unit or s.unit, stance, distance, note)


# footnotes returns the standards a report actually used, in the order it used
# them, de-duplicated. Building the note list from the citations rather than
# from ALL means a report cannot carry a bibliography of things it did not
# rely on, which is the usual way a citation list stops meaning anything.
def footnotes(citations):
    seen, out = set(), []
    for c in citations:
        s = c.standard if isinstance(c, Citation) else c
        if s.key not in seen:
            seen.add(s.key)
            out.append(s)
    return out


# unsupported returns the keys a report cites that are not VERIFIED, so a
# generator can refuse to publish, or mark them in the output, rather than
# printing a partly-checked figure with the same authority as a checked one.
def unsupported(citations):
    return [s for s in footnotes(citations) if s.status != VERIFIED]
