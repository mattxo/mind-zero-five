# Layer 6 — Information

## Status: Derived from Layer 5 gaps

The map calls this "Information." The territory matches — this is the emergence of information as a domain with its own properties, separable from physical carriers and the actors who create it.

## Derivation

Layer 5 creates tools, methods, knowledge, and infrastructure — all operating on physical things. But information has been implicit since Layer 0 (Events carry content) and Layer 1 (Signals carry messages) without ever being addressed as a domain in its own right. Four gaps force the transition:

1. **No information as substance** — information's properties are unaddressed
2. **No symbolic systems** — no persistent physical representation of meaning
3. **No communication infrastructure** — no concept of channels with inherent constraints
4. **No computation** — no symbol manipulation distinct from physical automation

### The key transition: physical to symbolic
Content is separable from carrier. The same message can be conveyed by speech, writing, smoke, or light. This separation — implicit since Layer 1 — becomes explicit and foundational.

## Primitives (12, in 3 groups)

### Group A — Representation (how meaning takes physical form)

| Primitive | Definition | Derivation |
|-----------|-----------|------------|
| **Symbol** | A physical entity (mark, sound, gesture) that represents something else by convention. The bridge between conceptual meaning (Term) and physical artifact (Tool). | Term (Layer 2) is meaning without physicality. Tool (Layer 5) is physicality without semantics. Symbol unites them through arbitrary convention — the decoupling of physical form from semantic content. |
| **Language** | A system of Symbols with combinatorial rules (grammar/syntax) enabling finite elements to produce infinite expressions. | Protocol (Layer 2) structures communication. Language adds generativity — expressing novel meanings through novel combinations of existing symbols. Combinatorial infinity from finite elements is a genuinely new property. |
| **Encoding** | Rules for translating between meaning and specific symbolic representation. The same meaning can be encoded differently. | Makes explicit what Symbol implies: meaning and representation are independent. Information can be translated between forms, optimized for different Channels, preserved through format changes while retaining content. |
| **Record** | Persistent externalized symbolic representation. Information that exists as a physical artifact independent of any actor's memory. | EventStore (Layer 0) stores events within the system. Record creates information artifacts outside the system — surviving the creator's death, discoverable by unknown future actors. Enables Knowledge to accumulate without limit across generations. |

### Group B — Dynamics (how information moves and endures)

| Primitive | Definition | Derivation |
|-----------|-----------|------------|
| **Channel** | A medium through which information travels, with inherent properties: capacity (how much can flow), noise (distortion), latency (delay). | Signal (Layer 1) is a one-time Act. Channel is the persistent medium with its own constraints. Different Channels enable different kinds of communication — speech (fast, ephemeral, short-range) vs. writing (slow, persistent, long-range). |
| **Copy** | Reproduction of information without consuming the original. The defining property of information vs. physical resources. | Exchange (Layer 2) is zero-sum: I lose, you gain. Copy is non-rival: we both have it. Undermines scarcity assumptions from Layers 1-2. Creates unresolved tension with Property (Layer 3) and incentive structures of Exchange. |
| **Noise** | Distortion of information during transmission or storage. A property of physical reality, not an attack or failure. | IntegrityViolation (Layer 0) is discrete and detectable. Noise is continuous, partial, often undetectable without comparison to original. Not adversarial (unlike Deception) — it's entropy acting on physical media. |
| **Redundancy** | Strategic repetition of information enabling error detection and correction. The fundamental defense against Noise. | The trade-off between efficiency (say it once) and reliability (say it enough to detect/correct errors) appears nowhere in prior layers. Emerges from information's interaction with physical reality. |

### Group C — Transformation (how information is processed)

| Primitive | Definition | Derivation |
|-----------|-----------|------------|
| **Data** | Raw symbolic representation awaiting interpretation. Pre-interpretive content that may become Evidence, Knowledge, or actionable information once processed. | Events (Layer 0) are things that happened. Knowledge (Layer 5) is verified understanding. Data is neither — it's uninterpreted symbolic content. The distinction between raw content and interpreted meaning is new. |
| **Computation** | Manipulation of symbols according to defined rules, producing new symbolic configurations. Operates on symbols, not matter. | Automation (Layer 5) transforms matter (grain → flour). Computation transforms symbols (premises → conclusions). Different substrate, different domain. Computation can process information about anything — substrate-independent. |
| **Algorithm** | A defined, finite procedure that solves a class of problems — any valid input to correct output. | Technique (Layer 5) is a practical procedure for specific outcomes. Algorithm adds generality: one procedure, infinite inputs. Mirrors Language's combinatorial property (finite rules → infinite expression). |
| **Entropy** | The measure of information content — quantifying how much uncertainty a message resolves. | Closes a circle from Layer 0: Uncertainty was "not knowing is valid." Entropy quantifies the amount of not-knowing and how much a message reduces it. Information itself becomes measurable. Surprising messages carry more information than expected ones. |

## Internal Dependencies

```
Layer 5 (Technology)
  └─> Representation (Symbol, Language, Encoding, Record)
        └─> Dynamics (Channel, Copy, Noise, Redundancy)
              └─> Transformation (Data, Computation, Algorithm, Entropy)
```

Note: Representation must exist before Dynamics (you need symbols before you can transmit or copy them). Dynamics must exist before Transformation becomes meaningful (you need information flow before processing matters at scale). But in practice, simple Computation (counting, tallying) co-emerges with basic Symbols.

## Key Insights

### 1. Copy changes everything
Copy is the most disruptive primitive in this layer. Every economic concept from Layers 1-4 assumes scarcity: Resources are finite, Exchange is zero-sum, Property is exclusive. Copy violates all three. When information can be reproduced without cost or consumption, the economics of scarcity break down. This tension — between information's natural abundance and institutional structures built for scarcity — remains one of the deepest unresolved conflicts in human civilization.

### 2. Record creates cultural immortality
Without Records, knowledge dies with the knower. With Records, knowledge accumulates across generations. This creates a second recursive engine alongside Tool (Layer 5): Knowledge → Record → Preserved Knowledge → More Knowledge → Better Records. The combination of both engines — technological and informational — produces exponential capability growth.

### 3. Language is the first infinite generator
Prior layers have finite elements producing finite combinations. Language breaks this: finite symbols + finite rules = infinite expressions. This is the same property Algorithm has (finite steps, infinite inputs) and it's no coincidence — both are manifestations of combinatorial structure, the deepest pattern in this layer.

### 4. Noise is not adversarial
Layer 0's Deception group models adversarial corruption. Noise is different — it's the physical world's indifference to meaning. Information encoded in matter is subject to the second law of thermodynamics. Noise is not someone trying to deceive you; it's the universe not caring about your message. This requires a fundamentally different response (Redundancy) than Deception does (Quarantine).

### 5. Computation + Automation = artificial minds
Layer 5's Automation creates artifacts that perform physical tasks. Layer 6's Computation creates processes that manipulate symbols. Combine them — automated computation — and you get machines that process information autonomously. This is the technical foundation for artificial intelligence: an artifact that manipulates symbols (thinks?) without having Self or Intent. The boundary between Tool and Actor, already blurred by Automation, nearly dissolves.

## Known Gaps (pointing toward Layer 7)

1. **No meaning beyond function.** Symbols represent, Languages express, Algorithms process — but all of this is functional. What about meaning that transcends utility? The beautiful, the true, the just? These are evaluative concepts absent from the informational layer.

2. **No normative foundation.** Layer 4 has Law and Rights, but WHY certain things should be rights, WHY certain behaviors should be norms — the underlying moral reasoning — is absent. Law says "don't steal." The question "why is stealing wrong?" has no home.

3. **No concept of harm beyond breach.** Layers 2-4 handle breach of agreement, violation of norm, liability for damage. But what about harm that follows every rule? Cruelty that's technically legal? Exploitation that violates no contract? The concept of moral wrong transcending legal wrong doesn't exist.

4. **No concept of what SHOULD be done.** Technology gives capability (what you CAN do). Information gives knowledge (what you KNOW). Neither addresses what you SHOULD do. The normative question — not just what is possible or known, but what is right — has no primitive.
