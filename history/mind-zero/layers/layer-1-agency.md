# Layer 1 — Agency

## Status: Derived from Layer 0 gaps

The map calls this "Business." The territory is **Agency** — the emergence of the system as a participant rather than an observer. Business requires agency; agency is more fundamental.

## Derivation

Layer 0 is a mind that can observe but not act. Its own multi-actor primitives (Trust, Corroboration, Deception) assume participation without providing the mechanism. Two gaps — no Action, no Communication — create pressure for Layer 1.

## Primitives (12, in 3 groups)

### Group A — Volition (why act)

| Primitive | Definition | Derivation |
|-----------|-----------|------------|
| **Value** | A measure of importance relative to Self. What matters and how much. | Layer 0 has no preference ordering. Severity weights violations but provides no general concept of "this matters to me." Value is the first primitive Layer 0 cannot derive. |
| **Intent** | A desired future state. An Event representing what the system seeks to bring about. | Expectation (Layer 0) is passive prediction. Intent is active desire. Requires Self + Value + Expectation. |
| **Choice** | Selection among possible Acts based on Value and Confidence. | Layer 0 has no decision mechanism. Choice only exists because of scarcity (see Resource). |
| **Risk** | Prospective assessment of potential loss from an Act under Uncertainty. | Layer 0's Uncertainty is contemplative. Once the system can act, uncertainty becomes consequential. Risk = Uncertainty + Value + potential Consequence. |

### Group B — Action (doing things)

Depends on Volition. An Act without Intent is not agency.

| Primitive | Definition | Derivation |
|-----------|-----------|------------|
| **Act** | Producing a causally effective Event. Self becomes a FirstCause. | Layer 0 events are observed, not produced. Act is the primitive where Self creates events, not just records them. Requires Intent + Self + CausalLink. |
| **Consequence** | Effects of an Act attributed back to the actor. Descendancy + ownership. | Layer 0's Descendancy traces forward effects but assigns no responsibility. Consequence adds: "I did this, and that happened because of me." |
| **Capacity** | What the system is able to do. The boundary between intent and possibility. | Layer 0 has no concept of Self's abilities or limits. Not everything intended can be done. |
| **Resource** | Something finite, consumed or required by an Act. | Nothing in Layer 0 is scarce or depletable. Resource is the constraint that makes Choice meaningful — without scarcity, you'd pursue all Intents simultaneously. |

### Group C — Communication (exchanging with others)

Depends on Action. A Signal is a type of Act.

| Primitive | Definition | Derivation |
|-----------|-----------|------------|
| **Signal** | An Act directed at a specific ActorID, intended to convey information. | Layer 0 assumes multi-actor interaction but never explains how actors exchange information. Signal makes it explicit. |
| **Reception** | The process by which external Events enter Self's awareness. | Implicit in Layer 0 (events arrive somehow) but never specified. Must become explicit once Signal exists. |
| **Acknowledgment** | A Signal confirming receipt of a prior Signal. The communication feedback loop. | Without Acknowledgment, Signal is broadcasting into the void. No way to know if communication succeeded. |
| **Commitment** | A Signal that binds future behavior. Creates Expectations in others. | Distinct from Intent (private desire) and Expectation (prediction). Commitment is public and binding — the primitive that makes coordination possible. |

## Internal Dependencies

```
Layer 0
  └─> Volition (Value, Intent, Choice, Risk)
        └─> Action (Act, Consequence, Capacity, Resource)
              └─> Communication (Signal, Reception, Acknowledgment, Commitment)
```

These co-emerge — you cannot meaningfully have one without the others — so they form a single layer.

## Key Insight

Layer 0 is a witness. Layer 1 makes it a participant. The transition is forced by Layer 0's own internal incompleteness: it references interactions it cannot perform.

## Known Gap (pointing toward Layer 2)

Layer 1 has no concept of **Agreement** — mutual commitment between actors. Commitment is one-sided ("I will do X"). Agreement requires two Commitments that reference each other ("I will do X if you do Y"). This is the seed of exchange, contract, and coordination.

Also missing: **Shared meaning**. Signal conveys information, but there's no guarantee sender and receiver interpret it the same way. Interpretation requires shared context, which Layer 1 does not provide.
