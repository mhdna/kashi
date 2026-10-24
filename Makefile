postgres:
	docker run --name postgres -p 5433:5432 -e POSTGRES_USER=root -e POSTGRES_PASSWORD=secret -d postgres:12-alpine

createdb:
	docker exec -it postgres createdb --username=root --owner=root kashi

dropdb:
	docker exec -it postgres dropdb kashi

migrateup:
	migrate -path ./db/migrations -database "postgresql://root:secret@localhost:5433/kashi?sslmode=disable" -verbose up

# migrate up the last migration
migrateup1:
	migrate -path ./db/migrations -database "postgresql://root:secret@localhost:5433/kashi?sslmode=disable" -verbose up 1

migratedown:
	migrate -path ./db/migrations -database "postgresql://root:secret@localhost:5433/kashi?sslmode=disable" -verbose down

# migrate down the last migration
migratedown1:
	migrate -path ./db/migrations -database "postgresql://root:secret@localhost:5433/kashi?sslmode=disable" -verbose down 1

sqlc:
	sqlc generate

test:
	# go test -v -cover ./...
	richgo test -v -cover ./...

server:
	go run .

# Live-reloading server for local dev. Needs `air` on PATH:
#   go install github.com/air-verse/air@latest
dev:
	air

mock:
	mockgen -package mockdb -destination db/mock/store.go github.com/mhdna/kashi/db/sqlc Store

proto:
	rm -f pb/*.go
	rm -rf doc/swagger/*.swagger.json
	protoc --proto_path=proto --go_out=pb --go_opt=paths=source_relative \
    --go-grpc_out=pb --go-grpc_opt=paths=source_relative \
	--grpc-gateway_out=pb --grpc-gateway_opt=paths=source_relative \
	--openapiv2_out=doc/swagger --openapiv2_opt=allow_merge=true,merge_file_name=kashi \
    proto/*.proto
	statik -src=./doc/swagger/ -dest=./doc

evans:
	evans --host localhost --port 8088 --package pb -r repl

seed:
	go run ./cmd/seed/main.go

# --- cloud deploy (example.com droplet, shared with kashi-commerce) -------
# kashi lives at cloud.example.com now (example.com itself serves
# kashi-commerce). One-time server setup lives in ui/deploy/SETUP.md and
# kashi-commerce/deploy/SETUP.md. Needs an ssh alias `kashi` ->
# root@<server-ip> (in ~/.ssh/config).
SSH      ?= kashi
UI_ROOT  ?= /var/www/kashi-ui
APP_DIR  ?= /home/kashi/app
PROD_DB  ?= postgresql://kashi:kashi@localhost:5432/kashi?sslmode=disable

# Full release: API binary + migrations, then the UI.
deploy: deploy-api deploy-ui
	@echo "Deployed -> https://cloud.example.com/"

# Cross-compile the API, ship it with the current migrations, run any
# pending ones on the prod DB (idempotent), restart the service.
deploy-api:
	# -s -w drops the debug tables (57 MB -> 41 MB, ~17 MB as rsync sends it):
	# the upload is most of a deploy. Panics still name their functions.
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -trimpath -ldflags "-s -w" -o /tmp/kashi-api-linux .
	rsync -avz /tmp/kashi-api-linux $(SSH):$(APP_DIR)/api-linux
	rsync -avz --delete ./db/migrations/ $(SSH):$(APP_DIR)/migrations/
	ssh $(SSH) 'chown -R kashi:kashi $(APP_DIR)/api-linux $(APP_DIR)/migrations && \
		chmod +x $(APP_DIR)/api-linux && \
		migrate -path $(APP_DIR)/migrations -database "$(PROD_DB)" up && \
		systemctl restart kashi-api && sleep 1 && systemctl is-active kashi-api'

# Build the SPA (uses ui/.env.production -> VITE_API_URL=/api) and mirror it
# into the Caddy web root. Fast path for front-end-only changes.
# Swaps files in at the end, deletes old ones last (see ui/Makefile).
deploy-ui:
	$(MAKE) -C ui build
	rsync -avz --delete-after --delay-updates --chmod=D755,F644 ui/dist/ $(SSH):$(UI_ROOT)/

# Post the feat/fix/perf commits shipped since the last posted one to the
# home page's IT Management feed, as Alex (cmd/release_notes). Run it after
# a deploy. DRY_RUN=1 prints the SQL instead of posting it.
deploy-notes:
	@posted=$$(ssh $(SSH) "LC_ALL=C.UTF-8 psql '$(PROD_DB)' -At -c \"SELECT route FROM admin_activities WHERE method = 'RELEASE'\"") && \
	sql=$$(printf '%s\n' "$$posted" | go run ./cmd/release_notes) && \
	if [ -z "$$sql" ]; then exit 0; \
	elif [ -n "$(DRY_RUN)" ]; then printf '%s\n' "$$sql"; \
	else printf '%s\n' "$$sql" | ssh $(SSH) "LC_ALL=C.UTF-8 psql '$(PROD_DB)' -v ON_ERROR_STOP=1 -q" && echo "Posted release notes."; fi

# Ship the WhatsApp bridge (../bnc/whatsapp-bridge), point its chat log at
# kashi's Inbox (CHAT_LOG_URL, added once) and restart it. Run it after
# deploy-api: the bridge reports to an endpoint that has to exist. The restart
# briefly disconnects the numbers, so pick a quiet time.
BRIDGE_SRC ?= ../bnc/whatsapp-bridge
BRIDGE_DIR ?= /home/wabridge/app
deploy-whatsapp:
	rsync -avz $(BRIDGE_SRC)/server.mjs $(BRIDGE_SRC)/report.mjs $(BRIDGE_SRC)/package.json $(BRIDGE_SRC)/package-lock.json $(SSH):$(BRIDGE_DIR)/
	ssh $(SSH) 'cd $(BRIDGE_DIR) && chown wabridge:wabridge server.mjs report.mjs package.json package-lock.json && \
		(grep -q "^CHAT_LOG_URL=" bridge.env || echo "CHAT_LOG_URL=http://127.0.0.1:$$(sed -n "s/^HTTP_SERVER_ADDRESS=.*://p" $(APP_DIR)/app.env)/whatsapp/bridge/chat" >> bridge.env) && \
		sudo -u wabridge env HOME=/home/wabridge npm ci --omit=dev && \
		systemctl restart whatsapp-bridge && sleep 3 && systemctl is-active whatsapp-bridge'

# Run pending migrations on prod without shipping anything.
migrate-prod:
	rsync -avz --delete ./db/migrations/ $(SSH):$(APP_DIR)/migrations/
	ssh $(SSH) 'migrate -path $(APP_DIR)/migrations -database "$(PROD_DB)" up'

.PHONY: postgres createdb dropdb migrateup migratedown migrateup1 migratedown1 sqlc test server dev mock proto evans seed deploy deploy-api deploy-ui deploy-notes deploy-whatsapp migrate-prod
