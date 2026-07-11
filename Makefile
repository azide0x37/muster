SHELL := /bin/sh
EXAMPLES := example/bt-audio-gateway example/dvd-ingester

.PHONY: test package clean list video

list:
	@printf '%s\n' $(EXAMPLES)

video:
	cd video && npm install --no-fund --no-audit && npx remotion render MusterExplainer out/muster-explainer.mp4

test:
	@for example in $(EXAMPLES); do \
		echo "==> $$example"; \
		$(MAKE) -C "$$example" test; \
	done

package:
	@for example in $(EXAMPLES); do \
		echo "==> $$example"; \
		$(MAKE) -C "$$example" package; \
	done

clean:
	@for example in $(EXAMPLES); do \
		echo "==> $$example"; \
		$(MAKE) -C "$$example" clean; \
	done
