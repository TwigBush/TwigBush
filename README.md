# TwigBush

**TwigBush** is an open source implementation of the [Grant Negotiation and Authorization Protocol (GNAP, RFC 9635)](https://www.rfc-editor.org/rfc/rfc9635.html) written in Go.
It provides a production-ready **Authorization Server (AS)** and supporting libraries for **Resource Servers (RS)**, enabling modern, key-bound, just-in-time access control for humans and AI agents.

---

## Features

* **GNAP Authorization Server**: Implements `/grant`, `/continue`, `/introspect`, and JWKS endpoints
* **Proof-of-Possession Tokens**: Detached JWS, HTTP Message Signatures, DPoP, and mTLS support
* **Short-Lived, Key-Bound Access Tokens**: Configurable TTL, audience, and constraints
* **Policy Integration**: Adapter for [OpenFGA](https://openfga.dev/) or other policy engines (e.g., Zanzibar-style graphs)
* **Resource Server Toolkit**: Example RS demo and importable client for introspection & JWKS fetching
* **Security First**: Audit logging, key rotation, revocation, and step-up authentication flows

---

## Project Layout

```
gnap-go/
  cmd/
    as/        # GNAP Authorization Server binary
    rs-demo/   # Example Resource Server
  internal/    # Core engine (tokens, signing, policy, storage, etc.)
  pkg/         # Importable helper clients for RS and AS
```

---

## Getting Started

### Prerequisites

* Go 1.22+
* Docker (for Postgres / OpenFGA integration)

### Run the Authorization Server

```bash
git clone https://github.com/TwigBush/TwigBush.git
cd gnap-go

# download dependencies
go mod download

# run the Authorization Server
go run ./cmd/as
```

The server listens on **`:8085`** by default.

### Run the Resource Server Demo

#### TODO
```bash
go run ./cmd/rs-demo
```

This demo validates GNAP proof-of-possession tokens against the AS.

---

## Example Endpoints

* `POST /grant` – Request a new access token with resource hints
* `POST /continue` – Complete an interaction or continuation flow
* `POST /introspect` – Token introspection for Resource Servers
* `GET /.well-known/jwks.json` – Public keys for token verification

---

## Roadmap

* [ ] Full DPoP support
* [ ] Advanced RS <-> AS coordination (per RFC 9767)
* [ ] Richer OpenFGA/Zanzibar policy integration
* [ ] CLI tooling for admin & debugging
* [ ] Helm charts and container images

See [Issues](../../issues) for active work.

---

## Contributing

We welcome contributions of all kinds!

* See [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.
* Maintainers and contributors are listed in [CONTRIBUTORS.md](CONTRIBUTORS.md).
* Governance and maintainer roles are defined in [MAINTAINERS.md](MAINTAINERS.md).

---

## License

Apache License 2.0 – see [LICENSE](LICENSE) for details.


