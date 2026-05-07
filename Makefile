SHELL := /bin/sh
EXAMPLES := example/bt-audio-gateway example/dvd-ingester

.PHONY: test package clean list

list:
	@printf '%s\n' $(EXAMPLES)

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
