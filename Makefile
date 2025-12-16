BINARIES := codex-yolo codex-yolo-resume codex-update codex-update-select codex-auth
RELEASE_DIR := Release
GO ?= go

.PHONY: build clean install test tidy

build:
	mkdir -p $(RELEASE_DIR)
	for bin in $(BINARIES); do \
		$(GO) build -o $(RELEASE_DIR)/$$bin ./cli/$$bin || exit 1; \
	done

clean:
	rm -rf $(RELEASE_DIR)

install:
	$(MAKE) clean
	$(MAKE) build
	for bin in $(BINARIES); do \
		sudo install -m 0755 $(RELEASE_DIR)/$$bin /usr/bin/$$bin || exit 1; \
	done

test:
	$(GO) test ./...

tidy:
	$(GO) mod tidy
