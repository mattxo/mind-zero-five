# Layer 3 — Society

## Status: Derived from Layer 2 gaps

The map calls this "Society." The territory matches — this is the emergence of group-level structure, norms, and collective agency.

## Derivation

Layer 2 handles pairs of agents. Four gaps force the transition to groups:

1. **No Norms** — group-level expectations beyond bilateral Agreement
2. **No Reputation** — shared trust information beyond private TrustScore
3. **No Authority** — delegated enforcement power beyond bilateral Accountability
4. **No Property** — group-recognized ownership beyond mere possession

All four require three or more actors. The dyad is exhausted. The triad is the minimal unit of social structure.

### The key transition: dyad to group

When A breaches an Agreement with B, and C — who is not party to the Agreement — learns of it and adjusts their own behavior toward A, something irreducibly new has occurred. Trust information has flowed through a network. Enforcement has scaled beyond the parties involved. The bilateral world of Layer 2 cannot express this.

## Primitives (12, in 3 groups)

### Group A — Collective Identity (how groups form and structure themselves)

| Primitive | Definition | Derivation |
|-----------|-----------|------------|
| **Group** | A bounded set of actors who recognize each other as members. Has a boundary (in/out) that affects behavior. | ActorRegistry (Layer 0) is "actors are known." Group adds boundary and shared context. Agreement (Layer 2) is between named parties; Group creates expectations for all members, including future ones. The generalization from specific to categorical is new. |
| **Membership** | The binding of an actor to a Group. Creates rights and obligations. | Not Commitment (Layer 1) — Membership can be inherited, imposed, or discovered. Not Agreement (Layer 2) — you don't negotiate your way into a family. The non-voluntary, categorical nature is new. |
| **Role** | A position within a Group carrying specific Capacities and Obligations beyond ordinary Membership. | Layer 1 has Capacity (what you can do) and Layer 2 has Obligation (what you owe). Role binds these to a position in group structure, not to an individual. The position persists even when the individual changes. |
| **Consent** | The group's collective acceptance of an arrangement (Authority, Norm, Membership). May be explicit, tacit, or inherited. | Layer 2's Acceptance is bilateral and explicit. Consent is collective and potentially implicit — the group may consent by not objecting or by inheriting from tradition. This collective, implicit quality is genuinely new. |

### Group B — Social Order (how groups maintain coherence)

| Primitive | Definition | Derivation |
|-----------|-----------|------------|
| **Norm** | A group-level behavioral Expectation. Shared, often implicit, enforceable, emergent. | Layer 0's Expectation is individual and predictive. Layer 2's Agreement is bilateral and negotiated. Norm is collective, often unspoken, and arises from patterns of Reciprocity repeated across the group. The group's immune system. |
| **Reputation** | An actor's Trust profile as known to the Group. Shared trust information. | Layer 0's TrustScore is private (my assessment of you). Reputation is public (the group's assessment). A single Breach cascades through the entire group's behavior toward the breacher. Trust scales beyond the dyad. |
| **Sanction** | A group-imposed Consequence for Breach of Norm. Involves the group, including non-parties. | Layer 2's Accountability is between parties to an Agreement. Sanction is imposed by the group. Types: reputational, exclusionary, compensatory, punitive. The enforcement mechanism for Norms. |
| **Authority** | Power to enforce Norms, resolve disputes, impose Sanctions. Assigned through Role, legitimated through Consent. | Introduces asymmetric power — one actor imposes consequences on another not through dyadic relationship but through group-sanctioned position. No Layer 2 equivalent: Layer 2's enforcement is always between the agreeing parties. |

### Group C — Collective Agency (how groups act as units)

| Primitive | Definition | Derivation |
|-----------|-----------|------------|
| **Property** | Group-recognized exclusive right of an actor to a Resource. | Reveals that ownership is social, not physical. A Resource "belongs to" an actor because the Group says so and will Sanction violators. Requires: Resource + Group + Norm + Sanction. Layer 2 has Exchange but no concept of who owns what before or after transfer. |
| **Commons** | A Resource belonging to the Group collectively, with shared access governed by Norms. | The inverse of Property — shared rights vs. exclusive rights. Introduces free-riding (benefiting without contributing), a problem with no Layer 2 equivalent because Layer 2 only handles bilateral exchange. |
| **Governance** | The process by which a Group makes collective decisions. | More than Protocol (Layer 2), which structures communication. Governance structures decision-making power — how the group admits Members, changes Norms, imposes Sanctions, allocates Resources. |
| **Collective Act** | When a Group performs an Act as a single agent. The Group becomes a new kind of Self. | Cannot be reduced to Acts of individual members. The tribe declares war, the jury delivers a verdict. Requires coordination through Governance and Roles. The birth of institutional agency — a Group with its own Identity, Commitments, and Reputation. |

## Internal Dependencies

```
Layer 2 (Exchange)
  └─> Collective Identity (Group, Membership, Role, Consent)
        └─> Social Order (Norm, Reputation, Sanction, Authority)
              └─> Collective Agency (Property, Commons, Governance, Collective Act)
```

## Key Insights

### 1. The triad is the phase transition
Dyads have relationships. Triads have structure. The moment a third party can witness, adjudicate, or enforce, the social emerges. This cannot be reduced to pairwise interactions.

### 2. Ownership is social, not physical
Property is perhaps the most revealing primitive in this layer. It demonstrates that "mine" is not a fact about the world but a claim that a Group enforces. Without the Group, there is only possession (having a Resource) and capacity to defend it. Property adds legitimacy — the group's recognition.

### 3. Groups become agents
Collective Act is the most consequential primitive. Once a Group can act as a unit, it becomes a new kind of Self — with its own event history, its own trust relationships, its own commitments. This recursive emergence (agents form groups that become agents) is the mechanism by which complexity scales.

### 4. Norms are the group's immune system
Norms emerge from Reciprocity (Layer 2) repeated across many actors. They are not designed but discovered — patterns of behavior that persist because they serve the group's coherence. When a Norm is violated, the group responds (Sanction), just as an immune system responds to intrusion.

## Known Gaps (pointing toward Layer 4)

1. **No Formal Rules.** Norms are informal, often implicit. When Groups grow large enough that not all members know each other, informal norms become insufficient. You need explicit, codified rules — Law. Law is Norm made formal: written in Terms, enforced by designated Authority, with codified Sanctions.

2. **No Rights.** Property is a group-recognized right to a Resource, but what about non-resource rights? The right to speak, the right to not be harmed, the right to leave the Group. These are claims that individuals hold *against the Group itself*. This is deeply important — it's the first potential conflict between individual and collective interest.

3. **No Due Process.** Authority can adjudicate, Sanction can punish. But there's no structured process for how disputes are raised, heard, and decided. The idea that enforcement itself must follow rules doesn't exist yet.

4. **No Inter-Group Relations.** Layer 3 deals with a single Group. When Group A's Norms conflict with Group B's Norms, what governs? When Groups interact — trade, alliance, conflict — what structures apply? This requires something above individual Groups.
