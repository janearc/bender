#!/bin/sh
# bootstrap.sh: build game out of this clone, and then get out of the way.
#
# nothing is installed and nothing is fetched. go builds game from the
# game directory beside this file, and game does the rest: the go.work
# here is what lets the members see each other without the network.
#
# it needs go and git on the path, and nothing else.
set -e

command -v go >/dev/null || { echo "bootstrap: go is not on the path" >&2; exit 1; }
command -v git >/dev/null || { echo "bootstrap: git is not on the path" >&2; exit 1; }

# the members are not here on the road: this file, the .game and the
# readme are, and the trees arrive only when game assembles a release.
# without this a person standing in the road gets a go module error
# about a directory that was never meant to exist here.
if [ ! -d game/cmd/game ]; then
	cat >&2 <<'EOF'
bootstrap: there is no game/ beside this script, so this is bender's
road rather than an assembled bender. the road carries the .game, this
script, the readme and the card; the members arrive when a release is
cut from it.

to build the stack from here:

	game release stack vX.Y.Z    assemble the members at their tags
	game release push vX.Y.Z     send it: needs --publish, and that
	                             word is the owner's to type

or clone what was published, which is what a reader does:

	git clone https://github.com/janearc/bender
EOF
	exit 1
fi

mkdir -p bin
echo "building game out of game/ with go; nothing is installed"
# the same two -ldflags game passes when it builds itself, so the binary
# can say which commit of this clone it came out of. without them the
# stamp is empty and anything that formats it as a time prints the zero
# value, which reads as a bug and is not one.
commit=$(git rev-parse --short HEAD 2>/dev/null || echo unknown)
built=$(date -u +%Y-%m-%dT%H:%M:%SZ)
go build -ldflags "-X main.build=$commit -X main.built=$built" \
	-o bin/game ./game/cmd/game

cat <<'EOF'

game is in bin/, built by go out of this clone and stamped with the
commit it came from, so `game version` names it.

now run

	./bin/game build stack --here

which builds every member in this clone, each in a game of its own, and
prints what came out clean. the members see each other through go.work,
so nothing is fetched.

to do the same thing in a clean room rather than on your own machine,
the Dockerfile beside this script is that room and its header says how.
a machine that has built go before will build this either way; a
container is how you find out whether anyone else can.
EOF
