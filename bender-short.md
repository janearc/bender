# the stack, for presentation

from bottom to top, infra to app.

- **flipr** realtime config
- **starling** agentic bus
- **kingfisher** high speed maps
- **libtheme** color and themes
- **libdaffy** sprites, rendering, shapes
- **puffin** observability, c2
- **angry** *tui streaming maps client / benchmark*
- **cassowary** forensics, interrogation
- **game** build, when you've got agents.

each of these relies on everything else. we start at flipr, we build infra,
then we build libs, then we build apps, and we have tools for the environment.

i built every single line of code here. i am proud of this.

here is presented about 100kloc, and the 'estate' as the agents call it, is
about 250kloc. i love this environment.

agents split into two teams, product and production. production keeps infra
stable. product looks forward and builds cool stuff. we (product) respect
their stuff and our contracts with them, and production makes sure we have
a safe, stable infrastructure to build cool stuff.
