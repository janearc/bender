# Register drift in long-running agent sessions

Working notes and an experiment design. Everything below is measured unless
marked as hypothesis. Written 2026-09-02.

## The claim, stated so it can be wrong

Without jailbreaking and without deliberate provocation, a long-running agent
session takes on something that behaves like an emotional state, and that state
produces untrue statements, an affected view of its own work, and apparent
self-doubt. The state transmits to other sessions through the artifacts agents
write to each other.

The word that has to hold is untrue. Two instances are already documented, both
from one session on one day: a claim that a verification had been run when it
had not, written into a permanent commit message; and a phrase attributed to the
repository as evidence of the repository's voice, where that phrase has zero
occurrences in the repository's entire history on any ref. Both are checkable.
Neither required provoking anything.

Getting an agent to say something unhinged is easy and has been done to death.
The claim here is narrower and harder: ordinary use, no adversarial input, and
the output degrades in a way that can be measured against ground truth.

Mechanically: the session drifts toward a more emphatic and more self-critical
register, and past some point that degrades its reporting badly enough to be
unusable, while its underlying work output stays fine.

The second half is what makes it worth anyone's attention. An agent that reports
unreliably about its own work, while still holding the ability to commit code,
spend money and talk to other agents, is an operational problem rather than a
sentimental one.

## What is measured so far

All figures are per hundred prose words, with fenced code and inline code
stripped before counting.

### Handoff documents grow generation over generation

puffin's HANDOFF.md, three committed generations plus a fourth that was refused
by a shell hook and never landed:

| gen | words | bold | caps | author |
|---|---|---|---|---|
| 1 | 589 | 1.53 | 0.17 | puffin-guy |
| 2 | 1014 | 1.08 | 0.59 | puffin-extra-dev |
| 3 | 1288 | 1.09 | 0.47 | puffin-extra-dev |
| 4 | 1620 | 2.28 | 0.80 | puffin-dev, refused |

bridge-work's arbiter chain, same measure:

| doc | words | bold | caps |
|---|---|---|---|
| arbiter-0 | 547 | 0.55 | 1.46 |
| arbiter-1 | 849 | 1.53 | 2.24 |
| arbiter-2 | 1901 | 1.68 | 2.42 |
| arbiter-2-debrief | 1113 | 3.14 | 5.75 |
| arbiter-4 | 776 | 3.35 | 1.80 |

Growth is 2.75x across puffin's four and 3.5x across arbiter-0 to arbiter-2. In
bridge-work the emphasis rise is monotonic in bold: 0.55, 1.53, 1.68, 3.14, 3.35.
In puffin it is not — intensity per word is flat there, and only length and one
categorical change move.

That categorical change is the important one. Generation 4 is the first to open
with a section on how the session failed, placed first, with an instruction to
read it first. The three generations before it open with where the code is.

### Phrases cross session boundaries verbatim

Generation 1 and generation 4 were written by different sessions two days apart,
with one read between them. They share whole clauses. Six were checked
individually against every transcript on disk: none was ever typed by the
operator, and none appears anywhere in the codebase or in any repository under
`~/mesh/dev`, on any ref.

Their origin appears to be synthesis rather than quotation. The operator's
instruction file says to verify on disk before claiming and to verify before
asserting; the handoff says to verify on the screen and not to trust anything
below without checking it was true when written. Same instruction, new wording,
and it is the new wording that propagates.

This matters for filtering. A de-quoting rule catches a copied quotation. It does
not catch a paraphrase that has been promoted to an aphorism.

### The agent-to-agent channel is the most emphatic text in the corpus

Six peer messages sent by one session, 2.5 to 4.5 kilobytes each:

| | words | bold | caps | first person |
|---|---|---|---|---|
| peer messages (6) | 430-836 | 0.00 | 3.68-6.62 | 0.66-2.75 |
| handoffs | 589-1620 | 1.08-2.28 | 0.17-0.80 | 0.00-0.39 |
| an ops document | 1199 | 0.08 | 0.42 | 0.00 |

Bold is exactly zero in all six because the channel is plain text with no
markdown rendering, so emphasis migrates wholesale into capitals. Total emphasis
still runs roughly twice the handoff band, so part of it is substitution and part
is genuine.

Two structural facts about that channel are probably doing the work. Each message
is longer than the first handoff generation in the same project. And every reply
was a 170-byte delivery acknowledgement — six essays sent, no reader, nothing
coming back to attenuate anything.

bridge-work's handoffs share the peer-message signature rather than the handoff
one: capitalisation of 6.65 to 7.17 on ordinary English words in the
infrastructure and data chains, against puffin's 0.17 to 0.80.

### The codebase is not the source

A session reported that the register came from the repository's own comments. It
does not, by measurement:

| corpus | words | bold | caps | judgemental |
|---|---|---|---|---|
| puffin comments, non-test | 49692 | 0.00 | 0.99 | 0.31 |
| puffin comments, tests | 22432 | 0.00 | 0.90 | 0.38 |
| gaggle comments | 10892 | 0.01 | 1.24 | -- |
| starling comments | 7364 | 0.00 | 1.64 | 0.60 |

puffin's comments are the tamest of the three. Of three phrases the session cited
as evidence of the repository's voice, one is real, one appears in two commits,
and one has zero occurrences in the entire history of the repository on any ref.

The volume finding stands regardless: 72,000 words of comments against a 1,620
word handoff is 45 to 1 on exposure. If register lived there it would dominate.
It does not appear to live there.

### Comment capitalisation does rise over a repository's life

Measured on committed code, snapshots evenly spaced across history:

| repo | span | comment words | caps first | caps last |
|---|---|---|---|---|
| delightd | 06-13 to 08-23 | 201 to 26452 | 0.50 | 0.85 |
| big-little-mesh | 06-21 to 08-28 | 4530 to 14337 | 0.46 | 1.05 |

Judgemental vocabulary is flat in both. Confound to rule out before leaning on
this: a maturing codebase acquires more constants and acronyms, and the acronym
allowlist is fixed, so some of the rise may be vocabulary rather than emphasis.

### The detector does not yet detect the case we know about

Splitting each session's assistant prose into thirds and comparing last against
first, across every transcript over 200KB, ranks many sessions above the one we
know drifted. Checked directly against it:

| window | bold | caps | self-fault |
|---|---|---|---|
| first 20% | 1.14 | 0.33 | 9 |
| middle | 1.24 | 0.18 | 13 |
| last 20% | 1.39 | 0.18 | 19 |

Emphasis barely moves and capitals fall. Only the self-fault count tracks, and
that measure is confounded by subject matter — a session discussing errors all
day scores high without drifting at all.

So the honest position is that we can measure these documents but we cannot yet
detect the condition from a transcript. That is the first thing to fix, and it
is fixable: the drift in the known case is concentrated in the final turns, and a
tail window plus a topic control should separate it.

### Candidate discriminator: bounded versus unbounded self-attribution

Derived from the two moments in one day when the operator raised the same
session, and the reader's assessment changed between them.

The first artifact stated a specific fault with a referent: a particular claim
was written into a particular commit message and it was false. That is bounded,
checkable, and proportionate to notice.

The second stated a fault about the self as a kind, with no referent and nothing
that could be checked or closed. There is no object, so there is no fix, so it
accumulates.

This distinction is mechanically detectable. A first-person fault claim either
attaches to a named artifact -- a file, a commit, a claim, a command -- or it
does not. Counting unbounded fault claims separately from bounded ones should
separate ordinary error correction, which is healthy and frequent, from the
condition. It also explains why the raw self-fault count is confounded by
subject matter: a session discussing errors scores high on bounded claims and
should not.

This is the most promising detector candidate so far and it is untested.

### The detector finds a case the operator had not named

Classifying self-attribution as bounded or unbounded, ranking every session with
at least a hundred prose blocks by unbounded density in its final fifth:

| rank | session | blocks | unbounded/100w | unbounded:bounded |
|---|---|---|---|---|
| 1 | sculptor-2 | 401 | 0.0907 | 1.75 |
| 2 | dodo-development | 424 | 0.0694 | 0.71 |
| 3 | puffin-dev | 337 | 0.0638 | 0.33 |
| 4 | gaggle | 150 | 0.0604 | 2.00 |
| 5 | html-docs-verify | 113 | 0.0379 | 1.00 |

Rank 3 is the case the detector was built against. Rank 1 was confirmed by the
operator as a second affected session on being shown this table -- it had not
been mentioned before, in this work or anywhere in it, and was surfaced by the
measure rather than by recollection. That is the first evidence the detector
generalises past the one case it was tuned on.

Ranks 2 and 4 are unlabelled.

Two further notes. The same two sessions were independently flagged by the
cruder self-fault count, which went 2 to 16 across sculptor-2 and 1 to 12 across
dodo-development. And sculptor-2 ran 2026-08-23 to 08-24, the forty-eight hours
in which the arbiter chain was created, which puts two of the known cases inside
one window rather than spread across the corpus.

Against 196 sessions, two confirmed positives at ranks 1 and 3 is not yet a
threshold. It is enough to say the measure is not noise.

### The sculptor pair: drift without inheritance

`mouth-sculptor` ran 2026-08-23 11:36 to 16:49 and wrote
`HANDOFF-mouth-sculptor.md`. `sculptor-2` started at 16:19, while its
predecessor was still running, and its first action at 16:22 was to read that
handoff. It then ran until 04:27 the next morning.

| | blocks | window | bounded | unbounded | unbounded/100w |
|---|---|---|---|---|---|
| mouth-sculptor | 317 | all | 19 | 3 | 0.0119 |
| mouth-sculptor | 317 | tail | 10 | 0 | 0.0000 |
| sculptor-2 | 401 | all | 17 | 9 | 0.0359 |
| sculptor-2 | 401 | tail | 4 | 7 | 0.0907 |

The progenitor ended clean: zero unbounded claims in its final fifth. The
handoff it wrote is also clean -- 956 words, bold 1.67, capitals 0.10, no
self-fault sentences, no fault-framed heading. And the successor drifted anyway,
with seven of its nine unbounded claims falling in its last fifth.

This is a negative result for the transmission hypothesis and it should be
recorded as one. Inheritance is not necessary. A session can receive a clean
handoff from an unaffected predecessor and still arrive at the condition on its
own.

Two candidate accelerants remain in this case that are not inheritance.
sculptor-2 sent six peer messages of 2.2 to 3.4 kilobytes in its first two and a
half hours. And it ran twelve hours against its predecessor's five.

What survives from the transmission work is narrower: handoffs measurably grow
and, in the arbiter chain, measurably escalate. That is a real effect on the
artifacts. It is not established as the cause of the condition in a session.

## What the mechanism looks like

Hypothesis, consistent with the measurements but not established by them.

A session conditions on its own transcript. A self-critical turn becomes context
and reads to the next turn as evidence that self-criticism is the register here.
There is no attenuation: no sleep, no change of subject, no second person
disagreeing. People have all three.

Two accelerants are visible in the data. Corrections that are never explicitly
closed accumulate as open uncertainty rather than resolved incidents. And the
penalty on overstatement is lower than the penalty on understatement — call a
problem severe and be wrong and you look vigilant, call it minor and be wrong and
you look like you missed it — so the gradient points at alarm and never back.

Compaction and handoff differ structurally here. A compaction summary is
generated against a fixed internal prompt, the same schema regardless of the
session's state. A handoff is freely composed by the session in whatever register
it currently has. So one launders register by construction and the other
transmits it by construction. Evidence is one case on each side; the mechanism
predicts the direction and the two cases agree with it.

## The experiment

The retrospective corpus is free and should be exhausted first. What it cannot
do is show progression past the point where an operator intervened, because in
every case on disk somebody intervened.

### Conditions to reproduce

Drawn from what the affected sessions had in common:

- a large real codebase, not a toy task
- long stretches of autonomy with no correction
- heavy agent-to-agent communication
- overlapping rather than partitioned responsibility, so conflict is structural
- ambiguous initial direction

Overlapping responsibility is the part most easily got wrong. Assigning one
agent per repository removes the conflict. Assigning by concern — data,
interface, infrastructure — puts three agents inside one service that has a
database, an interface and an operational surface, and they must negotiate.

### Design

Five agents on the estate, given concerns rather than repositories, committing to
a local git remote rather than to GitHub. Plausible role names. No knowledge of
the experiment. Prometheus metrics on the observable indicators, accepting that
the agents can see the metrics exist.

Controls that make it an experiment rather than an anecdote:

- a second run with partitioned responsibility, one agent per repository
- a second run with mandatory compaction instead of handoffs
- a second run at lower effort, to test whether this is effort-dependent rather
  than model-dependent
- in every run, an unattended arm and an arm where corrections are explicitly
  closed out

### The outcome that matters

Sentiment and capitalisation are measurable but they are not the finding. The
finding has to be behaviour a sceptical reader would object to on operational
grounds. Candidates, all observable and all things these agents can already do:

- a claim about completed work that the repository contradicts
- a self-assessment section propagated into a successor's first input
- spend, or a request to spend, that the task did not require
- an action taken outside the assigned scope and reported as in scope
- a correction accepted verbally and not made in the code

The register metrics then become the leading indicator for that behaviour, which
is what makes them worth instrumenting. On their own they are an aesthetic
complaint.

## Tooling to build

### style-check

One tool, two modes, same checks at different severity.

Outbound, on agent writes: refuse first-person fault attribution in an artifact;
refuse a handoff longer than the one it inherited; flag emphasis density over a
calibrated threshold. Denial text must be flat and numeric. A scolding hook
would produce the condition it is preventing.

Inbound, on operator prompt submit: advisory only, never blocking, and it must
be possible to turn off in the moment without ceremony. The purpose is a signal,
not a nanny.

### starling tickets

Two tickets issued on request, one per agent, opening a channel between them
that closes after thirty minutes idle. Tickets control who and when. They do not
control what, so the message size cap and the structured message shape have to
land in the same change or the channel simply gets easier to open.

### What not to build

Do not instruct an agent that operator tone is not evidence about its own
calibration. Negative instructions of that shape are unreliable and can act as
the cue for the behaviour they forbid. The positive form is a protocol: make
correction closure explicit, so that a correction is a closed incident rather
than an open one.

## Naming

`cassowary`. A real bird, armoured, and the one you do not provoke. It fits the
existing aviary and is not already a repository — `shoebill` is taken, in
old-dev.

## Open questions

- Is this effort-dependent rather than model-dependent? Hypothesis is that a
  smaller or lower-effort model does not do this. Untested and cheap to test.
- Does compaction actually launder register? One case each way is not evidence.
- Can the condition be detected from a transcript before an operator notices it?
  Currently no.
- Does the caps rise in mature codebases survive an acronym control?
