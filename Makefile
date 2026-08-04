default: build

build:
	go build -v -o terraform-provider-algolia .

install: build
	go install -v ./...

lint:
	golangci-lint run

generate:
	cd tools && go generate ./...

test:
	go test ./... -v $(TESTARGS) -timeout 120m

testexamples:
	./scripts/validate-examples.sh

testacc:
	TF_ACC=1 go test ./... -v $(TESTARGS) -timeout 120m

# End-to-end tests against a real Algolia application. Credentials are loaded
# from .env.e2e (gitignored) if present, otherwise from the environment.
# Create .env.e2e with ALGOLIA_APP_ID and ALGOLIA_API_KEY, then run `make e2e`.
e2e:
	@if [ -f .env.e2e ]; then set -a; . ./.env.e2e; set +a; fi; \
		TF_ACC=1 go test -tags e2e ./internal/e2e/... -v $(TESTARGS) -timeout 30m

.PHONY: build install lint generate test testexamples testacc e2e
