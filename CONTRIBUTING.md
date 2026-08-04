# Contributing to the Algolia Terraform Provider

## Requirements

- [Go](https://go.dev/doc/install) >= 1.25
- [Terraform](https://developer.hashicorp.com/terraform/install) >= 1.0

## Building

```bash
make build      # compile the provider
make install    # compile and install to $GOPATH/bin
```

## Running Tests

```bash
make test          # run unit tests (no credentials needed)
make testexamples  # validate every example against the local provider schema
make testacc       # run acceptance tests
make lint          # run golangci-lint
```

### Acceptance Tests

Acceptance tests run against a real Algolia application and require the following environment variables:

| Variable                          | Required           | Description                                                          |
| --------------------------------- | ------------------ | -------------------------------------------------------------------- |
| `ALGOLIA_APP_ID`                  | Yes                | Algolia Application ID                                               |
| `ALGOLIA_API_KEY`                 | Yes                | Algolia Admin API Key                                                |
| `ALGOLIA_ANALYTICS_REGION`        | For some resources | Region (`us` or `eu`) for Query Suggestions and Personalization      |
| `ALGOLIA_RUN_PERSONALIZATION_ACC` | No                 | Set to `1` to enable Personalization tests (daily API quota applies) |

To run a single test:

```bash
go test ./internal/services/index/ -run TestExpandTypoTolerance -v
```

## Generating Documentation

```bash
make generate   # regenerate docs via tfplugindocs
```

Documentation templates live in `templates/` and generated output goes to `docs/`.
