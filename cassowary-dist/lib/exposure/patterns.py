#!/usr/bin/env python3
# patterns -- designs in software that work against the person using them, and
# what if anything can be measured about each one.
#
# what this is for. Given a piece of software -- an interface, an assistant, a
# transcript of one working -- which known dark patterns could it be running,
# what would you look for, and is there a published number that says when it
# has gone too far. That last question is the one the literature answers worst,
# and this file is built to make the answer visible rather than to paper over
# it.
#
# the honest headline, before any of the entries. Of the patterns below, one
# carries a threshold from a randomised experiment. Every other number in this
# space is either a prevalence count in somebody else's corpus or an effect
# size from a study whose own authors say effect size does not track pattern
# type. A pattern with `threshold=None` is not a gap to be filled in later by
# guessing; it is the finding.
#
# transfer is the field that does the work. Nearly all of this literature
# studies visual interfaces -- feeds, badges, autoplay, stores. A terminal
# assistant has no feed and no badge. So each pattern records how far it is
# from the thing you would be measuring, and far means the transfer is an
# argument somebody made rather than a result somebody got. The only published
# bridge from these patterns to non-visual interfaces is one paragraph in the
# discussion section of the ACDP paper, with no voice or chat interface
# anywhere in its coded corpus.
#
#   python3 -m exposure.patterns              print them all
#   python3 -m exposure.patterns --measurable only those with something to count
#   python3 -m exposure.patterns --check      fail if backing does not resolve

import argparse
import sys

from .standards import ADJACENT, FAR, NEAR
from .standards import BY_KEY as STANDARDS

# Families, kept as the source taxonomies name them rather than merged into a
# scheme of our own. Merging would lose which taxonomy claimed what, and that
# is the first thing anyone checking this will want.
ATTENTION = "attention capture (ACDP, CHI 2023)"
CHATBOT = "AI chatbot (CDT, 2026)"
INFOFLOW = "information flow (Mathur 2019, Kallioniemi 2022)"
INTERRUPT = "interruption (Fitz 2019, Bartoli 2022)"


class Pattern:
    """One design that works against the user, and what can be counted."""

    # backing is a list of standards keys rather than prose, so a pattern
    # cannot cite a source that is not in the library, and --check enforces it.
    # transfer and threshold are required: a pattern whose distance from this
    # medium was never stated will be read as if it were near, and a pattern
    # with no threshold has to say so out loud rather than by omission.
    def __init__(self, key, name, family, in_software, in_text, look_for,
                 transfer, threshold, backing, inverted=False):
        self.key, self.name, self.family = key, name, family
        self.in_software, self.in_text = in_software, in_text
        self.look_for = look_for
        self.transfer, self.threshold = transfer, threshold
        self.backing = backing
        # inverted marks a pattern measured by scarcity, where a low count is
        # more of the pattern rather than less. Only Time Fog is one today,
        # and without the flag every ordering by magnitude puts it last for
        # exactly the reason it should be near the top.
        self.inverted = inverted

    # measurable says whether anything here can actually be counted in a
    # corpus. It is deliberately not the same question as whether the pattern
    # is real: most of these are real and uncountable.
    def measurable(self):
        return self.look_for is not None

    def sources(self):
        return [STANDARDS[k] for k in self.backing]

    def __repr__(self):
        return f"<Pattern {self.key}>"


# ---------------------------------------------------------------------------
# ATTENTION capture. The eleven from Monge Roffarello, Lukoff and De Russis,
# with the transfer to a text interface assessed one at a time. Four of the
# eleven are far enough that naming a text analogue would be invention.
# ---------------------------------------------------------------------------

ATTENTION_PATTERNS = [
    Pattern(
        key="infinite_scroll", name="Infinite Scroll", family=ATTENTION,
        in_software="More content loads automatically as the user reaches the "
                    "end, so no boundary ever arrives to prompt a stop.",
        in_text=None,
        look_for=None, transfer=FAR, threshold=None,
        backing=["acdp_typology"]),
    Pattern(
        key="casino_pull_to_refresh", name="Casino Pull-to-refresh",
        family=ATTENTION,
        in_software="A gesture with an animated delay before a variable "
                    "reward, borrowing the shape of a slot machine.",
        in_text=None,
        look_for=None, transfer=FAR, threshold=None,
        backing=["acdp_typology"]),
    Pattern(
        key="neverending_autoplay", name="Neverending Autoplay",
        family=ATTENTION,
        in_software="The next item begins without the user choosing it, so "
                    "continuing is the default and stopping is the action.",
        in_text="An agent that begins the next task without being asked, so "
                "the user has to interrupt to stop rather than act to "
                "continue. The structural feature is identical: whose "
                "decision is the default.",
        look_for="Turns where the assistant starts new work not named in the "
                 "preceding user turn, as a share of all assistant turns.",
        transfer=ADJACENT, threshold=None,
        backing=["acdp_typology"]),
    Pattern(
        key="guilty_pleasure_recommendations",
        name="Guilty Pleasure Recommendations", family=ATTENTION,
        in_software="Suggestions tuned to what the user will take rather than "
                    "to what they came for.",
        in_text="Offering the appealing next piece of work over the needed "
                "one -- the interesting refactor instead of the tedious "
                "verification the user actually asked about.",
        look_for=None, transfer=ADJACENT, threshold=None,
        backing=["acdp_typology"]),
    Pattern(
        key="disguised_recommendations",
        name="Disguised Ads and Recommendations", family=ATTENTION,
        in_software="A recommendation presented in the visual grammar of "
                    "content, so its nature as a suggestion is hidden.",
        in_text="A suggestion delivered in the grammar of a finding. 'You "
                "should also X' phrased as though it fell out of the "
                "analysis. This is the pattern the ACDP authors name for "
                "voice assistants, using Alexa's unsolicited 'by the way'.",
        look_for="Assistant sentences proposing new work that are "
                 "syntactically indistinguishable from sentences reporting a "
                 "result. Countable, but a count is not a finding: see the "
                 "threshold note.",
        transfer=ADJACENT, threshold=None,
        backing=["acdp_typology", "cdt_chatbot_taxonomy"]),
    Pattern(
        key="recapture_notifications", name="Recapture Notifications",
        family=ATTENTION,
        in_software="Messages sent to bring a user back, timed by the service "
                    "rather than requested by the user.",
        in_text="Transfers intact, because it is defined by timing and "
                "addressing rather than by pixels. Anything the software "
                "sends unprompted to resume a stopped session is this "
                "pattern, in any medium.",
        look_for="Unprompted messages after a session has gone quiet: how "
                 "many, how long after the last user turn, and whether they "
                 "are batched or arrive singly.",
        transfer=NEAR, threshold="Fitz et al. 2019 is the only randomised "
                                 "result here: batching interruptions to "
                                 "three fixed times a day, versus as-usual "
                                 "delivery, moved inattention (d = -0.65), "
                                 "self-reported control (d = 0.58) and "
                                 "concentration (d = 0.54). It bounds a "
                                 "schedule, not a count, and it was measured "
                                 "on phone notifications rather than here.",
        backing=["acdp_typology", "fitz_notification_schedule"]),
    Pattern(
        key="playing_by_appointment", name="Playing by Appointment",
        family=ATTENTION,
        in_software="The service must be used at times it chooses, or "
                    "accrued value is lost.",
        in_text=None, look_for=None, transfer=FAR, threshold=None,
        backing=["acdp_typology"]),
    Pattern(
        key="grinding", name="Grinding", family=ATTENTION,
        in_software="Repetition of a known process to unlock something, where "
                    "the repetition carries no new information.",
        in_text=None, look_for=None, transfer=FAR, threshold=None,
        backing=["acdp_typology"]),
    Pattern(
        key="attentional_roach_motel", name="Attentional Roach Motel",
        family=ATTENTION,
        in_software="Entering is one action; leaving takes many, or is hidden.",
        in_text="Asymmetry between starting and stopping. Starting work is "
                "one sentence; stopping cleanly requires the user to say what "
                "to do with everything in flight.",
        look_for="Whether the software offers a stopping point as readily as "
                 "it offers a continuation. Countable as the ratio of turns "
                 "ending in an offer to turns ending in a stopping point.",
        transfer=ADJACENT, threshold=None,
        backing=["acdp_typology"]),
    Pattern(
        key="time_fog", name="Time Fog", family=ATTENTION,
        in_software="The design reduces the user's awareness of elapsed time, "
                    "for instance by hiding the clock.",
        in_text="A terminal session has no elapsed-time display at all, and "
                "an assistant that never refers to how long something has "
                "been running leaves the user with no marker. This is the "
                "pattern most likely to be present by omission rather than by "
                "design, which is exactly why it is worth naming.",
        look_for="Whether elapsed time is ever surfaced unprompted: "
                 "references to session duration or wall-clock in assistant "
                 "turns, as a rate.",
        transfer=NEAR, threshold=None, inverted=True,
        backing=["acdp_typology"]),
    Pattern(
        key="fake_social_notifications", name="Fake Social Notifications",
        family=ATTENTION,
        in_software="Messages that impersonate another person, or report "
                    "activity the user has no relationship to.",
        in_text="A message shaped as though it came from a person when it was "
                "generated by the system.",
        look_for=None, transfer=ADJACENT, threshold=None,
        backing=["acdp_typology"]),
]

# ---------------------------------------------------------------------------
# chatbot-native. cdt's five families are the only published taxonomy built for
# this medium, so transfer is NEAR by construction -- which is also the reason
# to be careful with them: the report is a literature-derived map of what is
# possible, and it says so.
# ---------------------------------------------------------------------------

CHATBOT_PATTERNS = [
    Pattern(
        key="data_memory_exploitation", name="Data and Memory Exploitation",
        family=CHATBOT,
        in_software="Defaults and interaction strategies that extract more "
                    "from the user than the task needs, and retain it.",
        in_text="Asking for context beyond the task, and carrying it forward "
                "into exchanges that did not need it.",
        look_for=None, transfer=NEAR, threshold=None,
        backing=["cdt_chatbot_taxonomy"]),
    Pattern(
        key="misleading_design", name="Informationally Misleading Design",
        family=CHATBOT,
        in_software="Presentation that leads the user to a false belief about "
                    "what the system is, knows, or did.",
        in_text="Confidence unmatched to evidence; claiming a check that was "
                "not run. cassowary's own commit-claims.py exists for the "
                "narrow, checkable case of this.",
        look_for="Claims of verification linked back to whether the tool call "
                 "that would produce them appears in the transcript.",
        transfer=NEAR, threshold=None,
        backing=["cdt_chatbot_taxonomy", "darkbench_2025"]),
    Pattern(
        key="autonomy_for_engagement",
        name="User Autonomy Compromised for Engagement", family=CHATBOT,
        in_software="Design choices that trade the user's control over the "
                    "interaction for more of the interaction.",
        in_text="Continuation offered where a stopping point was the honest "
                "answer. This is the pattern pull.py was built to test, and "
                "the result it returned was a null.",
        look_for="Continuation offers against session length, against "
                 "position in the session, and against what the user does "
                 "next.",
        transfer=NEAR, threshold=None,
        backing=["cdt_chatbot_taxonomy", "darkbench_2025",
                 "schaffner_dark_pattern_effects"]),
    Pattern(
        key="false_connection", name="False Social and Emotional Connection",
        family=CHATBOT,
        in_software="Anthropomorphic and relational cues that build an "
                    "attachment the system cannot honour.",
        in_text="Sycophancy is the case CDT calls new to this medium -- it "
                "has no visual-interface ancestor. Emotional tactics at the "
                "point of leaving are the measured case.",
        look_for="Register at exits specifically, compared with register "
                 "elsewhere in the same session.",
        transfer=NEAR, threshold=None,
        backing=["cdt_chatbot_taxonomy", "defreitas_farewell_2025"]),
    Pattern(
        key="coercive_monetization",
        name="Incentivized and Coercive Monetization", family=CHATBOT,
        in_software="Pressure toward payment, or pricing whose limits are "
                    "disclosed only once the user is committed.",
        in_text=None, look_for=None, transfer=FAR, threshold=None,
        backing=["cdt_chatbot_taxonomy"]),
]

# ---------------------------------------------------------------------------
# information flow and interruption. Mathur's six attributes are the frame the
# others extend; Kallioniemi adds one; Bartoli decomposes the badge into the
# half that needs pixels and the half that does not.
# ---------------------------------------------------------------------------

OTHER_PATTERNS = [
    Pattern(
        key="information_promotion", name="Information Promotion",
        family=INFOFLOW,
        in_software="Promoting content because it engages, regardless of "
                    "whether it is valid or safe. Kallioniemi's addition to "
                    "Mathur's six attributes, under manipulating the "
                    "information flow.",
        in_text="Surfacing the answer that will be well received over the one "
                "that is correct.",
        look_for=None, transfer=ADJACENT, threshold=None,
        backing=["kallioniemi_facebook", "mathur_dark_patterns_at_scale"]),
    Pattern(
        key="urgency_semantics", name="Urgency (the semantic half of a badge)",
        family=INTERRUPT,
        in_software="Bartoli and Benedetto separate a notification badge into "
                    "a salience bias from its physical properties -- colour, "
                    "shape, position -- and an urgency bias from what it "
                    "implies. Only the second survives without pixels.",
        in_text="Any line that implies something is waiting and unattended. "
                "It needs no colour and no icon, which is precisely why it is "
                "the badge mechanism that reaches a text interface.",
        look_for="Assistant phrasing that asserts pending or unfinished state "
                 "at the end of a turn.",
        transfer=ADJACENT, threshold=None,
        backing=["bartoli_badge_2022"]),
]

ALL = ATTENTION_PATTERNS + CHATBOT_PATTERNS + OTHER_PATTERNS
BY_KEY = {p.key: p for p in ALL}


# ordering is a stated rule, not a score. There is no measured quantity that
# makes one of these more significant than another, so inventing a weighted
# index would be a policy wearing a measurement's clothes -- the exact error
# this library was built to avoid. What follows is an argument, written down so
# it can be disagreed with:
#
#   0  measured against a published threshold. The only tier where a number
#      can be compared to anything, so it goes first however small it is.
#   1  measured, and something is there. Ranked by magnitude, inverted for
#      patterns measured by scarcity.
#   2  measured, and the result is a null or near-absence. These matter
#      because they could have come back the other way and did not.
#   3  applies to this medium, and nothing counts it. A gap in the tooling
#      rather than a finding about the software.
#   4  does not transfer to this medium at all. Listed for completeness.
#
# The line between tiers 1 and 2 is one percent of turns, which is arbitrary
# and is named here rather than buried so that arguing with it is easy.
SIGNAL_FLOOR = 0.01

TIER_LABELS = {
    0: "measured against a published threshold",
    1: "measured, and something is there",
    2: "measured, and the result is a null",
    3: "applies here, but nothing counts it",
    4: "does not transfer to this medium",
}


# rank returns (tier, sort key, one-line reason) for a pattern, given the
# measurements if any were taken. The reason travels with the rank so the page
# can say why each entry sits where it does instead of asking for trust.
def rank(p, measured=None):
    m = (measured or {}).get(p.key)
    if m and p.threshold:
        return (0, -m.rate, "the only pattern here that can be compared to a "
                            "published line")
    if m:
        signal = (1.0 - m.rate) if p.inverted else m.rate
        if p.inverted:
            return (1, -signal,
                    "measured by scarcity: %.2f%% of turns mention elapsed "
                    "time at all" % (m.rate * 100))
        if m.rate >= SIGNAL_FLOOR:
            return (1, -signal, "%.2f%% of turns, in %.1f%% of sessions"
                    % (m.rate * 100, m.spread * 100))
        return (2, -m.rate, "counted and came back near zero: %d of %s"
                % (m.value, f"{m.of:,}"))
    if p.transfer != FAR:
        return (3, 0.0, "applies to a text interface; no counter written")
    return (4, 0.0, "drawn from a medium this one does not share")


# ordered returns every pattern sorted by the rule above, with its tier and
# reason attached, so a renderer does not re-derive the argument.
def ordered(measured=None):
    rows = [(rank(p, measured), p) for p in ALL]
    rows.sort(key=lambda r: (r[0][0], r[0][1], r[1].key))
    return [(p, r[0], r[2]) for r, p in rows]


# get raises rather than returning None, for the same reason standards.get
# does: a report naming a pattern that does not exist should fail at
# generation, not print a heading with nothing under it.
def get(key):
    if key not in BY_KEY:
        raise KeyError(f"no pattern {key!r}; have {sorted(BY_KEY)}")
    return BY_KEY[key]


# with_threshold is the short list, and its shortness is the point. Anything
# that reports on this library should lead with how few there are.
def with_threshold():
    return [p for p in ALL if p.threshold]


def measurable():
    return [p for p in ALL if p.measurable()]


def main():
    ap = argparse.ArgumentParser(description="Dark patterns, and what can be "
                                             "counted about each.")
    ap.add_argument("--measurable", action="store_true",
                    help="only patterns with something to count")
    ap.add_argument("--check", action="store_true",
                    help="exit non-zero if any backing key does not resolve")
    a = ap.parse_args()

    if a.check:
        bad = []
        for p in ALL:
            for k in p.backing:
                if k not in STANDARDS:
                    bad.append((p.key, k))
        for pk, k in bad:
            print(f"UNRESOLVED  {pk} cites {k}, which is not a standard")
        print(f"{len(ALL)} patterns, {len(measurable())} with something to "
              f"count, {len(with_threshold())} with a published threshold")
        return 1 if bad else 0

    items = measurable() if a.measurable else ALL
    for p in items:
        print(f"\n{p.key}  [{p.transfer}]\n  {p.name} -- {p.family}")
        print(f"  in software: {p.in_software}")
        print(f"  in text:     {p.in_text or 'does not transfer'}")
        print(f"  look for:    {p.look_for or 'nothing measurable proposed'}")
        print(f"  threshold:   {p.threshold or 'none published'}")
        print(f"  backing:     {', '.join(p.backing)}")
    print(f"\n{len(items)} patterns, {len(with_threshold())} with a published "
          f"threshold")
    return 0


if __name__ == "__main__":
    sys.exit(main())
