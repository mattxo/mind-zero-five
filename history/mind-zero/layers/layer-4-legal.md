# Layer 4 — Legal

## Status: Derived from Layer 3 gaps

The map calls this "Legal." The territory matches — this is the formalization of social order into explicit, enforceable, bounded systems of rules.

## Derivation

Layer 3 runs on informal norms. When Groups scale beyond the point where all members personally know each other, four things break:

1. **Norms become ambiguous** — subgroups interpret differently without explicit definition
2. **Reputation becomes unreliable** — you can't know every stranger's history
3. **Authority becomes arbitrary** — personal judgment without consistent rules
4. **Sanctions become inconsistent** — same Breach, different punishment

The solution is formalization: making the implicit explicit, the personal institutional, the arbitrary consistent.

## Primitives (12, in 3 groups)

### Group A — Codification (making rules explicit and enforceable)

| Primitive | Definition | Derivation |
|-----------|-----------|------------|
| **Law** | A Norm made explicit: written in Terms, stored persistently, backed by designated Authority with codified Sanctions. Explicit, codified, consistent, prospective. | Norm (Layer 3) is informal and implicit. Law is Norm + Term + Protocol + Authority + persistence. Formalization is the new concept. |
| **Right** | A claim an individual holds against the Group and its Authority. A limit on collective power. | Inverts all prior layers where the Group is sovereign. "Even if everyone consents, you cannot do this to me." Emerges from recognition that Authority can be abused. The principle that some things cannot be traded away by collective decision is genuinely new. |
| **Contract** | An Agreement (Layer 2) recognized and enforceable within a legal framework by third-party Authority. | Bridges personal (Agreement) and institutional (Law). Transforms "I trust you" into "the system ensures you." Validity depends on legal requirements (Capacity, genuine Consent, lawful subject). |
| **Liability** | Legal responsibility for harm, including unintentional harm and harm absent any Agreement. | Layer 2's Accountability is voluntary (you chose to agree). Liability extends to involuntary situations: negligence, failure of duty, unintended harm. Answers "who bears the cost when things go wrong without an Agreement?" |

### Group B — Process (how rules are applied fairly)

| Primitive | Definition | Derivation |
|-----------|-----------|------------|
| **Due Process** | The principle that enforcement of Law must itself follow rules. Authority must follow defined procedures before imposing Sanctions. | Enforcement without process is indistinguishable from oppression. Law applied reflexively — to the enforcers themselves. No Layer 3 equivalent: Authority can act without constraint. |
| **Adjudication** | The formal process by which Authority resolves disputes. Produces a binding Judgment. | Layer 3's Authority can resolve disputes informally. Adjudication adds structure: formal accusation, response, evidence evaluation, binding determination. The binding Judgment — carrying institutional force — is new. |
| **Remedy** | Legally prescribed response to Breach aimed at restoring the harmed party to their prior state. | Layer 3's Sanction is punitive (punish the breacher). Remedy is restorative (make the victim whole). Different purpose, different direction — Sanction looks at breacher, Remedy looks at victim. |
| **Precedent** | The principle that past Adjudications guide future ones. Similar cases, similar decisions. | Creates temporal consistency beyond what Norms provide. Each Judgment adds to an accumulating body of interpreted Law. Constrains Authority's discretion by binding future decisions to past reasoning. Introduces tension: predictability vs. flexibility. |

### Group C — Sovereign Structure (how authority is bounded and scaled)

| Primitive | Definition | Derivation |
|-----------|-----------|------------|
| **Jurisdiction** | The defined scope (territory, subject matter, persons) within which an Authority's power applies. | Layer 3's Authority is unbounded within the Group. As Groups grow and multiply, overlapping Authority creates chaos. Jurisdiction gives each Authority a defined domain. |
| **Sovereignty** | Within a Jurisdiction, the final Authority whose determinations cannot be overridden. The chain of appeal stops here. | Resolves infinite regress: "who judges the judge?" Analogous to Layer 0's FirstCause — a boundary marker. Without Sovereignty, disputes can never be finally resolved. |
| **Legitimacy** | Authority is rightful when it meets specific, verifiable conditions: proper establishment, Jurisdictional bounds, Due Process, respect for Rights. | Layer 3's Consent is collective but potentially vague. Legitimacy formalizes it with verifiable criteria. Creates the concept of illegitimate Authority — power that exists but is not rightful. Formal basis for resistance. |
| **Treaty** | An Agreement between Groups (as agents) creating shared rules for inter-Group interaction. The seed of international law. | Layer 3's Collective Act lets Groups act as agents. Treaty applies Layer 2's Agreement at the inter-Group level. Recognition that even between Sovereigns, some rules should apply. |

## Internal Dependencies

```
Layer 3 (Society)
  └─> Codification (Law, Right, Contract, Liability)
        └─> Process (Due Process, Adjudication, Remedy, Precedent)
              └─> Sovereign Structure (Jurisdiction, Sovereignty, Legitimacy, Treaty)
```

## Key Insights

### 1. Right is the most radical primitive so far
Every previous layer builds collective power: events aggregate into history, actors form groups, groups develop norms and authority. Right reverses the direction — it says the individual has claims that the collective cannot override. This is the first time the architecture protects the part against the whole.

### 2. Due Process is self-referential law
The most interesting structural feature of Layer 4 is that Law constrains its own enforcement. Due Process is Law about Law — the system's mechanism for preventing its own corruption. This reflexivity (the system monitoring itself) has precedent in Layer 0 (InvariantCheck, GraphHealth) but at Layer 4 it applies to social institutions, not just event graphs.

### 3. Precedent creates institutional memory
Norms (Layer 3) are held in living memory and drift over time. Precedent creates a different kind of memory — accumulated, interpreted, binding. Each Adjudication adds to it. This is the legal system's equivalent of EventStore, but for decisions rather than events.

### 4. Sovereignty and FirstCause are structural analogs
Both are boundary markers that terminate infinite regress. FirstCause says "here, the causal chain begins." Sovereignty says "here, the chain of appeal ends." Every layered system needs these terminators to avoid infinite recursion.

### 5. Legitimacy enables revolution
By creating formal criteria for rightful Authority, Legitimacy also creates the concept of wrongful Authority. This is the first primitive that provides a principled basis for rejecting existing power structures. Layer 3's Consent is descriptive (the group accepts or doesn't). Legitimacy is normative (the Authority is rightful or isn't, regardless of whether people currently accept it).

## Known Gaps (pointing toward Layer 5)

1. **No Mechanism.** Law says what should happen and what shouldn't. It says nothing about HOW things are done — the tools, techniques, methods by which Acts are performed, Resources transformed, and Capacity extended. The "how" of capability is entirely absent.

2. **No Knowledge System.** Writing good Law requires understanding reality — cause and effect, consequences of actions, nature of harms. But there's no systematic method for investigating reality. Evidence (Layer 0) exists as a concept, but no method for generating reliable, generalizable knowledge.

3. **No Scale for Enforcement.** Legal systems need to communicate laws, store records, process disputes, and enforce across large populations. The mechanisms for doing this at scale don't exist. The legal system assumes capabilities it cannot generate from its own primitives.

4. **No Innovation.** The legal system can adapt (new Laws, new Precedent) but it is reactive — it responds to problems. There's no concept of proactively extending capability, solving new kinds of problems, or creating new possibilities. The system can govern but cannot build.
