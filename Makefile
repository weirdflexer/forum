.PHONY: setup db-up db-stop db-status migrate seed admin backend frontend dev build start stop test integration e2e backup
setup:
	./scripts/setup.sh
db-up:
	./scripts/db.sh up
db-stop:
	./scripts/db.sh stop
db-status:
	./scripts/db.sh status
migrate:
	./scripts/backend.sh migrate
seed:
	./scripts/backend.sh seed
admin:
	./scripts/create-staff.sh
backend:
	./scripts/backend.sh serve
frontend:
	cd frontend && npm run dev
dev:
	./scripts/dev.sh
build:
	cd frontend && npm run build
	cd backend && go build -trimpath -o bin/forum ./cmd/api
start:
	./scripts/start.sh
stop:
	./scripts/stop.sh
test:
	python3 scripts/check-contract.py
	cd backend && go test ./... && go vet ./...
	cd frontend && npm run build
integration:
	./scripts/test-integration.sh
e2e:
	./scripts/test-e2e.sh
backup:
	./scripts/backup.sh
