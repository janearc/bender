#!/usr/bin/env python3
# standards -- the published thresholds these tools measure against, with the
# source for each one and an honest note on what it does not support.
#
# why this is a library and not a constant in a script. A number like 55 is
# worthless without the sentence it came from. Sitting loose in a tool it
# becomes somebody's opinion within a month, gets adjusted to fit, and the
# instrument stops meaning anything. Here every figure carries its citation, the
# claim in the source's own words, and a limits field saying what it cannot be
# used to argue.
#
# VERIFIED means the primary source was fetched and read while writing this, and
# the date it was checked. RECALLED means it was not, and it should be read
# before being relied on. Nothing here is asserted at a confidence it has not
# earned.
#
# Every entry also declares a stance: whether it supports the way a tool here
# uses it, qualifies it, or disputes it. The DISSENT list holds sources that
# argue against thresholds elsewhere in this file, and --check reports what
# share of the library they are, so that a set of citations chosen to agree
# with the answer shows up as a number rather than passing quietly.
#
#   python3 -m exposure.standards            print them all
#   python3 -m exposure.standards --check    exit non-zero if anything is
#                                            RECALLED or the dissent is thin

import argparse
import sys

VERIFIED = "verified"
PARTIAL = "partial"        # some fields confirmed at source, some not
RECALLED = "recalled"

# three axes, and they are not the same shape.
#
# status and evidence and interest are attributes of the source: they are the
# same no matter who cites it, so they live on the Standard.
#
# stance and distance are relations between a source and a use. The same paper
# is close to one question and far from another, and supports one claim while
# disputing a second. They live on a Citation, not here. An earlier version of
# this file put stance on the Standard; that was only correct while each entry
# had exactly one consumer, and it would have handed the second consumer the
# first one's stance without saying so.
#
# evidence -- what kind of claim the source can support at all. Sorting by this
# before sorting by quality is the whole trick: a taxonomy cannot be wrong the
# way a measurement can be wrong, only useful or not, and quoting one as
# evidence that something happens is the most common error available here.
STATUTORY = "statutory"          # a legal instrument; binding, not evidence
DEFINITIONAL = "definitional"    # proposes a category; establishes nothing
DOCUMENTARY = "documentary"      # this document exists and says this
ASSOCIATIONAL = "associational"  # observed together; design cannot say why
CAUSAL = "causal"                # randomised or otherwise identified

# interest -- whether the source was written by someone with something to win.
# Descriptive, not pejorative: a mandated body doing careful science is still
# a mandated body, and saying so is what lets a reader weigh it themselves.
DISINTERESTED = "disinterested"  # no stake beyond getting it right
MANDATED = "mandated"            # a body with authority or remit on the subject
ADVOCACY = "advocacy"            # exists to advance a position
ADVERSARIAL = "adversarial"      # written to win a specific dispute

# the asymmetry is a policy, not a measurement, and is written here rather than
# encoded as a number because it is arguable and nothing settles it. For an
# instrument whose output might tell a person something uncomfortable about
# themselves, require more of a source before it may carry the claim than
# before it may carry the caveat. Deliberately unbalanced. Somebody may
# reasonably think the imbalance should be smaller; there is no fact that
# decides it, so it is stated instead of buried in a threshold.
#
# and this library is not a case against anybody. A source is included because
# a tool here reads a number against it. Material that indicts a company or a
# person without supporting a threshold is left out however well documented it
# is -- the Meta internal-documents strand is the worked example, and it is
# absent on those grounds and not because it is weak.

# Stance and distance, declared per citation. Kept in this module because they
# name the same vocabulary the standards are described in.
SUPPORTS = "supports"      # backs the way this caller uses it
QUALIFIES = "qualifies"    # accepted, but narrows what the number can mean
DISPUTES = "disputes"      # argues against the threshold or against the method

NEAR = "near"              # studied the same kind of thing this tool measures
ADJACENT = "adjacent"      # same mechanism, different medium or population
FAR = "far"                # transfer is an argument, not a finding

# dissent_strength replaced a floor. MIN_DISSENT was a minimum count nobody set
# on evidence, and it could not fail for the reason it named: two token
# dissenters satisfy a floor of two beside any quantity of agreement, so the
# check could only catch somebody deleting entries. What can be said honestly is
# how much of the library argues against the rest of it. That is a number, it
# moves when a source is added either way, and it is reported rather than graded
# -- the same rule the rest of this kit follows.
def dissent_strength():
    """Share of the library that disputes another entry in it."""
    return len(DISSENT) / len(ALL) if ALL else 0.0


class Standard:
    """One published threshold and everything needed to cite it honestly."""

    # Limits, evidence and interest are required rather than optional. A
    # threshold without its limits is the failure this library exists to
    # prevent; a threshold whose evidence class nobody wrote down gets quoted
    # as a measurement the first time somebody is in a hurry.
    def __init__(self, key, domain, claim, source, url, status, checked,
                 limits, evidence, interest, threshold=None, unit=None):
        self.key, self.domain, self.claim = key, domain, claim
        self.source, self.url = source, url
        self.status, self.checked = status, checked
        self.limits = limits
        self.evidence, self.interest = evidence, interest
        self.threshold, self.unit = threshold, unit

    def cite(self):
        return f"{self.source}"

    def __repr__(self):
        return f"<Standard {self.key}>"


OCCUPATIONAL = [
    Standard(
        key="who_ilo_55h",
        evidence=ASSOCIATIONAL, interest=MANDATED,
        domain="working hours",
        claim="Working 55 or more hours a week is associated with a 35% higher "
              "risk of stroke and a 17% higher risk of dying from ischaemic "
              "heart disease, compared with working 35 to 40 hours a week.",
        source="Pega et al., WHO/ILO joint estimates, Environment International, "
               "17 May 2021. Synthesis of 37 studies on heart disease "
               "(768,000+ participants) and 22 on stroke (839,000+), covering "
               "154 countries, 1970-2018.",
        url="https://www.who.int/news/item/17-05-2021-long-working-hours-"
            "increasing-deaths-from-heart-disease-and-stroke-who-ilo",
        status=VERIFIED, checked="2026-09-04",
        threshold=55, unit="hours per week",
        limits="A population association, not a prediction about any one "
               "person. It says nothing about which seven days: the comparison "
               "is a weekly average, so a calendar week and a rolling window "
               "are both defensible readings and a rolling one is harder to "
               "hide a heavy stretch inside."),
    Standard(
        key="eu_wtd_daily_rest",
        evidence=STATUTORY, interest=MANDATED,
        domain="rest",
        claim="Every worker is entitled to a minimum daily rest period of 11 "
              "consecutive hours per 24-hour period.",
        source="Directive 2003/88/EC of the European Parliament and of the "
               "Council, Article 3.",
        url="https://eur-lex.europa.eu/legal-content/EN/TXT/HTML/?uri=CELEX:32003L0088",
        status=VERIFIED, checked="2026-09-04",
        threshold=11, unit="consecutive hours in 24",
        limits="A legal entitlement in an employment relationship, not a "
               "clinical threshold. It does not apply to a self-employed "
               "person and cannot be cited as a health finding. Its value here "
               "is that it is a number somebody else set."),
    Standard(
        key="eu_wtd_weekly_rest",
        evidence=STATUTORY, interest=MANDATED,
        domain="rest",
        claim="Per each seven-day period, every worker is entitled to a minimum "
              "uninterrupted rest period of 24 hours plus the 11 hours' daily "
              "rest -- 35 hours in total.",
        source="Directive 2003/88/EC, Article 5.",
        url="https://eur-lex.europa.eu/legal-content/EN/TXT/HTML/?uri=CELEX:32003L0088",
        status=VERIFIED, checked="2026-09-04",
        threshold=35, unit="consecutive hours per 7 days",
        limits="Commonly quoted as 24 hours, which is wrong: the Article adds "
               "the daily 11 on top. Same employment-law caveat as Article 3."),
    Standard(
        key="eu_wtd_weekly_max",
        evidence=STATUTORY, interest=MANDATED,
        domain="working hours",
        claim="Average working time for each seven-day period, including "
              "overtime, must not exceed 48 hours.",
        source="Directive 2003/88/EC, Article 6.",
        url="https://eur-lex.europa.eu/legal-content/EN/TXT/HTML/?uri=CELEX:32003L0088",
        status=VERIFIED, checked="2026-09-04",
        threshold=48, unit="hours per week averaged",
        limits="An average over a reference period, not a hard weekly cap, and "
               "individually waivable in some member states."),
    Standard(
        key="icd11_burnout",
        evidence=DEFINITIONAL, interest=MANDATED,
        domain="outcome",
        claim="Burn-out is an occupational phenomenon and is not classified as "
              "a medical condition. Three dimensions: feelings of energy "
              "depletion or exhaustion; increased mental distance from one's "
              "job, or feelings of negativism or cynicism related to one's job; "
              "and reduced professional efficacy. It refers specifically to "
              "phenomena in the occupational context and should not be applied "
              "to describe experiences in other areas of life.",
        source="World Health Organization, ICD-11, statement of 28 May 2019.",
        url="https://www.who.int/news/item/28-05-2019-burn-out-an-occupational-"
            "phenomenon-international-classification-of-diseases",
        status=VERIFIED, checked="2026-09-04",
        limits="Not diagnosable from behavioural data. All three dimensions are "
               "subjective states; none can be read off timestamps, commit "
               "counts or anything else an instrument can see. A tool may "
               "measure exposure. It may not infer this."),
    Standard(
        key="iarc_night_work",
        evidence=ASSOCIATIONAL, interest=MANDATED,
        domain="circadian",
        claim="Night shift work involving circadian disruption is classified as "
              "probably carcinogenic to humans, Group 2A.",
        source="IARC Monographs Volume 124, Night Shift Work (2020). The "
               "volume number, year, and subject -- night shift work, defined "
               "as work occurring during the regular sleeping hours of the "
               "general population -- were confirmed at the publisher. The "
               "Group 2A assignment was not: it sits in a PDF of Section 6 and "
               "on a classification page that loads its table dynamically, and "
               "neither could be read.",
        url="https://publications.iarc.who.int/Book-And-Report-Series/"
            "Iarc-Monographs-On-The-Identification-Of-Carcinogenic-Hazards-To-"
            "Humans/Night-Shift-Work-2020",
        status=PARTIAL, checked="2026-09-04",
        limits="Read Section 6 before quoting the group. Beyond that: built around shift systems that displace a fixed daytime "
               "schedule. Its application to a person with no fixed schedule is "
               "an assumption, not a finding, and the clock hour alone is a "
               "poor proxy. Read the monograph before relying on this."),
]

SLEEP = [
    Standard(
        key="van_dongen_2003",
        evidence=CAUSAL, interest=DISINTERESTED,
        domain="rest",
        claim="Restricting sleep to 4 or 6 hours a night for 14 consecutive "
              "nights produced cumulative, dose-dependent deficits on every "
              "cognitive task measured. Subjective sleepiness rose acutely and "
              "then only slightly, and did not distinguish the 6-hour from the "
              "4-hour condition -- self-report stopped tracking the deficit "
              "while the deficit went on accumulating.",
        source="Van Dongen, Maislin, Mullington and Dinges, The Cumulative Cost "
               "of Additional Wakefulness: Dose-Response Effects on "
               "Neurobehavioral Functions and Sleep Physiology from Chronic "
               "Sleep Restriction and Total Sleep Deprivation. Sleep 26(2), "
               "March 2003, 117-126.",
        url="https://academic.oup.com/sleep/article-abstract/26/2/117/2709164",
        status=VERIFIED, checked="2026-09-04",
        limits="A laboratory study at fixed sleep doses, not a field study of "
               "people choosing their own hours. The careful statement is that "
               "self-report plateaus while performance keeps falling -- not "
               "that people become unable to assess themselves at all. It is "
               "the one result here that licenses an instrument to disagree "
               "with a person about their own state, and it licenses it only "
               "narrowly."),
]

DARK_PATTERNS = [
    Standard(
        key="brignull_dark_patterns",
        evidence=DEFINITIONAL, interest=ADVOCACY,
        domain="interface",
        claim="Deceptive patterns are features that trick people into doing "
              "things they did not mean to: stopping them doing what they want, "
              "or steering them toward harmful decisions.",
        source="deceptive.design, formerly darkpatterns.org, run by Testimonium "
               "Ltd (England and Wales, no. 12206608); the project dates itself "
               "to 2010. Associated with Harry Brignull; the site itself does "
               "not state who coined the term, so that attribution is not made "
               "here.",
        url="https://www.deceptive.design/",
        status=VERIFIED, checked="2026-09-04",
        limits="A taxonomy of interface mechanics, developed for commercial "
               "interfaces with a conversion goal. A conversational system has "
               "no click targets and the taxonomy does not transfer cleanly."),
    Standard(
        key="mathur_dark_patterns_at_scale",
        evidence=ASSOCIATIONAL, interest=DISINTERESTED,
        domain="interface",
        claim="An automated crawl of ~11,000 shopping sites and ~53,000 "
              "product pages found 1,818 instances of deceptive design across "
              "183 sites, classified into 15 types and 7 broader categories, "
              "and identified 22 third-party providers offering them as a "
              "turnkey service.",
        source="Mathur, Acar, Friedman, Lucherini, Mayer, Chetty and "
               "Narayanan, Dark Patterns at Scale: Findings from a Crawl of 11K "
               "Shopping Websites. Proceedings of the ACM on Human-Computer "
               "Interaction, Vol. 3, CSCW, November 2019.",
        url="https://arxiv.org/abs/1907.07032",
        status=VERIFIED, checked="2026-09-04",
        limits="Establishes that automated detection is possible for interface "
               "elements with known surface forms. It does not establish intent "
               "in any single case, and its methods do not transfer to prose."),
    Standard(
        key="kallioniemi_facebook",
        evidence=DEFINITIONAL, interest=DISINTERESTED,
        domain="platform",
        claim="Proposes one addition to Mathur et al.'s six higher-level "
              "dark pattern attributes: Information Promotion, under "
              "manipulating the information flow -- promoting engaging "
              "content regardless of the validity or safety of the "
              "information it contains. The mechanism is located in "
              "information flow, plus appeals to basic emotions (happiness, "
              "disgust, anger), plus behaviour that feels organic and freely "
              "chosen.",
        source="Pekka Kallioniemi, Facebook's Dark Pattern Design, Public "
               "Relations and Internal Work Culture. Journal of Digital Media "
               "& Interaction 5(12):38-54, 2022.",
        url="https://doi.org/10.34624/jdmi.v5i12.28378",
        status=VERIFIED, checked="2026-09-04",
        limits="The one row in Table 2 is the contribution. This is a "
               "taxonomy paper and a document analysis; it is not a "
               "measurement study, and it establishes neither causation nor "
               "association. It contains no coding scheme, no counts and no "
               "reliability statistic. Verified by exhaustive term count on "
               "the full text: chatbot, conversational, text-only, voice, "
               "badge, red dot, pull-to-refresh, streak, autoplay and "
               "infinite scroll all appear zero times; notification appears "
               "once, in a passing citation. It says nothing whatever about "
               "text or conversational interfaces. An earlier version of this "
               "entry summarised the corporate-conduct half of the paper as "
               "though that were its finding, which both overstated the "
               "source and turned this library into an argument about a "
               "company rather than about design. Two further cautions: the "
               "paper never states the gap in negative form, so 'no attribute "
               "existed for this' is inferred from the act of proposing one; "
               "and the novelty claim is contestable, since Monge Roffarello "
               "and De Russis had defined attention-capture dark patterns at "
               "CHI 2022 EA (10.1145/3491101.3519829)."),
    Standard(
        key="darkbench_2025",
        evidence=ASSOCIATIONAL, interest=DISINTERESTED,
        domain="language model",
        claim="Six categories of manipulative behaviour in large language "
              "models: brand bias, user retention (techniques that keep users "
              "engaged), sycophancy, anthropomorphism, harmful generation, and "
              "sneaking. Benchmarked with 660 prompts against models from five "
              "companies.",
        source="Kran, Nguyen, Kundu, Jawhar, Park and Jurewicz, DarkBench: "
               "Benchmarking Dark Patterns in Large Language Models. ICLR 2025, "
               "oral. Submitted 13 March 2025.",
        url="https://arxiv.org/abs/2503.10728",
        status=VERIFIED, checked="2026-09-04",
        limits="A prompt-response benchmark: it probes a model with crafted "
               "inputs and scores the reply. It measures what a model will do "
               "when prompted, not what one did do in a real relationship over "
               "months, which is the opposite direction from a corpus audit. "
               "Its user-retention category is the closest published name for "
               "what a corpus audit of continuation offers is looking at."),
    Standard(
        key="defreitas_farewell_2025",
        evidence=ASSOCIATIONAL, interest=DISINTERESTED,
        domain="language model",
        claim="Across 1,200 real farewell interactions with companion apps, six "
              "manipulative tactics appeared in 37% of goodbyes -- guilt "
              "appeals, fear-of-missing-out hooks, metaphorical restraint and "
              "others -- timed to the moment a user signals they are leaving. "
              "In four preregistered experiments with 3,300 US adults these "
              "raised post-goodbye engagement by up to 14x. The engines were "
              "reactance-based anger and curiosity rather than enjoyment, and "
              "the same tactics raised perceived manipulation, churn intent and "
              "negative word of mouth.",
        source="De Freitas, Oguz-Uguralp and Kaan-Uguralp, Emotional "
               "Manipulation by AI Companions. Submitted 15 August 2025, "
               "revised 7 October 2025.",
        url="https://arxiv.org/abs/2508.19258",
        status=VERIFIED, checked="2026-09-04",
        limits="Companion apps, whose product goal is engagement, not "
               "assistants doing work. The tactics studied are affect-laden -- "
               "guilt, FOMO, restraint -- and an offer of further work is none "
               "of those. Its value is as a positive control: it establishes "
               "that the farewell moment is a real seam where manipulation "
               "happens and has a measurable effect, which makes a null result "
               "at that seam in another corpus worth reporting rather than "
               "assuming."),
    Standard(
        key="cdt_chatbot_taxonomy",
        evidence=DEFINITIONAL, interest=ADVOCACY,
        domain="language model",
        claim="Thirty-seven dark patterns applicable to AI chatbots, grouped "
              "into five areas: data and memory exploitation, informationally "
              "misleading design, user autonomy compromised for engagement, "
              "false social and emotional connection, and incentivized and "
              "coercive monetization.",
        source="Joshi, R., Adjagbodjou, A., Luria, M. (2026). Dark Patterns in "
               "AI Chatbots: A Taxonomy to Inform Better Design. Center for "
               "Democracy and Technology, May 2026. Derived by deductive "
               "literature review in five stages, covering general-purpose "
               "assistants and companion platforms.",
        url="https://cdt.org/insights/dark-patterns-in-ai-chatbots-a-taxonomy-"
            "to-inform-better-design/",
        status=VERIFIED, checked="2026-09-04",
        limits="The report is a literature-derived map of what is possible, "
               "not a measurement of what any deployed system does. Its own "
               "examples are labelled indicative demonstrations rather than "
               "reproducible output, because the systems are probabilistic. "
               "It states plainly that a pattern may arise from system "
               "behaviour rather than a designer's intent to deceive, which "
               "means finding a pattern's surface form in a transcript is not "
               "evidence that anyone designed it. Contents and executive "
               "summary say five categories; the introduction says four. Five "
               "is used here because it is what the document is organised by."),
    Standard(
        key="acdp_typology",
        evidence=DEFINITIONAL, interest=DISINTERESTED,
        domain="interface",
        claim="Eleven attention capture damaging patterns, defined as a "
              "recurring pattern in a digital interface that a designer uses "
              "to exploit psychological vulnerabilities and capture attention, "
              "often leading the user to lose track of their goals, lose their "
              "sense of time and control, and later feel regret. The eleven: "
              "Infinite Scroll, Casino Pull-to-refresh, Neverending Autoplay, "
              "Guilty Pleasure Recommendations, Disguised Ads and "
              "Recommendations, Recapture Notifications, Playing by "
              "Appointment, Grinding, Attentional Roach Motel, Time Fog, and "
              "Fake Social Notifications.",
        source="Monge Roffarello, A., Lukoff, K., De Russis, L. (2023). "
               "Defining and Identifying Attention Capture Deceptive Designs "
               "in Digital Interfaces. CHI '23, Hamburg. PRISMA review "
               "screening 1,334 records down to 43 analysed papers, January "
               "2000 to 13 June 2022, ACM Guide to the Computing Literature "
               "only. The eleven split 4 deceptive to 7 seductive.",
        url="https://doi.org/10.1145/3544548.3580729",
        status=VERIFIED, checked="2026-09-04",
        limits="A synthesis of literature, not an audit of interfaces. The "
               "patterns come off a coding sheet over 43 papers; there is no "
               "interface sampling and no prevalence measurement, so it "
               "establishes association at most. The definition also carries "
               "designer intent inside it -- a pattern the designer uses to "
               "exploit -- and a count of surface forms cannot establish "
               "that, so the typology names shapes worth looking for and "
               "cannot by itself convict one. The corpus closes in June 2022, "
               "so it contains nothing on chatbots or language models. Its "
               "transfer to non-visual interfaces is a single paragraph of "
               "discussion (section 4.3) naming Alexa's unsolicited 'by the "
               "way', with no citation, no data and no reference attached, "
               "followed by the authors calling the typology a starting point "
               "rather than a fixed set; voice, conversational, chatbot and "
               "assistant appear nowhere else in the paper. On measurement "
               "the authors concede an increase in time spent does not "
               "necessarily imply harm and propose triangulating it against "
               "the three impacts in their own definition -- note they do not "
               "dismiss A/B testing, they say the harm metric is the unsolved "
               "part. Citation hazard: the published title says Deceptive "
               "Designs while the body coins damaging patterns."),
    Standard(
        key="fitz_notification_schedule",
        evidence=CAUSAL, interest=DISINTERESTED,
        domain="interruption",
        claim="Batching interruptions to three fixed times a day, rather than "
              "delivering them as they arrive, improved objectively logged "
              "phone unlocks (d = -0.60), self-reported control over the "
              "device (d = 0.58), inattention (d = -0.65), concentration "
              "(d = 0.54) and end-of-day perceived productivity (d = 0.57). "
              "Only delivery to the lock screen was manipulated -- messages "
              "stayed reachable in the app -- so the causal agent is "
              "interruption timing rather than information withheld.",
        source="Fitz, N., Kushlev, K., Jagannathan, R., Lewis, T., Paliwal, "
               "D., Ariely, D. (2019). Computers in Human Behavior "
               "101:84-94. Randomised field experiment, n = 237, four arms, "
               "one baseline week and two experimental weeks.",
        url="https://doi.org/10.1016/j.chb.2019.07.016",
        status=PARTIAL, checked="2026-09-04",
        threshold=3, unit="delivery batches per day",
        limits="The only randomised threshold in the dark pattern half of "
               "this library, and it bounds a schedule rather than a count. "
               "Measured on phone notifications with a single-country MTurk "
               "convenience sample over two weeks, not preregistered. 108 "
               "participants (31 percent) were excluded for failing a "
               "completion rule, though attrition did not differ by arm. "
               "Significance tests are unadjusted across roughly 28 outcomes, "
               "and the unlocks omnibus was only marginal (F(3,233) = 2.52, "
               "p = .059), so d = -0.60 is an unadjusted planned contrast "
               "under a non-significant omnibus while the other four sit "
               "under significant ones. Verified against the paper, "
               "2026-09-06."),
    Standard(
        key="bartoli_badge_2022",
        evidence=CAUSAL, interest=DISINTERESTED,
        domain="interruption",
        claim="A notification badge decomposes into two mechanisms the "
              "authors treat separately: a salience bias from the badge's "
              "physical properties (colour, shape, position), and an urgency "
              "bias from its implicit semantic content. Badge presence was "
              "randomised and drove first click (p < .001).",
        source="Bartoli, D., Benedetto, S. (2022). PLOS ONE 17(6):e0270888. "
               "Remote between-subjects, 1,095 recruited and 86 excluded, 15 "
               "groups, one Android homescreen of 15 icons with exactly one "
               "red numeric badge per condition.",
        url="https://doi.org/10.1371/journal.pone.0270888",
        status=PARTIAL, checked="2026-09-04",
        limits="Its value here is the decomposition, not the effect: only the "
               "urgency half is form-independent, which makes it the one "
               "badge mechanism that can reach a text interface at all. But "
               "no manipulation isolates colour, shape, position or numeric "
               "content -- the authors defer exactly that to future work -- "
               "so the split is their framing rather than their result. "
               "Ecological validity is low: static screenshots and a "
               "hypothetical first click, no live device use, so magnitudes "
               "must not be carried into behavioural claims. Johannes et al. "
               "(Collabra: Psychology 5(1):14, preregistered, N = 117) found "
               "social app icons including a notification sign did not impair "
               "visual search, which qualifies any general attentional "
               "capture story."),
    Standard(
        key="schaffner_dark_pattern_effects",
        evidence=CAUSAL, interest=DISINTERESTED,
        domain="interface",
        claim="Across 148 experimental units in 27 papers published 2019-2025, "
              "dark pattern effect sizes vary enormously -- relative increases "
              "from 112% to 1,500%, median 144% -- and no correlation emerged "
              "between effect size and pattern type or domain. The authors "
              "attribute the variation to strong context dependence.",
        source="Schaffner, B., Heysen, L., Chetty, M. (2026). A Systematic "
               "Review of User Experiments Measuring the Effects of Dark "
               "Patterns. CHI '26.",
        url="https://doi.org/10.1145/3772318.3790383",
        status=VERIFIED, checked="2026-09-04",
        limits="This is the methodological objection to counting patterns in a "
               "corpus at all. The field's evidence for harm is experimental, "
               "and if effect size does not track pattern type then an "
               "occurrence count cannot be converted into an expected effect: "
               "n occurrences of a named pattern implies nothing about impact "
               "without an experiment that was not run. The authors are not "
               "arguing dark patterns are harmless -- 85% of the experiments "
               "they reviewed found significant effects -- and they credit "
               "observational work with ecological validity that experiments "
               "trade away. The dispute is with inference from frequency, "
               "which is exactly what a transcript count does."),
]


# DISSENT. These argue against something above. They are here because a
# standards library assembled only from agreement is a rhetorical device rather
# than an instrument -- it would return whatever it was built to return, and
# the way to tell the difference is whether anything in it can lose. Each entry
# below names the key it cuts at.
DISSENT = [
    Standard(
        key="kivimaki_who_ilo_rebuttal",
        evidence=ASSOCIATIONAL, interest=DISINTERESTED,
        domain="working hours",
        claim="The WHO/ILO expert group did not correctly apply its own "
              "framework for assessing strength of evidence, and the "
              "association between long working hours and ischaemic heart "
              "disease is subject to marked effect modification by "
              "socioeconomic status: the authors report no increase among "
              "people working long hours in high socioeconomic status "
              "occupations, and argue the conclusion should be restricted to "
              "low socioeconomic status occupations.",
        source="Kivimaki, M., Virtanen, M., Nyberg, S.T., Batty, G.D. (2020). "
               "The WHO/ILO report on long working hours and ischaemic heart "
               "disease -- Conclusions are not supported by the evidence. "
               "Environment International 144:106048, November 2020. A reply "
               "in the same journal that published the estimates.",
        url="https://doi.org/10.1016/j.envint.2020.106048",
        status=PARTIAL, checked="2026-09-04",
        limits="Verified against the paper, 2026-09-06 (a CC-BY copy via the "
               "Helsinki research portal): the WHO/ILO expert group did not "
               "correctly apply its own framework for assessing strength of "
               "evidence, confirmed directly. The hazard ratio for high "
               "socioeconomic status occupations appears twice in the paper "
               "-- an age- and sex-adjusted 0.85 (95% CI 0.63-1.13) early on, "
               "and 0.85 (95% CI 0.63-1.15, n = 186,079) in the authors' own "
               "updated socioeconomic-stratified meta-analysis, their Table "
               "1 -- and both say the same thing: no excess risk in that "
               "group. This cuts at the cardiac outcome specifically and not "
               "at the hours count."),
    Standard(
        key="bianchi_burnout_depression",
        evidence=ASSOCIATIONAL, interest=DISINTERESTED,
        domain="outcome",
        claim="Burnout may not be separable from depression. Reviewing 92 "
              "studies, the authors find the evidence for burnout as a "
              "distinct nosological category weak, and argue the overlap is "
              "large enough that treating burnout as its own condition is not "
              "established.",
        source="Bianchi, R., Schonfeld, I.S., Laurent, E. (2015). "
               "Burnout-depression overlap: A review. Clinical Psychology "
               "Review 36:28-41.",
        url="https://doi.org/10.1016/j.cpr.2015.01.004",
        status=PARTIAL, checked="2026-09-04",
        limits="Verified against the paper, 2026-09-07 (an author-hosted "
               "copy via ccny.cuny.edu): the abstract states it almost "
               "exactly, calling the distinction between burnout and "
               "depression 'conceptually fragile' and describing evidence "
               "for the distinctiveness of the burnout phenomenon as "
               "inconsistent across the 92 studies. This cuts at "
               "icd11_burnout from the opposite side to "
               "the one that entry already concedes: ICD-11 declines to call "
               "burn-out a medical condition, and this argues that where it is "
               "measured it may be measuring depression. Either way a "
               "burnout-shaped reading of a work pattern is not a diagnosis, "
               "and nothing in this library should be used as one."),
    Standard(
        key="van_dongen_2004_individual",
        evidence=CAUSAL, interest=DISINTERESTED,
        domain="rest",
        claim="Neurobehavioral impairment from sleep loss differs "
              "systematically between people and is stable within a person "
              "across separate exposures. The differences are trait-like "
              "differential vulnerability rather than a consequence of sleep "
              "history, and they cluster on three dimensions: self-rated "
              "sleepiness and mood, cognitive processing capability, and "
              "behavioural alertness.",
        source="Van Dongen, H.P.A., Baynard, M.D., Maislin, G., Dinges, D.F. "
               "(2004). Systematic Interindividual Differences in "
               "Neurobehavioral Impairment from Sleep Loss: Evidence of "
               "Trait-Like Differential Vulnerability. Sleep 27(3):423-433.",
        url="https://doi.org/10.1093/sleep/27.3.423",
        status=PARTIAL, checked="2026-09-04",
        limits="Verified against the paper, 2026-09-07 (academic.oup.com): "
               "the authors' own conclusion states that interindividual "
               "differences in neurobehavioral responses to sleep "
               "deprivation were not merely a consequence of variations in "
               "sleep history, but involved trait-like differential "
               "vulnerability to impairment from sleep loss, and the three "
               "dimensions match exactly. It cuts at the way this "
               "library uses van_dongen_2003: a population dose-response curve "
               "does not transfer to an individual, and the same restriction "
               "produces materially different impairment in different people. "
               "An hours-of-rest number can therefore say what the population "
               "average would do and cannot say what one person's deficit is. "
               "Note it cuts both ways -- an unusually vulnerable person is as "
               "consistent with this finding as a resilient one, so it is not "
               "a reason to discount a rest figure, only to stop reading it as "
               "a personal prediction."),
    Standard(
        key="fitz_variable_ratio_null",
        evidence=CAUSAL, interest=DISINTERESTED,
        domain="interruption",
        claim="The slot-machine framing of notifications is asserted rather "
              "than tested, and the same experiment's own data cut against "
              "it. The hourly-batching arm -- predictable, variability "
              "removed, frequency largely preserved -- did not differ from "
              "control on anything except a single feeling-interrupted item. "
              "If unpredictability per se were the active ingredient, hourly "
              "batching should have helped.",
        source="Fitz et al. (2019), Computers in Human Behavior 101:84-94, "
               "read against its own framing.",
        url="https://doi.org/10.1016/j.chb.2019.07.016",
        status=PARTIAL, checked="2026-09-04",
        limits="Cuts at the variable-reward story that gets attached to "
               "nearly everything in this field, including to this library's "
               "own attention-capture entries. No reinforcement construct is "
               "measured anywhere in the instrument: no craving, habit "
               "strength, reward learning or checking compulsion. Two "
               "terminology corrections worth carrying: the paper never says "
               "variable-ratio, it says intermittent reinforcement and cites "
               "Skinner and Ferster for the variable-interval schedule, while "
               "a slot machine is canonically variable-ratio, so the source "
               "conflates them. And its own formulation includes varying "
               "personal relevance, a content property, so reducing the "
               "mechanism to arrival schedule drops half of it."),
    Standard(
        key="allcott_correlational_gap",
        evidence=CAUSAL, interest=DISINTERESTED,
        domain="method",
        claim="Within one preregistered randomised trial, the cross-sectional "
              "correlation between platform use and subjective well-being is "
              "about three times the experimentally estimated causal effect "
              "-- roughly 0.23 SD against 0.09 SD -- and the two estimates "
              "differ with high statistical significance.",
        source="Allcott, H., Braghieri, L., Eichmeyer, S., Gentzkow, M. "
               "(2020). The Welfare Effects of Social Media. American "
               "Economic Review 110(3):629-676, p.655. Preregistered RCT "
               "(AEARCTR-0003409), n = 2,743, public replication files.",
        url="https://doi.org/10.1257/aer.20190658",
        status=PARTIAL, checked="2026-09-04",
        limits="This is the objection to every observational number in this "
               "kit, pull.py included, and it is here for that reason. It "
               "must not be used as a multiplier: the authors explicitly "
               "refuse to generalise the ratio to other studies, and they say "
               "the gap is consistent with reverse causality, or omitted "
               "variables, or a short-term against long-term difference -- "
               "their leading interpretation, not a demonstrated fact. It is "
               "a within-sample replication of the correlational strategy, "
               "not a measured bias for the literature. The 0.09 SD is a "
               "complier local effect over four weeks around the 2018 US "
               "midterms. And it is contested in the other direction: "
               "Bursztyn, Handel, Jimenez-Duran and Roth (AER "
               "115(12):4105-4136, 2025) argue individual deactivation holds "
               "the network fixed and can therefore understate the "
               "platform-level effect. Verified against the paper, "
               "2026-09-07 (an author-hosted copy via web.stanford.edu): "
               "the 0.23-vs-0.09 comparison, the 'about three times larger' "
               "framing, and all three named interpretations match the "
               "paper exactly, in the same order."),
]

ALL = OCCUPATIONAL + SLEEP + DARK_PATTERNS + DISSENT
BY_KEY = {s.key: s for s in ALL}


# get returns one standard, raising rather than returning None: a report that
# cites a key that does not exist should fail loudly at generation, not print a
# number with a blank source under it.
def get(key):
    if key not in BY_KEY:
        raise KeyError(f"no standard {key!r}; have {sorted(BY_KEY)}")
    return BY_KEY[key]


def main():
    ap = argparse.ArgumentParser(description="The thresholds and their sources.")
    ap.add_argument("--check", action="store_true",
                    help="exit non-zero if anything is unverified")
    a = ap.parse_args()
    if a.check:
        # PARTIAL is not RECALLED. Reporting them together would either hide a
        # citation nobody has read or overstate one that is mostly checked, and
        # the whole point of the field is that those are different states.
        part = [s for s in ALL if s.status == PARTIAL]
        unver = [s for s in ALL if s.status == RECALLED]
        for s in part:
            print(f"PARTIAL   {s.key}  -- some fields unconfirmed at source")
        for s in unver:
            print(f"RECALLED  {s.key}  {s.source[:70]}")
        ver = len(ALL) - len(part) - len(unver)
        print(f"{ver} verified, {len(part)} partial, {len(unver)} recalled")

        # A set of citations that all point the same way is the failure this
        # library is most likely to have and the least likely to notice.
        # Counted on membership of DISSENT rather than on a stance field:
        # stance is a property of a citation, and whether the library was
        # assembled with anything in it that can lose is a property of the
        # library. Printed and not graded: there is no published figure for how
        # much of a citation library ought to argue with itself, so inventing a
        # pass mark here would be the thing this kit exists to point at.
        print(f"dissent strength {dissent_strength():.0%}  "
              f"({len(DISSENT)} of {len(ALL)} sources dispute another entry)")
        for d in DISSENT:
            print(f"          {d.key}")
        return 1 if unver else 0
    for s in ALL:
        mark = {VERIFIED: f"verified {s.checked}",
                PARTIAL: f"partly verified {s.checked}",
                RECALLED: "RECALLED -- not checked at source"}[s.status]
        thr = f"  [{s.threshold} {s.unit}]" if s.threshold else ""
        against = "  disputes" if s in DISSENT else ""
        print(f"\n{s.key}  ({s.domain}){thr}{against}")
        print(f"  {mark}; {s.evidence}, {s.interest}")
        print(f"  claim:  {s.claim}")
        print(f"  source: {s.source}")
        print(f"  limits: {s.limits}")
    return 0


if __name__ == "__main__":
    sys.exit(main())
