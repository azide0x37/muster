SHELL := /bin/sh
EXAMPLES := example/bt-audio-gateway example/dvd-ingester
VERSION := $(shell cat VERSION)
GO_ENV := GOCACHE=$(CURDIR)/.cache/go-build GOEXPERIMENT=nojsonv2
GO_LDFLAGS := -s -w -X github.com/azide0x37/muster/internal/cli.Version=$(VERSION)
CLI := $(CURDIR)/.cache/bin/muster
CORE_PLATFORMS := linux-amd64 linux-arm64 linux-armv7
DIST := $(CURDIR)/dist
RELEASE_REPO := azide0x37/muster

.PHONY: test package package-core test-core-packages clean clean-core list video build-cli

list:
	@printf '%s\n' $(EXAMPLES)

video:
	cd video && npm install --no-fund --no-audit && npx remotion render MusterExplainer out/muster-explainer.mp4

build-cli:
	mkdir -p "$(dir $(CLI))" "$(CURDIR)/.cache/go-build"
	$(GO_ENV) go build -buildvcs=false -mod=readonly -ldflags '$(GO_LDFLAGS)' -o "$(CLI)" ./cmd/muster

test: build-cli
	$(GO_ENV) go test -mod=readonly ./cmd/... ./internal/...
	$(GO_ENV) go vet -mod=readonly ./cmd/... ./internal/...
	MUSTER_CLI_SOURCE="$(CLI)" MUSTER_CLI_VERSION="$(VERSION)" ./tests/test_shared_muster_lifecycle.sh
	@for example in $(EXAMPLES); do \
		echo "==> $$example"; \
		MUSTER_CLI_SOURCE="$(CLI)" MUSTER_CLI_VERSION="$(VERSION)" $(MAKE) -C "$$example" test; \
	done

package: build-cli package-core
	@for example in $(EXAMPLES); do \
		echo "==> $$example"; \
		$(MAKE) -C "$$example" package; \
	done
	./tests/test_core_packages.sh
	MUSTER_CLI_SOURCE="$(CLI)" MUSTER_CLI_VERSION="$(VERSION)" ./tests/test_example_packages.sh

package-core: clean-core
	@set -eu; \
	for platform in $(CORE_PLATFORMS); do \
		case "$$platform" in \
			linux-amd64) goarch=amd64; goarm= ;; \
			linux-arm64) goarch=arm64; goarm= ;; \
			linux-armv7) goarch=arm; goarm=7 ;; \
		esac; \
		name="muster-$(VERSION)-$$platform"; \
		root="$(DIST)/$$name"; \
		mkdir -p "$$root/bin" "$$root/docs" "$(CURDIR)/.cache/go-build"; \
		CGO_ENABLED=0 GOOS=linux GOARCH="$$goarch" GOARM="$$goarm" $(GO_ENV) \
			go build -buildvcs=false -mod=readonly -trimpath -ldflags '$(GO_LDFLAGS)' -o "$$root/bin/muster" ./cmd/muster; \
		cp VERSION README.md MUSTER.md RELEASE.md SECURITY.md CHANGELOG.md "$$root/"; \
		cp docs/OBJECT_MODEL.md "$$root/docs/"; \
		COPYFILE_DISABLE=1 tar --no-xattrs -C "$(DIST)" -czf "$(DIST)/$$name.tar.gz" "$$name"; \
		if command -v sha256sum >/dev/null 2>&1; then \
			sha=$$(sha256sum "$(DIST)/$$name.tar.gz" | awk '{print $$1}'); \
		else \
			sha=$$(shasum -a 256 "$(DIST)/$$name.tar.gz" | awk '{print $$1}'); \
		fi; \
		printf '%s\n' "$$sha" > "$(DIST)/$$name.tar.gz.sha256"; \
		printf '{\n  "version":"$(VERSION)",\n  "platform":"%s",\n  "artifact_url":"https://github.com/%s/releases/download/v%s/%s.tar.gz",\n  "sha256":"%s"\n}\n' \
			"$$platform" "$(RELEASE_REPO)" "$(VERSION)" "$$name" "$$sha" > "$(DIST)/muster-$$platform.json"; \
	done

test-core-packages:
	./tests/test_core_packages.sh

clean-core:
	rm -rf "$(DIST)"

clean: clean-core
	@for example in $(EXAMPLES); do \
		$(MAKE) -C "$$example" clean; \
	done
