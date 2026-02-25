# 🗺 SD-WAN Project Architecture Pattern (Updated 2026)

## 1. System Overview
This is a **Protocol-Agnostic SD-WAN Controller**. It separates the Control Plane (Intelligence) from a Data Plane that can consist of Linux-based Gateways or Enterprise Vendor Hardware.

## 2. Component Definitions

### A. The Orchestrator (Control Plane)
- **Role:** Centralized Intent Engine.
- **New Capability:** **Multi-Vendor Driver Layer.** The Orchestrator uses Netmiko (Python) or Go-SSH to push CLI commands to vendor hubs (Cisco FlexVPN, Juniper Hub-and-Spoke).

### B. The Hub (Universal Data Plane)
- **Linux Hub:** Can run dual-stack (WireGuard + IPsec) to bridge modern and legacy spokes.
- **Vendor Hub (New):** A Cisco/Juniper/Fortinet device acting as a standard **IPsec Hub**. 
    - *Constraint:* Spokes connecting to a Vendor Hub must use IPsec (IKEv2), as vendor gear typically lacks native WireGuard support.

### C. The Spokes
- **Native Spokes:** Use WireGuard when connecting to a Linux Hub; fallback to IPsec when connecting to a Vendor Hub.
- **Legacy Spokes:** Always use IPsec.

## 3. Communication Patterns (The "Translation" Logic)
When the Orchestrator assigns a Spoke to a Hub, it performs a **Protocol Match**:
1.  **IF** (Hub == Linux) **AND** (Spoke == Linux) → **Deploy WireGuard**.
2.  **IF** (Hub == Vendor) **OR** (Spoke == Vendor) → **Deploy IPsec (IKEv2)**.

## 4. AI Agent Guidelines
- **Driver Modules:** Create a `drivers/` directory. Each vendor (Cisco, Juniper, Linux) should have a driver that translates "Intent" (e.g., `AddTunnel`) into specific CLI or API commands.
- **Abstraction:** The main Go logic should never see "Cisco" or "Linux." It should only see a `HubInterface` with a `PushConfig()` method.

## 5. Dynamic Mesh Engine (Flow-Based Intent)
To optimize latency and reduce Hub load, the Orchestrator supports **Dynamic Path Selection**:

- **Telemetry Source:** Hubs stream flow data (NetFlow/IPFIX) to the Redis cache.
- **Mesh Trigger:** The Orchestrator evaluates flows. If persistent heavy traffic is detected between two Spokes, a "Direct-Mesh" intent is generated.
- **Protocol Flexibility:**
    - **Linux-to-Linux:** Dynamic WireGuard peering.
    - **Vendor-to-Vendor:** Dynamic IPsec/VTI or Auto-Discovery VPN (ADVPN/NHRP).
- **TTL (Time to Live):** Mesh tunnels are ephemeral. If traffic drops below a threshold for X minutes, the Orchestrator tears down the shortcut to save resources.

## 6. Departmental Segmentation (Firewall Zones)
The solution enforces logical isolation through "Departments."

- **Isolation Strategy:** - **Layer 2:** VLAN tagging per department at the Spoke switchports.
    - **Layer 3:** VRF (Virtual Routing and Forwarding) on Vendor gear; Network Namespaces on Linux Hubs.
- **Zone-Based Firewall (ZBF):** - Traffic is assigned to a **Security Zone** based on its Department.
    - Policies are defined as `Zone_A -> Zone_B` relations.
- **Vendor Translation:**
    - **Cisco:** Orchestrator pushes `ip inspect` or `zone-pair` CLI commands.
    - **Linux:** Orchestrator pushes `nftables` rules using named sets for zones.

## 7. Hierarchical Multi-Tenancy (Realms & Orgs)
The solution follows a 3-tier hierarchy for Service Provider (MSP) readiness:

1. **Realm Layer:** The top-level administrative boundary. Policies defined here can be "Global" (e.g., Standard DNS).
2. **Organization Layer:** Logical tenant isolation. Routing tables are completely separate at this level.
3. **Department Layer:** Micro-segmentation within an Organization. Enforced by Zone-Based Firewalls.

### Data Sovereignty:
- All database queries MUST include a `realm_id` to prevent cross-tenant data leakage.
- AI agents must prioritize isolation; a "Finance" zone in Org A must never reach a "Finance" zone in Org B unless explicitly bridged via a Gateway.

## 8. Infrastructure Roles (Gateway vs. Anchor)
To maximize deployment flexibility, the platform distinguishes between Infrastructure-owned and Client-owned hardware:

### A. Core Gateways (Provider Tier)
- **Role:** High-availability multi-tenant ingress points.
- **Ownership:** Maintained at the **Realm** level.
- **Function:** Acts as a transit point for organizations without dedicated data center footprints.

### B. Site Anchors (Enterprise Tier)
- **Role:** Single-tenant dedicated convergence points.
- **Ownership:** Assigned to a specific **Organization**.
- **Function:** Serves as the primary tunnel termination point for a specific client’s headquarters or private cloud.

## 9. Fabric Management (The Control Fabric)
The platform utilizes a dual-channel communication strategy to ensure reliability:
- **Control Fabric:** A lightweight, high-priority path used exclusively for orchestration, telemetry, and heartbeats.
- **Service Fabric:** The primary path for end-user data, utilizing either WireGuard or IPsec based on node capability.

## 10. Cloud Fabric Extension (Multi-Cloud Ingress)
The platform extends the Service Fabric into Public Cloud environments (AWS, Azure, GCP) using a dual-modality approach:

### A. Virtualized Cloud Anchors
- **Implementation:** A containerized or VM-based Edge Node deployed directly into the Cloud VPC.
- **Protocol:** Prefers **WireGuard** for high-throughput, low-latency transit to the Core Gateway.
- **Benefit:** Full visibility and telemetry identical to an on-premise Site Anchor.

### B. Native Cloud Peering
- **Implementation:** The Orchestrator leverages Cloud APIs to provision native VPN Gateways.
- **Protocol:** **IPsec (IKEv2)**.
- **Benefit:** Zero-footprint deployment; utilizes cloud-native routing (e.g., AWS Transit Gateway).

## 11. Global Routing Intent
Cloud environments are mapped to **Organizations** and **Departments** just like physical sites. This ensures that a "Finance" department in a physical office can reach the "Finance" database in AWS, while remaining isolated from the "Guest" zone in the cloud.

## 12. Deployment Modalities

### A. Managed Service (SaaS Model)
- **Hosted Orchestrator:** A central, multi-tenant cluster managed by the Realm Provider.
- **Security:** Strict database-level row isolation using `realm_id`.
- **Connectivity:** Edge Nodes connect back to the Cloud-hosted Control Fabric via public internet (authenticated via TLS/Mutual-Auth).

### B. Self-Hosted (On-Prem Model)
- **Private Orchestrator:** Deployed as a Docker Compose stack or K8s cluster within the client's own data center.
- **Sovereignty:** All data, telemetry, and keys remain within the client's physical/logical perimeter.
- **Offline Capability:** Designed to operate in air-gapped environments with local Core Gateways.

## 13. High Availability & Persistence
Regardless of modality, the system maintains state using:
- **Primary State:** PostgreSQL 16 (Relational/Audit).
- **Transient State:** Redis 7 (Heartbeats/Flow Telemetry).
- **Secrets:** Environment-based vaulting to ensure keys are never written to disk in plain text.

## 14. Macro-Segmentation (Standalone Network Contexts)
Within a single Organization, the platform supports multiple **Standalone Network Contexts**. These act as completely isolated routing domains (VRFs).

- **Isolation Level:** Traffic between two Contexts (e.g., 'Production' and 'Development') is prohibited by default at the routing level.
- **Cross-Context Bridging:** If communication is required, it must be explicitly defined via a "Policy-Based Handover" at a Core Gateway.
- **Addressing:** Each Context can utilize overlapping IP space (RFC1918) without conflict, as they reside in separate VRFs/Namespaces.

### Visual Hierarchy:
[Realm] 
  └── [Organization]
        ├── [Context: Production]
        │     ├── {Dept: App-Servers}
        │     └── {Dept: Database}
        └── [Context: Legacy-Industrial]
              ├── {Dept: PLC-Control}
              └── {Dept: Monitoring}

## 15. Cognitive Control Plane (LLM-Driven Intent)
The platform leverages Large Language Models (LLMs) to abstract complex configuration tasks:

- **Natural Language Intent (NLI):** Administrative changes are submitted via structured prompts.
- **Validation Loop:** The LLM generates a "Proposed Change" (JSON). The Go Orchestrator validates this against safety rules (e.g., "No overlapping subnets") before execution.
- **Self-Healing:** Telemetry from Redis is fed back to the LLM. If a "Dynamic Mesh" fails to establish, the LLM analyzes the logs and suggests a protocol fallback (e.g., "Switching Spoke A to IPsec due to MTU issues").
- **Auditability:** Every LLM-generated change is stored in a `governance_logs` table with the original prompt and the resulting Git diff of the config.

## 16. Brownfield Adoption & Discovery
The platform is designed to ingest existing infrastructure ("Brownfield") without requiring a factory reset or downtime:

### A. Discovery Service
- **Logic:** Utilizing the Multi-Vendor Driver layer, the Orchestrator performs a read-only audit of legacy configurations.
- **Entity Matching:** Existing VLANs, VRFs, and Tunnels are automatically proposed as candidates for specific **Network Contexts** or **Departments**.

### B. Managed vs. Unmanaged State
- **Unmanaged:** The Orchestrator monitors the device but does not push configuration changes.
- **Partially Managed:** The Orchestrator only manages the "SD-WAN Tunnels" while leaving local branch routing to the legacy config.
- **Fully Managed:** The Orchestrator takes full "Source of Truth" ownership of the device.

### C. LLM-Assisted Translation
- The **Cognitive Control Plane** analyzes legacy CLI configs (e.g., a 2,000-line Cisco `running-config`) and translates them into the platform's standardized JSON intent.

## 17. Transactional State & Consistency Alignment
To ensure stability during Brownfield migrations and complex updates, the platform utilizes a State Machine for every Managed Node:

### The "In Transition" Phase:
- **Consistency Guard:** During a migration, if "odd zones" or legacy ACLs are detected that don't map to the 4-tier model, the node is marked `IN_TRANSITION`.
- **Atomic Rollbacks:** No configuration is "committed" until the Orchestrator validates the entire path consistency. If a Spoke is `IN_TRANSITION`, the Hub will hold both the old and new tunnel definitions to prevent a blackout.
- **Shadow Alignment:** The LLM identifies "Odd Zones" (legacy segments) and suggests temporary "Bridge Zones" to maintain connectivity while the node moves toward the final standardized state.

### State Flow:
[Discovered] -> [In Transition] -> [Validating] -> [Aligned/Standardized]

## 18. Policy Normalization & Zone Collapsing
For high-density legacy environments (hundreds of rules/dozens of zones), the platform employs a "Normalization Engine":

### A. Intent-Based Policy Mapping
- **Shadow Detection:** The Orchestrator identifies redundant, shadowed, or conflicting rules during the `IN_TRANSITION` phase.
- **Zone Mapping:** Legacy zones are mapped to **Standardized Departments**. 
  - *Example:* `Legacy_Zone_7` -> `Context: Production / Dept: IoT`.

### B. Policy "Hydration"
- Instead of pushing 500 static rules, the Orchestrator pushes **Dynamic Sets**. 
- It uses `ipset` (Linux) or `Object-Groups` (Vendor) to group IPs, keeping the actual firewall policy human-readable and high-performance.

### C. The "Safety Valve"
- For "Odd Rules" that don't fit the 4-tier model, the Orchestrator creates a **Legacy Exception Zone**. This keeps the network running while the LLM flags the rule for manual review or eventual decommissioning.

## 19. Local Compute & Inference (The GPU Worker)
To ensure data sovereignty and low-latency orchestration, the platform supports local GPU-accelerated inference:

- **Hardware Target:** NVIDIA RTX 50-Series (5070 Ti+) via CUDA.
- **Inference Engine:** Managed via a sidecar service (Ollama/vLLM).
- **Model Role:** - **Refactoring:** Collapsing legacy firewall rules.
    - **Validation:** Checking proposed configurations for logic errors.
    - **Documentation:** Automatically generating `CHANGELOG.md` for every network transition.

## 20. Cognitive Quality Assurance (Policy Verification)
To prevent "Hallucinations" in firewall refactoring, the platform enforces a Zero-Trust Logic Gate:

- **Deterministic Validation:** LLM outputs are validated against the Go-defined JSON Schema.
- **Reachability Diffing:** The Orchestrator compares the "Effective Security Posture" of legacy rules vs. LLM-optimized rules using a symbolic execution engine.
- **Human-in-the-Loop (HITL):** For high-density rule changes (>50 rules), the Orchestrator generates a "Comparison Report" and requires a manual sign-off before leaving the `IN_TRANSITION` state.

## 21. Cognitive Gap Analysis (Legacy vs. Intent)
The platform utilizes the local LLM to reconcile reality with intention during the discovery phase:

- **State Comparison:** The LLM ingests "Current State" telemetry and compares it against the "Desired State" defined in the Postgres DB.
- **Drift Detection:** It identifies configuration drift (e.g., a local admin manually added an 'Odd Zone' on a physical firewall) and flags it for remediation.
- **Remediation Scripting:** The LLM generates the specific sequence of `Go` driver calls required to transition the node from its legacy mess to the standardized architecture.
- **Explainability:** For every change, the LLM provides a "Reasoning Block," explaining why certain legacy elements were collapsed or discarded to meet the new security posture.

## 22. Digital Twin & Sandbox Replication
To eliminate the risk of "Intent-to-Reality" mismatches, the platform supports a high-fidelity virtualized simulation layer:

- **Shadow Instance Spawning:** The Orchestrator can trigger the creation of virtual replicas (using Containerlab or lightweight KVM instances) of any managed Spoke or Gateway.
- **Pre-Commit Simulation:** Before leaving the `IN_TRANSITION` state, the "Shadow Node" is loaded with the proposed configuration.
- **Verification Testing:** Synthetic traffic probes are executed within the virtual context to verify that the **Macro-Segmentation (Contexts)** and **Micro-Segmentation (Departments)** are behaving as intended.
- **Drift Simulation:** Allows the admin to "test" a legacy Brownfield config against a proposed refactor in a 100% isolated environment.

## 23. The Foundation Overlay (OAM & ZTP)
The platform utilizes a dedicated, high-resiliency **Foundation Overlay** for all administrative, orchestration, and discovery tasks.

### A. Zero-Touch Provisioning (ZTP)
- **Identity Bootstrapping:** New nodes call home via a secure ZTP URL (pre-configured or via DHCP Option 66).
- **Mutual Authentication:** The Brain verifies the node's hardware ID/TPM before pushing the initial "Nerve Center" configuration.
- **Initial Handshake:** The node is automatically assigned to its **Realm** and placed in a `PENDING_ADOPTION` state.

### B. Management Plane Isolation
- **The Nerve Center Fabric:** A permanent, encrypted control tunnel that exists independently of the Service Fabric.
- **Priority (QoS):** OAM traffic is tagged with the highest priority (DSCP CS6/CS7) to ensure the Brain never loses contact with the "Muscle" (the nodes) during high congestion.
- **Discovery Engine:** Used to "audit" legacy brownfield configurations before the **In Transition** state is triggered.