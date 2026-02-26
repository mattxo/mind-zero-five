# Layer 2 — Exchange

## Status: Derived from Layer 1 gaps

The map calls this "Market." The territory is **Exchange** — the emergence of structured cooperation between agents. Markets are systems of exchanges; Exchange is more fundamental.

## Derivation

Layer 1 ends with agents that can want, act, and communicate. But two gaps remain:

1. **No Agreement.** Commitment (Layer 1) is one-sided. There is no mechanism for mutual, atomic binding — "both or neither." Two Commitments that reference each other are not an Agreement; the atomicity (simultaneous binding) is genuinely new.

2. **No Shared Meaning.** Signal (Layer 1) conveys information, but nothing guarantees sender and receiver interpret it identically. Without common ground, coordination is unreliable.

These gaps are forced by Layer 1's own structure: Commitment implies a receiver who can rely on it, but reliability requires mutual understanding and binding structure that Layer 1 doesn't provide.

## Primitives (12, in 3 groups)

### Group A — Common Ground (how actors build shared understanding)

| Primitive | Definition | Derivation |
|-----------|-----------|------------|
| **Term** | A Signal with a defined, shared meaning. A symbol both parties interpret identically. | Layer 1 has Signal but no guarantee of shared interpretation. Term makes communication reliable. |
| **Protocol** | Agreed-upon rules for how Signals are structured and interpreted. The shared framework that makes Terms meaningful. | Layer 1 has no structure for communication beyond Signal/Reception/Acknowledgment. Protocol provides the grammar. |
| **Offer** | A proposed Agreement. A Signal that says "here is what I propose we both commit to." | Layer 1 has Commitment (one-sided) but no mechanism for proposing mutual arrangements. Offer is a conditional, contingent Signal — new structure. |
| **Acceptance** | A Signal that converts an Offer into a binding Agreement. | The act of transforming a proposal into mutual obligation. No Layer 1 equivalent — Acknowledgment confirms receipt but doesn't create binding. |

### Group B — Mutual Binding (the structure of agreements)

| Primitive | Definition | Derivation |
|-----------|-----------|------------|
| **Agreement** | An atomic binding of two or more conditional Commitments. Both bind or neither does. | Cannot be composed from Layer 1's one-sided Commitments. The atomicity — simultaneous binding — is genuinely new. Requires Offer + Acceptance. |
| **Obligation** | The state of owing — an unfulfilled Commitment within an Agreement. Persists in time, is attributable, is tracked. | The residue of Agreement. Exists between promise and fulfillment. Layer 1's Commitment creates expectations; Obligation is the enforceable remainder. |
| **Fulfillment** | An Obligation is satisfied. The committed Act has been performed. | An Act (Layer 1) that satisfies an Obligation. Generates positive TrustUpdate. The satisfaction relationship — linking a specific Act to a specific Obligation — is new. |
| **Breach** | An Obligation is not satisfied within the expected time. | More specific than Layer 0's Violation. Breach is a Violation of a voluntary Commitment within an Agreement. The voluntariness — the actor chose to be bound — is what distinguishes it. Generates negative TrustUpdate. |

### Group C — Value Transfer (what flows between actors)

| Primitive | Definition | Derivation |
|-----------|-----------|------------|
| **Exchange** | Transfer of something of Value between actors, structured by Agreement. | The first economic primitive. Requires Agreement + Resource + Value. Cannot exist without mutual binding — unilateral transfer is a gift (an Act), not an Exchange. |
| **Accountability** | Responsibility for Breach, grounded in voluntary entry into Agreement. | Distinct from Layer 1's Consequence (automatic) and Layer 0's Signature (attributable). Accountability is voluntary responsibility — the actor chose to enter the Agreement that creates the obligation they breached. |
| **Debt** | A persistent imbalance of Value between actors. | When Exchange is incomplete or asymmetric, Debt exists. Related to Obligation but specifically about Value asymmetry. Creates pressure toward resolution. |
| **Reciprocity** | The expectation that value given will be value returned, across interactions. | Not a specific Agreement but a general principle emerging from repeated Exchange. The first proto-norm — governs behavior across multiple interactions rather than within a single one. Layer 1 has no cross-interaction concepts. |

## Internal Dependencies

```
Layer 1 (Agency)
  └─> Common Ground (Term, Protocol, Offer, Acceptance)
        └─> Mutual Binding (Agreement, Obligation, Fulfillment, Breach)
              └─> Value Transfer (Exchange, Accountability, Debt, Reciprocity)
```

## Key Insights

### 1. Atomicity is new
The most important concept in Layer 2 is atomicity — "both or neither." Layer 0 and Layer 1 deal with individual events and individual actors. Agreement introduces the first primitive that is irreducibly multi-agent. You cannot have half an Agreement.

### 2. Reciprocity bridges individual and collective
Reciprocity is the first primitive that exists *between* specific interactions. It's not about this exchange or that exchange — it's about the *pattern* across exchanges. This is a proto-norm, and it points upward: if Reciprocity is a norm for dyads, what are the norms for groups? That's where Layer 3 begins.

### 3. The economic emerges from the social
Exchange doesn't come from "rational self-interest" or "market forces." It comes from: Foundation (events exist) → Causality (events connect) → Identity (actors exist) → Expectations (patterns are predicted) → Trust (actors are evaluated) → Value (things matter) → Intent (futures are desired) → Act (events are produced) → Signal (information is shared) → Agreement (mutual binding) → Exchange (value transfers). The economic is the *eleventh* emergence, not the first.

## Known Gaps (pointing toward Layer 3)

1. **No Norms.** Reciprocity is a proto-norm between two actors. But what governs behavior in groups of three or more? When Actor A breaches an Agreement with Actor B, can Actor C care? Layer 2 has no concept of group-level expectations.

2. **No Reputation.** Trust (Layer 0) is between Self and a specific actor. But in groups, trust information is shared — "A told me B is unreliable." This shared trust information is Reputation, and it doesn't exist yet.

3. **No Authority.** When Breach occurs, who enforces Accountability? Layer 2 assumes it's between the parties. But in groups, enforcement may be delegated to a third party. This delegation of enforcement power has no primitive yet.

4. **No Property.** Exchange transfers Resources, but there's no concept of who "owns" a Resource before or after transfer. Ownership — the right to exclusive use — is not in Layer 2.
