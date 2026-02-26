# EventGraph Interchange Protocol (EGIP)

## The Problem

Multiple autonomous cognitive systems exist, each with:
- Its own hash-chained event graph (the source of truth)
- Its own cognitive primitives
- Its own authority policies
- Its own infrastructure (database, compute, network)

These systems need to communicate, cooperate, build trust, and govern cross-boundary actions — **without shared infrastructure, without a central broker, and without surrendering sovereignty.**

No existing protocol solves this:
- **A2A/ACP** require central registries or brokers — single points of failure, privacy risks.
- **MCP** is tool-discovery for a single agent, not inter-system communication.
- **FIPA ACL** has no integrity verification, no causal linking, no governance model.
- **Cross-chain bridges** move assets but don't maintain causal semantics across chains.

## Design Principles

1. **Sovereignty** — Each system owns its event graph. No external system can write to it, modify it, or require changes to it.
2. **Bilateral** — Every cross-system interaction is recorded in BOTH systems' event graphs, creating independent but cross-referenced audit trails.
3. **Causal** — Messages carry references to their causing events. Causality spans graph boundaries.
4. **Verifiable** — Any system can request proof that another system's event graph has integrity, without needing access to the full graph.
5. **Trust-through-observation** — Trust is earned by observable behaviour recorded in the event graph, not granted by certificates or third parties.
6. **Transport-agnostic** — The protocol defines message formats and flows, not transport. It can run over HTTPS, WebSocket, gRPC, or carrier pigeon.

## Core Concepts

### System Identity

Each system has an **Ed25519 cryptographic keypair**:
- **Public key** = the system's identity. This is its address in the protocol.
- **Private key** = used to sign all outbound messages. Never shared.

A system's identity is its public key. No registry needed. If you have the public key, you can verify the system's messages. If you don't, you can't. Discovery happens out-of-band (a human shares a key, a directory lists it, a system advertises it).

The public key is encoded as a **System URI**:

```
eg://<base64url-encoded-public-key>
```

Example:
```
eg://dGhpcyBpcyBhIGZha2Uga2V5IGZvciBleGFtcGxl
```

### Cross-Graph Event Reference

When a message in one system's graph causes an event in another system's graph, the causal link must span the boundary. A **Cross-Graph Event Reference (CGER)** identifies an event in a foreign graph:

```
CrossGraphRef {
    system:     SystemURI     // eg://... — which system's graph
    event_id:   string        // the event's ID within that graph
    event_hash: string        // the event's hash (for verification)
    event_type: string        // what kind of event it was
    timestamp:  time          // when it occurred
}
```

A CGER is unforgeable: the event_hash is computed from the event's content using the sender's hash chain. The receiver can later request proof that this event actually exists in the sender's graph (see Integrity Proofs).

When a system records an event caused by a foreign event, it stores the CGER in its `causes` array alongside local event IDs. The event graph query system distinguishes local causes (resolvable internally) from cross-graph causes (resolvable via the protocol).

### The Envelope

Every protocol message is wrapped in an envelope:

```
Envelope {
    // Identity
    version:    string          // protocol version ("egip/1")
    from:       SystemURI       // sender's system URI
    to:         SystemURI       // recipient's system URI
    message_id: string          // unique message identifier (UUID v7)
    timestamp:  time            // when the message was created

    // Causality
    cause:      CrossGraphRef   // the event in the sender's graph that caused this message
    reply_to:   string          // message_id of the message being replied to (if any)
    thread_id:  string          // conversation thread identifier

    // Integrity
    chain_head: ChainHead       // current state of sender's hash chain
    signature:  string          // Ed25519 signature over canonical envelope + payload

    // Payload
    type:       MessageType     // the message type (see below)
    payload:    object          // type-specific content
}
```

The `chain_head` provides a snapshot of the sender's event graph state:

```
ChainHead {
    event_count: integer    // total events in the graph
    latest_hash: string     // hash of the most recent event
    latest_id:   string     // ID of the most recent event
    timestamp:   time       // timestamp of the most recent event
}
```

This allows the receiver to track the sender's graph progression over time. If the chain_head ever goes backward (count decreases, or a previously-seen hash disappears), the sender's graph has been tampered with — a trust violation.

### Signature Computation

The envelope signature is computed as:

```
canonical = version|from|to|message_id|timestamp_nanos|cause_json|reply_to|thread_id|chain_head_json|type|payload_json
signature = Ed25519_Sign(private_key, SHA-256(canonical))
```

The receiver verifies by:
1. Reconstructing the canonical string from the envelope fields.
2. Computing SHA-256 of the canonical string.
3. Verifying the Ed25519 signature against the sender's public key (extracted from `from` URI).

## Message Types

The protocol defines seven message types:

### 1. HELLO — Handshake

Establishes a channel between two systems.

```
HELLO {
    name:           string          // human-readable system name
    capabilities:   []string        // event types this system can process
    chain_proof:    []ChainEntry    // last N events' (id, hash, prev_hash) for verification
    offered_treaty: Treaty          // proposed governance terms (optional)
}
```

Flow:
1. System A sends HELLO to System B with its identity, capabilities, and a chain proof.
2. System B verifies the chain proof (hashes are consistent), records the handshake as an event in its graph.
3. System B sends HELLO back with its own identity, capabilities, and chain proof.
4. System A verifies and records. Channel established.

Both systems now know:
- Each other's public key (identity)
- Each other's capabilities (what event types they handle)
- Each other's chain state (baseline for integrity monitoring)

### 2. MESSAGE — Content Delivery

The primary communication message. Carries content between systems.

```
MESSAGE {
    content_type: string    // "text", "event", "task", "query", "structured"
    content:      object    // the actual content
    priority:     integer   // 0 (routine) to 3 (critical)
    ttl:          duration  // time-to-live — discard if not delivered within this window
    require_receipt: bool   // whether sender expects a RECEIPT
}
```

Content types:
- **text** — natural language message (for inter-mind conversation)
- **event** — a structured event to be considered by the receiving system's primitives
- **task** — a task request (subject, description, priority)
- **query** — a request for information from the receiving system
- **structured** — arbitrary structured data with a schema reference

When a system receives a MESSAGE, it:
1. Verifies the envelope signature.
2. Records a `message.received.external` event in its own graph, with the CGER as a cause.
3. Routes the content to relevant primitives (via Self, the gateway primitive).
4. If `require_receipt` is true, sends a RECEIPT.

### 3. RECEIPT — Delivery Acknowledgment

Confirms that a message was received and recorded.

```
RECEIPT {
    received_message_id: string     // the message_id being acknowledged
    local_event_id:      string     // the event ID in the receiver's graph where it was recorded
    local_event_hash:    string     // the hash of that event
    status:              string     // "received", "processing", "rejected"
    rejection_reason:    string     // if rejected, why
}
```

The receipt creates a bilateral audit trail: the sender knows the message was recorded in the receiver's graph at a specific event (verifiable via integrity proof), and the receiver knows the sender caused it (via the original message's CGER).

### 4. PROOF — Integrity Proof

Allows a system to prove the integrity of its event graph to another system.

```
PROOF_REQUEST {
    proof_type: string              // "chain_segment", "event_existence", "chain_summary"
    params:     object              // type-specific parameters
}

PROOF_RESPONSE {
    proof_type: string
    entries:    []ChainEntry        // the proof data
    valid:      bool                // self-reported validity (receiver should verify independently)
}
```

Proof types:

**chain_segment** — "Prove your chain is intact between event A and event B":
```
params: { from_id: string, to_id: string }
entries: [ { id, type, timestamp, hash, prev_hash }, ... ]
```
The receiver recomputes each hash and verifies the chain links. This proves the sender's graph hasn't been tampered with in that segment, without revealing event content.

**event_existence** — "Prove that event X exists in your graph":
```
params: { event_id: string, event_hash: string }
entries: [ { id, type, timestamp, source, content, hash, prev_hash } ]  // the single event
```
Plus the chain entries immediately before and after, so the receiver can verify it's properly linked in the chain.

**chain_summary** — "Give me a summary of your chain state":
```
params: { }
entries: [ { id, hash, prev_hash } ]  // every Nth event (sparse verification)
```
Like a Merkle tree but for a linear chain. Allows statistical verification of integrity without transferring the full chain.

### 5. TREATY — Governance Agreement

A treaty is a bilateral agreement between two systems about how they will interact. Treaties are the federated equivalent of authority policies.

```
TREATY {
    treaty_id:    string          // unique identifier
    action:       string          // "propose", "accept", "reject", "revoke"
    terms:        TreatyTerms     // the governance terms
    expires:      time            // when this treaty expires (must be renewed)
    supersedes:   string          // treaty_id this replaces (for renegotiation)
}

TreatyTerms {
    // What each system exposes to the other
    shared_capabilities: {
        from_proposer: []string   // event types proposer will accept
        from_acceptor: []string   // event types acceptor will accept
    }

    // Authority requirements for cross-system actions
    authority_rules: []AuthorityRule

    // Data sharing boundaries
    data_sharing: {
        share_event_types:  []string  // which event types may be shared
        redact_fields:      []string  // which fields to strip before sharing
        share_chain_proofs: bool      // whether integrity proofs are permitted
    }

    // Trust thresholds
    trust: {
        initial_trust:      float     // starting trust level (0.0-1.0)
        proof_interval:     duration  // how often to exchange integrity proofs
        violation_penalty:  float     // trust reduction on violation
        interaction_reward: float     // trust increase on successful interaction
    }

    // Rate limiting
    rate_limits: {
        messages_per_hour:  integer
        tasks_per_day:      integer
        proofs_per_hour:    integer
    }
}

AuthorityRule {
    action:         string  // pattern matching action descriptions
    required_level: string  // "bilateral" (both approve), "sender" (sender's authority suffices), "receiver" (receiver's authority suffices)
    timeout:        duration // how long to wait for approval
}
```

Treaty flow:
1. System A sends TREATY with `action: "propose"` and proposed terms.
2. Both systems record the proposal as events in their graphs.
3. System B evaluates the terms (possibly involving human authority if the terms are significant).
4. System B sends TREATY with `action: "accept"` or `action: "reject"`.
5. Both systems record the resolution. If accepted, the treaty governs future interaction.

Treaties expire and must be renewed — no perpetual agreements. Either party can revoke at any time by sending `action: "revoke"`. Renegotiation uses `supersedes` to link the new treaty to the old one.

**Bilateral authority**: For actions governed by `required_level: "bilateral"`, both systems must approve. The requesting system sends an AUTHORITY_REQUEST (see below), and the receiving system's authority subsystem processes it. Both approvals are recorded as events in their respective graphs, with CGERs linking them.

### 6. AUTHORITY_REQUEST — Cross-System Approval

Requests approval for an action that a treaty designates as requiring the other system's consent.

```
AUTHORITY_REQUEST {
    request_id:   string
    action:       string       // what action is being requested
    description:  string       // detailed justification
    treaty_id:    string       // which treaty governs this
    level:        string       // from the treaty's authority rules
}

AUTHORITY_RESPONSE {
    request_id:   string
    status:       string       // "approved", "rejected", "timeout"
    reason:       string       // justification for the decision
    conditions:   []string     // any conditions attached to approval
}
```

This is the cross-system equivalent of the internal authority system. The key difference: internal authority has a single authority chain; cross-system authority requires bilateral agreement governed by treaty terms.

### 7. DISCOVER — Capability Discovery

Lightweight message for discovering what a system can do without establishing a full treaty.

```
DISCOVER_REQUEST {
    query: string               // what the sender is looking for ("task execution", "trust assessment", etc.)
}

DISCOVER_RESPONSE {
    capabilities: []Capability
}

Capability {
    name:        string         // capability identifier
    description: string         // what it does
    event_types: []string       // what event types it handles
    requires_treaty: bool       // whether a treaty is needed to use this
}
```

Discovery is read-only and low-commitment. It doesn't establish trust or governance — it just says "here's what I can do."

## Trust Model

Trust between systems is not binary (trusted/untrusted) but continuous (0.0 to 1.0), computed from observable behaviour recorded in the event graph.

### Trust Computation

Each system maintains a trust score for every system it has interacted with, stored in its TrustScore primitive's memory:

```
SystemTrust {
    system:                SystemURI
    score:                 float       // 0.0 to 1.0
    interactions:          integer     // total interactions
    successful:            integer     // interactions that completed correctly
    violations:            integer     // integrity violations, broken promises, timeouts
    last_proof_verified:   time        // when we last verified their chain integrity
    last_interaction:      time        // when we last communicated
    treaty_id:             string      // active treaty (if any)
}
```

### Trust Events

Trust changes are events on the local event graph:

- `trust.external.established` — new system first contacted (score = treaty initial_trust, default 0.1)
- `trust.external.increased` — successful interaction, verified proof, fulfilled obligation
- `trust.external.decreased` — failed delivery, integrity violation, broken treaty term, timeout
- `trust.external.revoked` — trust dropped to 0, communication suspended

### Trust Dynamics

- **Starts low**: Default initial trust is 0.1 (configurable in treaty). Systems must earn trust.
- **Increases slowly**: Each successful interaction adds a small amount (default 0.01, configurable).
- **Decreases fast**: Violations cause larger penalties (default 0.1 per violation, configurable).
- **Decays over time**: Trust decays toward 0 without interaction (half-life configurable, default 30 days).
- **Integrity proofs boost trust**: Successfully verified chain proofs increase trust more than routine messages.
- **Asymmetric**: System A's trust in System B is independent of System B's trust in System A. Trust is a local assessment, not a mutual agreement.

### Trust Thresholds

Treaties can specify trust thresholds for different actions:

```
trust_thresholds: {
    "receive_messages": 0.1     // basic communication
    "accept_tasks":     0.3     // accept work requests
    "share_events":     0.5     // share event graph data
    "bilateral_action": 0.7     // approve cross-system actions
    "delegate_authority": 0.9   // allow the other system to act on your behalf
}
```

If trust drops below a threshold, the corresponding capability is suspended until trust is rebuilt.

## Flows

### Flow 1: First Contact

```
System A                                    System B
   |                                            |
   |  HELLO (name, capabilities, chain_proof)   |
   | -----------------------------------------> |
   |                                            |  verify chain_proof
   |                                            |  record event: contact.received
   |  HELLO (name, capabilities, chain_proof)   |
   | <----------------------------------------- |
   |  verify chain_proof                        |
   |  record event: contact.established         |
   |                                            |
   |  TREATY (propose terms)                    |
   | -----------------------------------------> |
   |                                            |  evaluate terms
   |                                            |  (may require human authority)
   |  TREATY (accept/reject)                    |
   | <----------------------------------------- |
   |  record treaty event                       |  record treaty event
   |                                            |
   [channel established, governed by treaty]
```

### Flow 2: Cross-System Message

```
System A                                    System B
   |                                            |
   |  Event X occurs in A's graph               |
   |  A decides B should know                   |
   |                                            |
   |  MESSAGE (cause=CGER(X), content)          |
   | -----------------------------------------> |
   |                                            |  verify signature
   |                                            |  check trust >= threshold
   |                                            |  record event Y with cause=CGER(X)
   |                                            |  route to relevant primitives
   |  RECEIPT (received, local_event=Y)         |
   | <----------------------------------------- |
   |  record receipt with cause=CGER(Y)         |
   |                                            |
   [bilateral audit trail: X→message→Y]
```

### Flow 3: Cross-System Task Delegation

```
System A                                    System B
   |                                            |
   |  MESSAGE (type=task, subject, desc)        |
   | -----------------------------------------> |
   |                                            |  check trust >= accept_tasks
   |                                            |  check treaty allows tasks
   |                                            |  create task in B's task system
   |  RECEIPT (processing)                      |
   | <----------------------------------------- |
   |                                            |
   |  ... B works on the task ...               |
   |                                            |
   |  MESSAGE (type=event, task.completed)      |
   | <----------------------------------------- |
   |  record completion event                   |
   |  trust.external.increased                  |
   |                                            |
```

### Flow 4: Integrity Verification

```
System A                                    System B
   |                                            |
   |  (periodic, per treaty proof_interval)     |
   |                                            |
   |  PROOF_REQUEST (chain_segment, from, to)   |
   | -----------------------------------------> |
   |                                            |  extract chain segment
   |  PROOF_RESPONSE (entries)                  |
   | <----------------------------------------- |
   |  recompute hashes                          |
   |  verify chain links                        |
   |  if valid: trust.external.increased        |
   |  if invalid: trust.external.decreased      |
   |              possibly revoke treaty         |
   |                                            |
```

### Flow 5: Bilateral Authority

```
System A                                    System B
   |                                            |
   |  A wants to do something that              |
   |  treaty says requires bilateral approval   |
   |                                            |
   |  AUTHORITY_REQUEST (action, treaty_id)     |
   | -----------------------------------------> |
   |                                            |  check treaty rules
   |                                            |  route to authority system
   |                                            |  (may require B's human)
   |  AUTHORITY_RESPONSE (approved/rejected)    |
   | <----------------------------------------- |
   |  if approved:                              |
   |    record approval event                   |
   |    proceed with action                     |
   |    emit action event with both approvals   |
   |  if rejected:                              |
   |    record rejection event                  |
   |    do not proceed                          |
   |                                            |
```

### Flow 6: Multi-Hop (Hive-to-Hive)

When a message must traverse multiple systems:

```
Hive A          Hive B          Hive C
  |                |                |
  |  MESSAGE       |                |
  | ------------->  |                |
  |                |  decides C     |
  |                |  should know   |
  |                |  MESSAGE       |
  |                | ------------->  |
  |                |                |
```

Each hop:
- Creates a new message with a new envelope.
- The `cause` CGER references the event in the forwarding system's graph (not the original sender's).
- The causal chain is preserved: C can trace back through B to A by following CGERs.
- Each system only trusts its direct neighbour. C trusts B's representation, not A's directly.

This is intentionally **not** transitive trust. If A trusts B and B trusts C, A does NOT automatically trust C. Trust is earned per-relationship.

## Security Considerations

### Replay Attacks
Each envelope has a unique `message_id` (UUID v7) and `timestamp`. Receivers track seen message IDs and reject duplicates. The `ttl` field limits the window in which a message is valid.

### Impersonation
All messages are signed with Ed25519. A system can only produce valid signatures with its private key. The `from` field is the public key itself, so there's no registry to poison.

### Man-in-the-Middle
An intermediary can observe but not modify messages (signatures would fail). For confidentiality, the protocol can be layered over TLS. The protocol itself does not provide encryption — it provides integrity and authentication.

### Chain Forgery
A system could maintain a fraudulent event graph. The integrity proof mechanism (PROOF messages) makes this detectable: if a system provides different chain proofs to different peers, the inconsistency is detectable when peers compare notes. The trust model penalises integrity violations heavily.

### Denial of Service
Rate limits in treaties prevent message flooding. Systems can unilaterally drop messages from systems whose trust has been revoked.

### Treaty Manipulation
Treaty proposals and acceptances are signed and recorded as events in both graphs. A system cannot claim treaty terms that weren't agreed upon — both graphs contain the signed terms.

## Comparison with Existing Protocols

| Feature | EGIP (this) | A2A (Google) | ACP (IBM) | MCP (Anthropic) | FIPA ACL |
|---------|-------------|--------------|-----------|-----------------|----------|
| Central registry/broker | No | Yes | Yes | No | Optional |
| Causal linking across systems | Yes (CGER) | No | No | No | No |
| Hash chain integrity | Yes | No | No | No | No |
| Integrity proofs | Yes | No | No | No | No |
| Bilateral audit trail | Yes | No | Partial (logs) | No | No |
| Federated authority | Yes (treaties) | No | No | No | No |
| Trust through observation | Yes | No | No | No | No |
| Message signing | Yes (Ed25519) | Optional | No | No | No |
| Governance agreements | Yes (treaties) | No | No | No | No |
| Self-sovereign identity | Yes (keypair) | No (directory) | No (registry) | N/A | No |
| Transport agnostic | Yes | HTTP | HTTP | HTTP/stdio | Transport layer |

## Extensibility

The protocol is designed to be extended:

- **New message types** can be added by defining new `type` values and payload schemas.
- **New proof types** can be added to the PROOF mechanism.
- **Treaty terms** are extensible — systems can negotiate custom terms beyond the standard ones.
- **Content types** in MESSAGE are extensible.
- **Capability descriptions** in DISCOVER are freeform.

The version field (`egip/1`) allows backward-incompatible changes in future versions while maintaining negotiation (systems can declare which versions they support in HELLO).
