# Contributing to TwigBush

Welcome, and thank you for your interest in contributing! 🎉
This project follows the [CNCF community values](https://www.cncf.io/) of openness, inclusivity, and technical excellence. Contributions are welcome from anyone, whether you are a seasoned Go developer, a security researcher, or just exploring GNAP for the first time.

---

## Code of Conduct

This project adheres to the [CNCF Code of Conduct](https://github.com/cncf/foundation/blob/main/code-of-conduct.md).
By participating, you are expected to uphold this code. Please report unacceptable behavior to [maintainers@twigbush.org](mailto:maintainers@twigbush.org).

---

## How to Contribute

### 1. Reporting Issues

* Use [GitHub Issues](https://github.com/TwigBush/docs/issues) to report bugs, request features, or suggest improvements.
* Please include:

  * What you expected to happen
  * What actually happened
  * Steps to reproduce (if applicable)
  * Version/commit of `TwigBush` you tested

### 2. Proposing Features

* Open an issue with the label `enhancement`.
* Describe the use case and reference the relevant section of [RFC 9635 (GNAP)](https://www.rfc-editor.org/rfc/rfc9635.html).
* Maintainers will discuss design and scope before implementation.

### 3. Submitting Code

* Fork the repo and create a feature branch from `main`.
* Keep commits atomic and well-described.
* Add unit tests for new functionality.
* Run `go fmt ./...` and `go test ./...` before opening a PR.
* Open a Pull Request (PR) against `main` with a clear title and description.
* Sign your commits using [Developer Certificate of Origin (DCO)](https://developercertificate.org/).

### 4. Review Process

* At least one maintainer must approve before merging.
* Larger changes may require design discussion in a GitHub Discussion or design doc.
* Be responsive to reviewer comments; reviewers will do their best to be constructive.

---

## Development Environment

### Requirements

* Go 1.22+
* Docker & docker-compose (for local Postgres/OpenFGA testing)
* Make (optional, for convenience tasks)

### Common Tasks

```bash
# Format all code
go fmt ./...

# Run tests
go test ./...

# Start the Authorization Server
go run ./cmd/as

# Start the Resource Server example
go run ./cmd/client

# Start the grant introspection demo
go run ./cmd/demo

# Build binaries
go build -o bin/as ./cmd/as
go build -o bin/rs-demo ./cmd/demo
```

### Project Layout

* `cmd/as` – GNAP Authorization Server
* `cmd/client` – GNAP Resource Server example
* `cmd/demo` – grant introspection demo
* `internal/` – Core packages (handlers, token, storage, signing, policy)
* `pkg/` – Importable helpers for external projects

---

## Documentation

* End-user and operator docs live in [the docs repo](https://github.com/TwigBush/docs).
* TODO: example code to better location
* API examples (grant, continue, introspect) live in `/gnap-go/cmd/demo` and  `/gnap-go/cmd/client`. 
* Keep docs updated with code changes.

---

## Communication

* Discussions happen in [GitHub Discussions](https://github.com/orgs/TwigBush/discussions).
* Security issues? Please **do not** open a public issue. Report privately to [security@twigbush.org](mailto:security@twigbush.org).

---

## Recognition

All contributors are listed in [CONTRIBUTORS.md](CONTRIBUTORS.md).
We follow the [CNCF CLA/DCO model](https://github.com/cncf/cla) to ensure contributions are open and inclusive.



