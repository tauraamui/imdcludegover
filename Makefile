.DEFAULT_GOAL := default

.PHONY: ci-run
ci-run: test lint

.PHONY: test
test:
	gotestsum ./...

.PHONY: test-no-cache
test-no-cache:
	gotestsum ./... -count=1

.PHONY: test-verbose
test-verbose:
	gotestsum --format standard-verbose ./...

.PHONY: coverage
coverage:
	go test -coverpkg=./... -coverprofile=profile.cov ./... && go tool cover -func profile.cov && rm profile.cov

.PHONY: install-gotestsum
install-gotestsum:
	go get github.com/gotestyourself/gotestsum

.PHONY: install-linter
install-linter:
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.42.1

.PHONY: lint
lint:
	golangci-lint run

# static check comes with VSCode's Go extension as one of the tools
# will look into adding an install step for it when I get more time
.PHONY: staticcheck
staticcheck:
	staticcheck ./...
