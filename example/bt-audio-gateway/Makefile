SHELL := /bin/sh
PROJECT := bt-audio-gateway
RELEASE_REPO := azide0x37/bt-audio-gateway
VERSION := $(shell cat VERSION)
DIST := dist
PACKAGE_ROOT := $(DIST)/$(PROJECT)-$(VERSION)
TARBALL := $(DIST)/$(PROJECT)-$(VERSION).tar.gz

.PHONY: test doctor package clean render-units changelog

test:
	./tests/test_shell_syntax.sh
	./tests/test_shellcheck.sh
	./tests/test_units.sh
	./tests/test_installer_idempotent.sh
	./tests/test_legacy_upgrade.sh
	./tests/test_release_security.sh
	./tests/test_update_registration_rollback.sh
	./tests/test_readme_compliance.sh
	./tests/test_muster_yaml.sh
	./tests/test_release_docs.sh

doctor:
	./bin/doctor.sh

render-units:
	./bin/render-units.sh

changelog:
	./bin/generate-changelog.sh

package: changelog clean
	mkdir -p "$(PACKAGE_ROOT)"
	cp -R AGENTS.md CHANGELOG.md CODEX_TASK.md MUSTER.md README.md RELEASE.md SECURITY.md VERSION muster.yaml muster.lock.json bin etc src systemd tests "$(PACKAGE_ROOT)/"
	cp src/bt-audio-watch src/bt-audio-route "$(PACKAGE_ROOT)/bin/"
	find "$(PACKAGE_ROOT)" -type f -name '*.sh' -exec chmod 0755 {} \;
	chmod 0755 "$(PACKAGE_ROOT)/src/bt-audio-watch" "$(PACKAGE_ROOT)/src/bt-audio-route" "$(PACKAGE_ROOT)/bin/bt-audio-watch" "$(PACKAGE_ROOT)/bin/bt-audio-route"
	COPYFILE_DISABLE=1 tar --no-xattrs -C "$(DIST)" -czf "$(TARBALL)" "$(PROJECT)-$(VERSION)"
	if command -v sha256sum >/dev/null 2>&1; then sha256sum "$(TARBALL)" | awk '{print $$1}' > "$(TARBALL).sha256"; else shasum -a 256 "$(TARBALL)" | awk '{print $$1}' > "$(TARBALL).sha256"; fi
	{ cat bin/release-transaction.sh; printf 'MUSTER_STANDALONE=1\n'; awk 'NR > 1 && $$0 != "# shellcheck disable=SC1091" && $$0 != ". \"$$SCRIPT_DIR/release-transaction.sh\"" { print }' bin/install.sh; } > "$(DIST)/install.sh"
	chmod 0755 "$(DIST)/install.sh"
	printf '{\n' > "$(DIST)/manifest.json"
	printf '  "project": "%s",\n' "$(PROJECT)" >> "$(DIST)/manifest.json"
	printf '  "version": "%s",\n' "$(VERSION)" >> "$(DIST)/manifest.json"
	printf '  "artifact": "%s",\n' "$(PROJECT)-$(VERSION).tar.gz" >> "$(DIST)/manifest.json"
	printf '  "artifact_url": "https://github.com/%s/releases/download/v%s/%s",\n' "$(RELEASE_REPO)" "$(VERSION)" "$(PROJECT)-$(VERSION).tar.gz" >> "$(DIST)/manifest.json"
	printf '  "sha256": "%s",\n' "$$(cat "$(TARBALL).sha256")" >> "$(DIST)/manifest.json"
	printf '  "installer": "install.sh"\n' >> "$(DIST)/manifest.json"
	printf '}\n' >> "$(DIST)/manifest.json"

clean:
	rm -rf "$(DIST)"
