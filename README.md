# TwigBush

**TwigBush** is an **early-stage, experimental** implementation of the [Grant Negotiation and Authorization Protocol (GNAP, RFC 9635)](https://www.rfc-editor.org/rfc/rfc9635.html) and its [Resource Server Connections extension (RFC 9767)](https://www.rfc-editor.org/rfc/rfc9767.html).
It is written in Go and aims to provide a **cloud-native GNAP Authorization Server (AS)** and supporting libraries for **Resource Servers (RS)**.

This project is not production-ready. It is published to encourage feedback, experimentation, and contributions from the community.

---

## Join the community on Discord

Have questions, want to shape the roadmap, or need help integrating TwigBush? Join our Discord and say hello.

[![Join our Discord](https://img.shields.io/badge/Discord-Join-5865F2?logo=discord&logoColor=white)](https://discord.gg/TGUCQBerDG)

**What you’ll find**
- Announcements and release notes
- Help with setup, tokens, and agent delegation
- Architecture chat, patterns, and examples
- Show and tell from users building with TwigBush

**How to get the most out of it**
- Post your use case and stack when you join
- Include version info and minimal steps for recreating bugs
- Share ideas and vote on priorities

---

## Features

* **GNAP Authorization Server**: Manages grant lifecycle and token issuance
* **Proof-of-Possession Tokens**: mTLS, detached JWS, and HTTP message signatures
* **Short-Lived, Key-Bound Tokens**: Configurable TTL, audience, and constraints
* **Resource Server Toolkit**: RS discovery, introspection, and resource registration (per RFC 9767)
* **Policy Integration**: Adapters for [OpenFGA](https://openfga.dev/) or other policy engines (Zanzibar-style graphs)
* **Security First**: Key rotation, audit logging, revocation, and step-up authentication flows

---

## Project Layout

```
gnap-go/
  cmd/
    as/        # GNAP authorization server
    client/    # Example client integration 
    demo/      # Interactive demo server
  internal/    # Core engine: grants, tokens, signing, storage, policy
  web/         # Web client code for demo
```

---

## Getting Started

### Requirements

* Go 1.22+
* Docker (for Postgres/OpenFGA integration)

### Run the Authorization Server

```bash
git clone https://github.com/TwigBush/TwigBush.git
cd TwigBush
go mod download
go run ./cmd/as
```

The AS listens on `:8085` by default.

### Run the GNAP Playground

```bash
go run ./cmd/playground
```

The playground listens on `http://localhost:8088/playground` by default.

### Run the Resource Server Command Line Client Example

```bash
go run ./cmd/client
```

This example validates GNAP proof-of-possession tokens against the AS.

---

## Example Endpoints

* `POST /grant` – Create a new grant and access token
* `POST /continue` – Continue a grant interaction
* `POST /introspect` – RS token introspection (RFC 9767 §3.3)
* `GET /.well-known/jwks.json` – JWKS for token validation
* `GET /.well-known/gnap-as-rs` – RS-facing AS discovery (RFC 9767 §3.1)

---

## Roadmap

See [Projects for roadmap items](https://github.com/TwigBush/TwigBush/projects?query=is%3Aopen)

See [Issues](https://github.com/TwigBush/TwigBush/issues) for active work.

---

## Project Charter

### Mission

TwigBush exists to provide a **cloud-native, open source reference implementation of GNAP (RFC 9635)** and its extensions (e.g., RFC 9767 for RS connections).
The project’s goal is to make **key-bound, just-in-time access control** practical for modern workloads, including multi-cloud environments, microservices, and AI/agent-driven systems.

### Scope

TwigBush is focused on:

* A Go-based **Authorization Server (AS)** that implements GNAP grant flows
* **Resource Server (RS) libraries and examples** for GNAP validation, introspection, and registration
* **Pluggable policy adapters** (OpenFGA, Zanzibar-style graphs)
* **Developer tooling** (CLI, SDKs, container images, Helm charts)
* **Standards alignment and interoperability** with IETF GNAP work

Out of scope:

* Non-standard extensions not discussed in GNAP drafts
* Proprietary connectors or commercial integrations (to be maintained outside the core repo)

### Governance

TwigBush follows an **open governance** model:

* Decisions are made in public via GitHub issues and discussions
* Maintainers are listed in [CONTRIBUTORS.md](CONTRIBUTORS.md)
* New maintainers are nominated and approved by existing maintainers through documented consensus
* Community involvement from implementers, operators, and researchers is strongly encouraged

### Alignment with CNCF

TwigBush aligns with CNCF Sandbox goals:

* **Early-stage and experimental**: intended to validate GNAP implementations and gather feedback
* **Cloud-native focus**: written in Go, containerized, with Kubernetes-ready packaging
* **Standards-first**: directly aligned with GNAP RFCs (9635, 9767) for interoperability
* **Open collaboration**: seeking contributors across security, identity, payments, and AI/agent ecosystems

---

## Contributing

TwigBush is at a **proof-of-concept stage**. Breaking changes should be expected.
We welcome feedback, issue reports, and contributions.

* See [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines
* Maintainers and contributors are listed in [CONTRIBUTORS.md](CONTRIBUTORS.md)

---

## License

Apache License 2.0 – see [LICENSE](LICENSE) for details.
