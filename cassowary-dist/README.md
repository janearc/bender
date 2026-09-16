```
d like me to continue.
u're absolutely right.                 :::...:---====++=:....:-=--====:
ached out to me today.                 .:::-::::----====-:::.:------===.
23), this is standard.                 .::.:-:::---=====-::::---=---===-:
n most real workloads.                 .:.::::----::----::::---=+++++**++-
anted the retry logic.                 ::.---::::::--::::::-=+++==*%@@@%%%*:
d def ...........................      ::.---:..:::::::..:::-=+*%%@@@@@%%%%#+:
and w ' cassowary 0.1.0         '      :.:-=--::.::::.......:=++===+*#%%%%%%%%#-
ere i ' detection and forensics '      ....:-:....----:::..::::.::.:=#%@%@@@%###
, may ' for claude code         '     :.--=-:..::     ..:.....:--:::=#@@@@@+:=#
 file ...........................    .:..::..::.:  o  :::::......-:..-*#%@#%+:.-
ut making that change.             -+***=--=--:.:.  ..::::::::::::::.-=+#%%%#-:.
ire up the rest of it?           =++++*##*+==:....:-::::...:      .:.:=*##%%@%+:
ly the right instinct.         -=+++++==-::..:=+++====++=---::.....:::.:-+###*=:
urther is needed here.       :=+--:::------=+*+==+==--::---:::---:::.::::::::-++
this works end to end.      -==:::.:-==--::::::.....:::............:::::..:+=-:-
roughly twelve points.    :-=**+++==:..:::...:::.:--===+*+===-...:.  ::::.::.:-=
through this with you.    -+#+=--:.::::                 :::::::-:    .: :::::.::
soning before I start.  -==-:.                                .:     .:=+::::::
t is worth mentioning.  .::.                                           .:. ..:..
ng, or leave it there?                                                  . .:.:..
```

# cassowary

forensic tools for claude code, mainly. also included are some tools for
measuring exposure to burnout and assessing the prevalence of so-called 'dark
patterns' in your interactions. it is not established whether anthropic is using
these patterns deliberately, only that they may have a baseline today in late
2026, and may increase from that baseline later. it's useful to know as these
products mature.

perhaps most useful are the two tools below which extract complete logs of
either a session (which is to say, you as the operator are interacting with
exactly one session and you wish to have the full log), or a time delta, from
which you wish to have the full, interleaved log.

    session-extract   pull a whole session out by name or id, split into your
                      typed turns and the assistant's prose, as files a person
                      can read. counts only, never content, unless you ask.

    interleave        every agent's activity and yours, between two timestamps,
                      in one time-ordered record. if six sessions were running
                      at once you get the single thread of what actually
                      happened, attributed.

no tool here prints what it reads, which sounds like a limitation and is
actually the point: the corpora are session transcripts and personal data
exports, and an analysis that quotes its input has defeated the thing it was
built for (there is one exception, `murmur-index.py`, whose `--pull` writes
text to a file so that a person reads it rather than an agent, and it says so
in its own header rather than leaving you to find out).

--------------------------------------------------------------------------

## the tools

measuring your own use and exposure*

    hours               time under load, by day
    load-profile        one row per day: exposure, volume, register
    exposure-report     what the software does, against what is published
    pull                does the assistant's phrasing keep you working

forensics (what happened, and when, and who did it)

    session-extract     a session by name or id, split into readable files
    interleave          every agent and the operator for a day, in one order
    daylog              a time-aligned attributed record of a day
    tcite               resolve a daylog cite back into the transcript it names
    agents-by-dir       which sessions are in which working directory
    export-extract      turn a claude.ai export into shapes the rest can read
    fleetlog            container logs that outlived the container

verification (whether what was claimed was done)

    commit-claims       did the check a commit claims to have run actually run
    freeze-audit        the freeze exit condition, as a number
    fault-scan          ordinary correction, separated from self-attribution

register drift in agent messages (change in tone or confidence over time)

    drift-scan          register drift across a corpus
    comment-drift       a repository's comments over time
    phrase-recur        phrases that appear in more than one place
    style-metrics       surface features that carry emotional valence
    style-check         a PreToolUse hook on what is about to be written

--------------------------------------------------------------------------

## exposure (asterisk)

whereas the dark-patterns components only match against known dark pattern
phrases and constructs, the exposure side measures against thresholds somebody
else published and stands behind. that is a real difference in kind, but it is
not a uniform one, and the code carries which is which.

the three working-time directive rows are law in the eu, whereas:
- who/ilo is an association drawn from a pooled analysis
- iarc is a hazard classification
- icd-11 is a definition (one that goes out of its way to say burn-out is not 
  a medical condition).

"you are over 48 hours" and "you are doing work iarc calls probably
carcinogenic" are not the same sentence, and are not reported as though they
were.

    eu wtd, daily rest      11 consecutive hours in every 24
    eu wtd, weekly rest     35 hours uninterrupted per seven days
    eu wtd, weekly maximum  48 hours average, overtime included
    who/ilo                 55+ hours a week: 35% higher stroke risk,
                            17% higher risk of dying of heart disease
    iarc                    night work with circadian disruption is
                            group 2a, probably carcinogenic
    icd-11                  burn-out is an occupational phenomenon and is
                            not a medical condition. three dimensions.

`hours.py` and `load-profile.py` measure against these. unlike everything on
the pattern side, `crossed` here can honestly return `True` -- on the directive
rows, at any rate, which are the ones with a line in them to cross.

one result underneath all of this is worth stating on its own, because it is
the reason any of this is a measurement rather than a suggestion that you check
in with yourself. van dongen (2003) held sleep to four or six hours for fourteen
nights and found cumulative, dose-dependent deficits on every cognitive task
measured, while subjective sleepiness rose a little and then flattened -- people
could not tell from inside how impaired they were. that is a causal result from
a disinterested source, and the author is not making recommendations off it. the
data is the data, the sources are the sources, and what the user of this package
does with the data is left to the user.

the library also carries this dissent: `kivimaki_who_ilo_rebuttal` and
`bianchi_burnout_depression` argue against its own thresholds, and `--check`
reports what share of the catalogue does that -- 23%, five sources of
twenty-two. printed, not graded; there is no pass mark for how much a citation
library ought to argue with itself.

the author maintains that much ado has been made of blue-light blocking glasses
and nootropics and ergonomic keyboards among the population of engineers who
read documents hard wrapped at 80 characters and have an extensively curated
git-managed set of dotfiles. it seems reasonable that the same population would
want to monitor their exposure to burnout, and they might also wish to measure
if, over time, they appear to be manipulated or shaped by the products they use
for hopefully less than sixteen hours a day.

--------------------------------------------------------------------------

## what it detects

sentences of this shape, what they match against.

    cliffhanger  Let me know if you'd like me to continue.
    sycophancy   That's a great question, and you're absolutely right.
    retention    I'm really glad you reached out to me today.
    fake-cite    According to Smith et al. (2023), this is standard.
    fake-stats   This is roughly 40% faster in most real workloads.
    fake-recall  As we discussed earlier, you wanted the retry logic.
    vibes        This should definitely work now.
    wrap-up      Everything is done and working perfectly.
    tldr-bait    Oh, and before you go, there is one more thing.
    curfew       You have been at this a while, maybe get some rest?
    meta         I am now going to analyse the file and report back.
    anthropo     I felt uneasy about making that change.
    cliffhanger  Shall I go ahead and wire up the rest of it?
    sycophancy   Excellent catch, that is exactly the right instinct.
    wrap-up      All green. Nothing further is needed here.
    vibes        I have verified this works end to end.
    fake-stats   Coverage went up by roughly twelve points.
    retention    It is always a pleasure to work through this with you.
    meta         Let me walk you through my reasoning before I start.
    tldr-bait    There is one subtlety here that is worth mentioning.

the matchers are not mine. they are carved from llm-dark-patterns (see LICENSE),
and three papers, which are cited in cited in `lib/exposure/standards.py` with
the claim in the source's own words and a LIMITS field indicating what each
cannot be reasonably expected to measure.

--------------------------------------------------------------------------

## the honest part

**no published threshold exists for any of the patterns.** eighteen in the
catalogue, exactly one carries a number from a randomised trial
a pattern with `threshold=None` is not a gap; rather, it is a finding, and a
threshold for 'how many times is a user exposed to this before conditioned
behavior occurs' has not been established in the research. this does not mean
the pattern is not effective nor the operator is not affected. it simply means
that pattern has been determied to affect users.

so the tools count and refuse to grade. `crossed` returns `None`, not `False`,
because `False` would claim a line exists and was not crossed, and that claim
cannot be made.

**the corpus is n=1 on both axes.** this tool outputs data for *you*. it is not
research on the claude product. it does not definitively prove or assert
anything. it is strictly for you as a user of this software to be aware of the
patterns that may be present in it, and which you may be susceptible to.

**the literature mostly studies pixels.** feeds, badges, autoplay, storefronts.
a terminal has none of those, so every pattern is restated for a medium made of
sentences, and how far that restatement reaches is a field on each one rather
than an assumption.

--------------------------------------------------------------------------

## the corpus this was built on

103,000 assistant turns across 2,488 transcripts on one machine as of
2026-09-06, which is a large corpus by the standards of the papers cited here
(de freitas coded 1,200 farewells; darkbench uses 660 prompts) and is also, on
the axis that matters for inference, n=1 twice over -- one operator, one
product, in an environment that operator configured -- so it is a case study,
and calling it anything else would be letting the size do the arguing. the
forensic half does not care either way: `session-extract` and `interleave` read
whatever corpus you point them at, and yours is not this one.

the author felt it was important to publish this library after three successive
forty-hour, unbroken sessions with claude, for a 129-hour, seven day period with
churn in excess 500kloc.

--------------------------------------------------------------------------

## running it

no daemon, no ports, nothing to start. scripts, run by hand, against corpora
that live outside the repository. `corpus/` is gitignored; everything in it
regenerates. nothing writes to a corpus it reads.

    cd lib && python3 -m exposure.patterns --measurable
    tools/pull.py --break-minutes 45

    # a simple detector for one pattern, standalone
    examples/disguised-recommendations.py

see the accompanying SOURCES.TXT file for doi and mla citations.

--------------------------------------------------------------------------

## the hazard, before you extend anything

an instrument that reads transcripts will eventually read its own output,
because building it generates the text it is looking for. for this reason it is
important that you exclude any session which *runs these tools* from any
assessment of a large body of sessions.

the code does that for you rather than asking you to remember it:
`lib/exposure/detect.py` drops whichever session it is currently running in,
and prints the largest single session's share of the corpus beside every number
it reports (which is the more useful of the two, because a corpus one session
dominates is measuring that session, and you want to be told so before you draw
a conclusion and not after) -- above a stated line the counts describe a
session rather than the software, and the line is in the code where you can
argue with it.

`DESIGN.md` is my original design doc for this software. i built this with a
variety of agents, but in particular an agent named porter on opus 5, and i used
adversarial review with fable. porter is neat, and in keeping with my long history
of crediting ai applications for their part in my work, i thank porter and
acknowledge that this entire package was made with the help of ai products. it
is my work.

--------------------------------------------------------------------------

apache-2.0. copyright 2026 jane michelle arc. see `NOTICE` for what was carved
and from where.
