# Kashi UI -- build + deploy the SPA to cloud.grandbz.com on the shared
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
# --delay-updates/--delete-after swap the new files in together at the end
# and only then remove the old ones, so open tabs never see an index.html
# pointing at chunks that aren't there yet (or are already gone).
deploy: build
	rsync -avz --delete-after --delay-updates --chmod=D755,F644 $(DIST)/ $(SSH):$(UI_ROOT)/
	@echo "Deployed -> https://cloud.grandbz.com/"

caddy-reload:
	ssh $(SSH) 'systemctl reload caddy'
