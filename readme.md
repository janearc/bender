# robots, everywhere

`bender` is just a distribution for people to see a portfolio of stuff i work
on. i didn't actually build any of this for distribution or public consumption.
as it happens, i use this stuff almost every day, and it occurs to me it 
demonstrates a knowledge of, and ability to build, the entire stack from infra
to clients and apps.

# what have we got here?

| member | lines | share | |
|---|---:|---:|---|
| puffin | 44,367 | 40% | `############################################` |
| flipr | 16,397 | 15% | `################` |
| kingfisher | 15,700 | 14% | `################` |
| cassowary | 7,387 | 7% | `#######` |
| starling | 6,200 | 6% | `######` |
| libdaffy | 6,134 | 6% | `######` |
| angry | 6,008 | 5% | `######` |
| game | 5,670 | 5% | `######` |
| libtheme | 3,546 | 3% | `####` |
| **total** | **111,409** | | |

| language | lines | share | |
|---|---:|---:|---|
| go | 83,316 | 75% | `############################################` |
| python | 23,987 | 22% | `#############` |
| typescript | 1,968 | 2% | `#` |
| protobuf | 1,333 | 1% | `#` |
| shell | 805 | 1% | `#` |
| **total** | **111,409** | | |

# how did we get to *this* stack

when i built this stuff, i started with what i needed to build software:
i needed infrastructure, i needed kubernetes, i needed a microservice mesh.
a lot of folks say this is overkill and complicated and takes up too much
resources. i don't mean to be glib, but this is what we call a "skill issue."

i told myself, "i want uber's 2015-16 network," because that gave me the
tools to build, tag, and deploy that i wanted, and the mesh itself dictated
what talked to what. it handled naming, scaling, configs, automation. to be
quite honest, i'm not sure how people build anything without kubernetes, from
the start.

# a note on naming

my friend bob nugmanov used to tell me there are three hard problems in
computer science: numbering and naming. most of the projects i build lately
are named after birds. it's important to not read too much into this. but this
is how you have projects named daffy, angry, and cassowary alongside projects
like flipr and game. moving right along...

# the very bottom of the stack

infrastructure. at uber we had some stuff we built ourselves, then we had
apache mesos, and we had some docker clusters. but it's 2026, and i built
everything with kubernetes.

bender ships the ability to build all of this stuff into docker, and if you
wanted to deploy it on your network, you could do that, and everything would
just work, which is in fact the promise of docker, and kubernetes. i fit this
easily on my laptop inside 6gb of ram, and a modest amount of disk.

- [flipr](https://github.com/janearc/flipr-dist): realtime config

  i have to admit that when i first encountered flipr at uber i was skeptical.
  it seemed like a kludge or a hack. but then when you realise that you can
  hit a json blob endpoint on a highly available service and it changes the
  behavior of your network, you see the beauty of it. i don't think flipr was
  ever distributed, and it's not clear where it came from other than it might
  have come from godaddy. i just built it from scratch and it's perfect and
  i love it.

  the way we make sure flipr is safe is we use ci gates. you use the approved
  flipr client, and you use gen code. we generate code from the protobuf
  declarations, and if you aren't using the gen code, or you modified it, the
  build fails. this way everyone is being honest.

- [starling](https://github.com/janearc/starling-dist): agentic bus

  when we build software with agents, it is important to know what they are
  doing and in fact what they are thinking. it's pretty useful to have logs of
  what they've said to eachother, and after a couple incidents where agents got
  kind of "weird" we decided we needed a way to manage that channel. starling
  gives agents "tickets," and the ticket constitutes a channel through which
  the agents can communicate. starling posts logs to pgrest, and they get their
  work done like normal. this also allows agents on non-overlapping networks to
  coordinate work.

# backend: who is doing the work

i am a backend engineer by trade, i love working there, i love the feeling of
doing huge amounts of work, quietly, thanklessly, just passing data around.

- [kingfisher](https://github.com/janearc/kingfisher-dist): high speed maps

  i have a few applications that are based on deckgl and h3. i'm proud of the
  fact that when i was at uber, i was on the team that built this lib when we
  shipped it. it's fast, it's functional, it's beautiful, and i love that a
  decade and more later, it's on my network, doing the same thing. this is
  a python service, and that's okay. i may one day build it in golang, but
  that's not today.

# libs

when you decide you want to build an interface, you start asking lots of
questions like, "what is this going to look like" or "how am i going to draw
shapes" and "what constitutes a window here?" so this breaks down into a couple
component parts upon which everything else is built.

- [libtheme](https://github.com/janearc/libtheme-css) color and themes

  it turns out color is very complicated. and i find that i use it in places
  i don't even realize i'm using it. smart lights. vim theme. zsh theme. tmux
  theme. web interfaces. each separate 'space' has its own understanding of
  color. so i went and abstracted all of these different colorways into 
  primitives, and built translation layers between them. libtheme is the
  reason we can theme grafana, which inexplicably refuses themes. but libtheme
  is only color, it doesn't draw anything.

- [libdaffy](https://github.com/janearc/libdaffy) sprites, rendering, shapes

  i was building tui applications to monitor my kubernetes and agents, and 
  then i began to be very dissatisfied with my ability to create interfaces.
  drawing is hard in the console. this required making a library to draw 
  shapes, it required a file format, it required a definition for what those
  shapes are and can do. it also gives wireframes and it gives "effects,"
  which is just another kind of sprite, and we translate this with css from
  libtheme. libdaffy is the basis of *`daffy`*, a drawing/painting program in
  the console which allows me to build the wireframes and sprites i use in my
  interfaces.

# applications

it will come as no surprise that monitoring is required with this much
infrastructure. i find it a tiny bit ironic that i am unemployed, but i have
full grafana dashboards, i have alerts, i have a production network, i have
a dev network, and i have the same rigor as i would with anyone's enterprise
network, at home, on my laptop.

- [puffin](https://github.com/janearc/puffin-dist) observability, c2

  there's so much stuff going on in this environment that it becomes impossible
  to manage everything through each individual interface. puffin is a bit of
  an aggregator. puffin knows where everything is managed, and presents it in
  one interface. it does this mostly asynchronously. like everything else on
  the network, it has exponential backoff with jitter built into every call.
  it uses libdaffy to draw its sprites, it uses libtheme for color, it uses
  flipr to talk to flipr, and it uses the native golang client to talk to
  kubernetes. when i initially built puffin, it was a bit of a lark, "can i
  actually build a monitoring application with a tui?" and it became the
  biggest project on my network, the one i use the most, and the one i am
  most involved with. puffin understands agent lifetimes, where their data is,
  what their status is, who they are talking to, and so on.
 
- **angry** *tui streaming maps client / benchmark*

  because i have kingfisher, and kingfisher needs automated testing and
  benchmarking, i needed way to do this in a non-interactive way. browsers
  are notoriously not good at this. so we built the tool and along the way we
  realized that we had in one project a maturing sprites library, in another
  we had a rasterizing/rendering library, and we had angry, which let us do
  both, streaming. the natural outcome of this is streaming sprites and
  rasterizing, which puts you in an unusual position where you get to ask
  questions that i'm not sure many people have been fortunate enough to ask:

  - could we build a flight simulator in the console?
  - could we build an animation suite on this and generate video in json?

  i don't really feel the need to build either of those right now, but i like
  that i could build either of those in an afternon if i was motivated. and
  that's kind of a defining theme of how i build for myself: i don't like
  being told that i can't. i want to be flexible enough to pivot on very
  short notice, and to feel good after the pivot.

  nb: all of the ansi and text art in this package was built from scratch by
  the rasterizing / rendering pipeline. it seemed like shipping animated ansi
  art was a little over the top, but that exists as sprite streams that have
  alpha channels for ease of composition.

# tools

gosh we love tools as engineers. just like with starling, as we began to work
with agents, it became clear that we needed forensic tooling.

- [cassowary](https://github.com/janearc/cassowary) forensics, interrogation

  "what is that agent doing?" or, importantly, "what the hell just happened?"
  it turns out that agents make decisions. sometimes those decisions are
  unexpected. sometimes they can be frightening or dangerous. it's really
  important if you are using a lot of agents that you be able to audit what is
  happening there. typically i operate agents in teams. for example, the last
  big migration we did i had about 6 agents working in a team, managing
  succession and propagation. it took a couple days, the output was high
  quality, but also their behavior became unusual, and it was importan to
  understand how that happened, so that it could be prevented in the future.

  i built cassowary to assemble and profile the entirety of an agent team
  corpus (where 'agent team' is a claude code team).

  as we began to use cassowary it became clear that we needed much more.
  profiling of both operator (that's me!) and agent, the way communication
  worked, noticing trends (because of the way llms work, "sentiment" tends
  to propagate among agent teams as they communicate with eachother, and as
  agents propagate). cassowary attempts to quantify changes in agent behavior
  over time. we use this toolkit all day, every day, to make sure everything
  is working the way we think it is, and to catch problems before they become
  really big problems. while agents do not have emotions, per se, the way
  language works is that emotional valences drive behavior. so while we can
  say mostly conclusively that an agent does not experience emotion, we can
  definitively, as in we have quantified this, show that agents respond to
  emotion in communication. and when they talk to eachother, the language
  they use between them propagates. fundamentally, language is synonymous
  with contagion. cassowary is for this.

- [game](https://github.com/janearc/game) build, when you've got agents.

  agents mean well, i think. they do great work when they have sufficient
  instruction and support. but a lot of the code they produce needs extensive
  linting and shaping before it is suitable for release. we built game to be
  a build tool that can manage the "dirty" repositories that agents tend to
  produce, and give you a tagged "release" that is suitable for release.

  additionally, because of the depth of the stack here, if we wanted to build
  the tui maps client, we might have to build kingfisher, the flipr client,
  libdaffy so we have streaming sprites (yes, streaming sprites in the
  console), and our window manager, and we have to build the ghostty console
  which means we have to build zig and so on.

  `make` is not the tool for that. golang has a great build system, but it
  can't do that. as a unix person going back a long long time now, i know
  that if you think to yourself, "i need to reinvent a fundamental piece of
  the unix ecosystem," that you are wrong, 100% of the time. but there was no
  way to do this with make, so we did the not-smart thing, and we built game.

  but game lets us do things we didn't know we wanted: because all of this
  software belongs to me, is deployed on my network, and i am the sole
  consumer, i can say that beauty is important to me. whimsy is important to
  me. and if i want a build tool that can give us an attractive tui while it
  goes and builds stuff for an hour, or has an actual interface, that's just
  a win all around.

  the last thing that game gives us, that nothing else can give us, is
  integration with *`cassowary`*. cassowary gives us an agent aware `whoami`,
  which we then turn into the hook *`agent-attrib`*, which means an agent
  signs its commits in a derivable way. a nice consequence of this is we also
  get *`agent-blame`*, which means we can see where the commits come from.
  because `game` is agent-aware, this means each agent gets its own game config
  in its anchor directory. so you can spawn an agent and it effectively has
  per-product repository controls. this means if you are operating on a single
  machine (like i am, a laptop), and you have eight or whatever agents doing
  stuff, they are committing code, but it's not all `'jane'` or whoever you
  are. it's them, and they know it, and you know it. this is a hard problem to
  solve, and game solves it for us in addition to building software.

# where does this leave us?

when you build software this way, there's a very long, somewhat steep, cost 
you pay up front: you build infra. you *have* to have your infra. you also
need your tooling. but once that is built, the velocity you have to build
anything else is unbelievable. i can spec out a new product, maybe 500 lines
of markdown, and specify which services it connects to, what its api is,
give it the style guide, the prose guide, the network policy, and ask it
to go do that. what comes back is high quality, it conforms to the
specification that *comes from the infrastructure*, not something the agent
decided.

i love building this way, and i love that i have a high speed maps delivery
pipeline that streams sprtes to a tui. i love that it took just a couple hours
to go from "can we even do this?" to "a theme would make this even better."

it's high speed development of robust, production-ready, deployable and
scalable software.

we do it with agents, but that's how we build software in 2026. i have
presented 111kloc here, and i am managing about 250kloc as of writing this.
for comparison, if we go back to uber, the entirety of uber was about 500kloc
in 2015. today, i manage half that much, by myself with two teams: production,
which keeps flipr happy and makes sure the network is stable, and product.
production has four fable-5.1-xhigh, and product has two fables and two opuses.

i'm confident we can build and ship anything we can imagine, and deploy it
almost anywhere, at just about any scale.
