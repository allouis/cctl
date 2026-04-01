# Start dev environment: Go backend (air) + Vite frontend (HMR)
# Go auto-rebuilds on .go changes, frontend has hot reload on :5173
dev:
    #!/usr/bin/env bash
    trap 'kill 0' EXIT
    air &
    cd web/app && npm run dev &
    wait

# Run all checks (tests + lint + build)
check: test lint build

# Run all tests
test: test-go test-frontend

# Run Go tests
test-go:
    go test ./...

# Run frontend tests
test-frontend:
    cd web/app && npm test

# Run all linters
lint: lint-go lint-frontend

# Type-check Go
lint-go:
    go vet ./...

# Type-check frontend (without emitting)
lint-frontend:
    cd web/app && npx tsc --noEmit

# Full Nix build (includes Go tests, TS compilation, hash verification)
build:
    nix build
