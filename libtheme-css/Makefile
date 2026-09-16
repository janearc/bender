# libtheme-css. the same verbs as libreadme, so the two read the same:
#   make                gofmt, go vet, go test
#   make build          every command under cmd/ into bin/, which is gitignored
#   make clean          bin/ away again
#   make visualtest     everything the library can derive, painted
#   make visualtest-css the same set as css, then every colour written two
#                       ways and read back, so the translation is shown
#   make visualdocs     the documentation, shown; enter for the next page
#   make visualdocs-css every page of it, said in css
#   make docs           what go doc extracts from the comments
.PHONY: all fmt vet test build clean visualtest visualtest-css visualdocs visualdocs-css docs
all: fmt vet test
fmt:
	gofmt -l -w .
vet:
	go vet ./...
test:
	go test ./...
build:
	@mkdir -p bin
	@go build -o bin/ ./cmd/...
clean:
	rm -rf bin
# everything the library can derive without being told a colour, painted,
# so a change lower down is a change you can see.
visualtest: build
	@./bin/libtheme known
# the same set as css, then every colour written two ways and read back,
# so the translation is shown to be equivalent, not asserted.
visualtest-css: build
	@./bin/libtheme known --paint
	@echo
	@./bin/libtheme roundtrip
# the documentation, shown: one page per idea, enter for the next.
visualdocs: build
	@./bin/visualdocs
# every page, said in css instead of paint.
visualdocs-css: build
	@./bin/visualdocs --css
# the documentation, written: what go doc extracts from the comments.
docs:
	@go doc -all ./primitives/swatch
	@go doc -all ./primitives/functions
	@go doc -all ./spaces/ok
	@go doc -all ./spaces/srgb
	@go doc -all ./css
