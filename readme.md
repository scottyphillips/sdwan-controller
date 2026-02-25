# 🌐 SD-WAN Controller Core

A modern, high-performance Software-Defined WAN controller built with **Go 1.24**, designed for automated network orchestration and real-time telemetry.

## 🚀 Project Intent
The goal of this project is to build a centralized "brain" for a distributed network. It replaces manual CLI configuration with an intent-based system that manages device state, secure tunnels, and traffic steering across diverse hardware.

### Key Pillars:
* **Zero-Trust Discovery:** Automatically identify and onboard routers/switches.
* **Stateful Orchestration:** Uses PostgreSQL as the "Source of Truth" for network intent.
* **Hybrid Transport:** Supports modern high-speed tunnels and legacy vendor standards.
* **Modern Stack:** Leverages Go 1.24's "Swiss Table" maps and high-performance concurrency.

## 🏗 Hybrid VPN Architecture (The Bridge)
To ensure compatibility with traditional vendor gear (Cisco, Juniper, etc.) that does not natively support WireGuard, this project implements a **Hub-and-Spoke Gateway Bridge**:

* **Northbound (Modern Edge):** Direct **WireGuard** tunnels for Linux-based nodes and high-performance endpoints.
* **Southbound (Legacy Vendor):** A Linux-based **StrongSwan/VyOS Bridge** that translates WireGuard traffic into **IPsec (IKEv2)** for legacy vendor interoperability.
* **Orchestration:** The Go Controller manages the "Bridge" as a specialized node, pushing both WireGuard and IPsec configurations to maintain a seamless overlay.

## 🛠 Tech Stack
* **Language:** Go 1.24 (Swiss Table optimization for IP lookups)
* **Database:** PostgreSQL 16 (Relational state management)
* **Cache:** Redis 7 (Real-time telemetry and state tracking)
* **Infrastructure:** Docker Compose (Containerized development environment)

## 📈 Development Roadmap
- [x] Initial Go 1.24 environment setup
- [x] Secure Docker-based infrastructure (.env obfuscation)
- [ ] **Next:** Define Device Schema (Postgres models for Native & Bridged nodes)
- [ ] Implement Network Discovery logic (ARP/SNMP scanning)
- [ ] Build the Multi-Protocol Orchestrator (WireGuard + IPsec Bridge)