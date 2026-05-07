SHELL := /bin/sh
PROJECT := dvd-ingester
VERSION := $(shell cat VERSION)
DIST := dist
PACKAGE_ROOT := $(DIST)/$(PROJECT)-$(VERSION)
TARBALL := $(DIST)/$(PROJECT)-$(VERSION).tar.gz

.PHONY: test doctor package clean render-units

test:
	./tests/test_shell_syntax.sh
	./tests/test_shellcheck.sh
	./tests/test_units.sh
	./tests/test_udev_rule.sh
	./tests/test_installer_idempotent.sh
	./tests/test_readme_compliance.sh
	./tests/test_muster_yaml.sh

doctor:
	./bin/doctor.sh

render-units:
	./bin/render-units.sh

package: clean
	mkdir -p "$(PACKAGE_ROOT)"
	cp -R AGENTS.md CODEX_TASK.md MUSTER.md README.md RELEASE.md SECURITY.md VERSION muster.yaml bin etc src systemd udev tests "$(PACKAGE_ROOT)/"
	find "$(PACKAGE_ROOT)" -type f -name '*.sh' -exec chmod 0755 {} \;
	chmod 0755 "$(PACKAGE_ROOT)/src/dvd-rip-one"
	tar -C "$(DIST)" -czf "$(TARBALL)" "$(PROJECT)-$(VERSION)"
	if command -v sha256sum >/dev/null 2>&1; then sha256sum "$(TARBALL)" | awk '{print $$1}' > "$(TARBALL).sha256"; else shasum -a 256 "$(TARBALL)" | awk '{print $$1}' > "$(TARBALL).sha256"; fi
	cp bin/install.sh "$(DIST)/install.sh"
	chmod 0755 "$(DIST)/install.sh"
	printf '{\n' > "$(DIST)/manifest.json"
	printf '  "project": "%s",\n' "$(PROJECT)" >> "$(DIST)/manifest.json"
	printf '  "version": "%s",\n' "$(VERSION)" >> "$(DIST)/manifest.json"
	printf '  "artifact": "%s",\n' "$(PROJECT)-$(VERSION).tar.gz" >> "$(DIST)/manifest.json"
	printf '  "artifact_url": "https://github.com/<owner>/<repo>/releases/download/v%s/%s",\n' "$(VERSION)" "$(PROJECT)-$(VERSION).tar.gz" >> "$(DIST)/manifest.json"
	printf '  "sha256": "%s",\n' "$$(cat "$(TARBALL).sha256")" >> "$(DIST)/manifest.json"
	printf '  "installer": "install.sh"\n' >> "$(DIST)/manifest.json"
	printf '}\n' >> "$(DIST)/manifest.json"

clean:
	rm -rf "$(DIST)"
