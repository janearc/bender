# the portfolio: a clean room for the whole stack.
#
# it carries a toolchain and nothing of this machine: no checkout, no
# module cache, no GOFLAGS, no GOPRIVATE. bender is cloned at run time
# and builds itself from within, which is how a stranger gets it.
#
# build it, then run the stack in it:
#
#	docker build -t portfolio .
#	docker run --rm portfolio sh -c '\
#		git clone -q https://github.com/janearc/bender /b && \
#		cd /b && sh bootstrap.sh && ./bin/game build stack --here'
#
# it prints a row per member and a count at the end. nine of nine is the
# whole stack. to try a clone that is not published yet, mount its bare
# repository read-only and clone from the mount instead:
#
#	docker run --rm -v /path/to/bender.git:/road/bender.git:ro ...
FROM golang:1.26.4-bookworm

# git because members init and clone real repositories in their tests;
# python3 because two members are python and their suites are the test
# targets their .game declares; ca-certificates for the module proxy.
RUN apt-get update \
 && apt-get install -y --no-install-recommends \
      git python3 python3-pip ca-certificates \
 && rm -rf /var/lib/apt/lists/*

# uv, because the python members run their suites through it and that is
# the house rule for python. pinned: an unpinned toolchain makes a clean
# room a different room every time.
RUN pip install --break-system-packages --no-cache-dir uv==0.12.15

# staticcheck at the version the stack's lint expects. a linter that did
# not run is not a linter that passed, and game refuses a release when it
# is missing.
RUN go install honnef.co/go/tools/cmd/staticcheck@v0.8.1

WORKDIR /portfolio
