# Kashi UI -- build + deploy the SPA to cloud.example.com on the shared
# droplet, where Caddy serves it statically and reverse-proxies /api/* to
# the Go server.
#
# This is the UI-only fast path. For a full release (API + migrations + UI)
# use `make deploy` in the repo root. One-time server setup: deploy/SETUP.md.
# Needs an ssh alias `kashi` -> root@<server-ip> (in ~/.ssh/config).

SSH     ?= kashi
UI_ROOT ?= /var/www/kashi-ui
DIST    ?= dist

.PHONY: install build deploy caddy-reload

install:
	pnpm install --frozen-lockfile

build: install
	pnpm run build

# Build, then mirror dist/ into the Caddy web root. --delete drops files no
# longer in a fresh build; --chmod keeps them readable by the `caddy` user.
deploy: build
	rsync -avz --delete --chmod=D755,F644 $(DIST)/ $(SSH):$(UI_ROOT)/
	@echo "Deployed -> https://cloud.example.com/"

caddy-reload:
	ssh $(SSH) 'systemctl reload caddy'
