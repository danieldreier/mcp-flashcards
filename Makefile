.PHONY: build build-all test clean release tag push-tag release-github

VERSION ?= $(shell git describe --tags --abbrev=0 2>/dev/null || echo "0.1.0")
NEXT_VERSION ?= $(shell echo $(VERSION) | awk -F. '{$$NF = $$NF + 1;} 1' | sed 's/ /./g')
RELEASE_NOTES ?= "Release $(NEXT_VERSION)"
BINARY_NAME=flashcards
MAIN_PACKAGE=./cmd/flashcards

build:
	go build -o $(BINARY_NAME) $(MAIN_PACKAGE)

build-windows:
	GOOS=windows GOARCH=amd64 go build -o $(BINARY_NAME).exe $(MAIN_PACKAGE)

build-mac:
	GOOS=darwin GOARCH=amd64 go build -o $(BINARY_NAME) $(MAIN_PACKAGE)

build-linux:
	GOOS=linux GOARCH=amd64 go build -o $(BINARY_NAME)_linux $(MAIN_PACKAGE)

build-all: build-windows build-mac build-linux

test:
	go test ./...

clean:
	rm -f $(BINARY_NAME) $(BINARY_NAME).exe $(BINARY_NAME)_linux

# Example usage: make tag VERSION=0.3.2 MESSAGE="Fix some bug"
tag:
	git tag -a v$(VERSION) -m "Version $(VERSION): $(MESSAGE)"

push-tag:
	git push origin v$(VERSION)

# Example usage: make release-github VERSION=0.3.2
release-github: build-all
	gh release create v$(VERSION) --title "Version $(VERSION)" --notes "$(RELEASE_NOTES)" $(BINARY_NAME) $(BINARY_NAME).exe $(BINARY_NAME)_linux

# Example for full release process: 
# make release VERSION=0.3.2 MESSAGE="Fix some bug" RELEASE_NOTES="## Bug Fixes\n\n- Fixed something\n\n$(shell cat RELEASE_NOTES.md)"
release: tag push-tag release-github

# Default help command
help:
	@echo "Available commands:"
	@echo "  make build            - Build for current platform"
	@echo "  make build-all        - Build for Windows, macOS, and Linux"
	@echo "  make test             - Run tests"
	@echo "  make clean            - Remove build artifacts"
	@echo "  make tag              - Create git tag (specify VERSION and MESSAGE)"
	@echo "  make push-tag         - Push the tag to remote (specify VERSION)"
	@echo "  make release-github   - Create GitHub release with built binaries (specify VERSION and RELEASE_NOTES)"
	@echo "  make release          - Full release workflow: tag, push tag, build for all platforms, create GitHub release"
	@echo ""
	@echo "Example for full release:"
	@echo '  make release VERSION=0.3.2 MESSAGE="Fix some bug" RELEASE_NOTES="## Bug Fixes\n\n- Fixed something\n\n$$(cat RELEASE_NOTES.md)"'

# Default target
default: help 