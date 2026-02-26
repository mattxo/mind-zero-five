# Australian Provisional Patent — Filing Guide

## What You're Filing

**Title:** Autonomous Cognitive Agent System with Hash-Chained Causal Event Graph and Primitive Communication Protocol

**File:** `provisional-specification.md` in this directory

This single provisional covers four inventions (20 claims):
1. The autonomous agent system (eventgraph + primitives + tick engine + decision trees + authority) — claims 1-7
2. The intra-system communication protocol (four-event vocabulary + listen/say + gateway routing + three-layer knowledge) — claims 8-10
3. The inter-system communication protocol / EGIP (cross-graph causal references + signed envelopes + integrity proofs + treaties + trust accumulation) — claims 11-16
4. The layer ontology derivation method (gap-driven derivation + convergence validation) — claims 17-20

Protocol design document: `protocol-design.md` in this directory

You can split these into separate complete (standard) applications when you file within the 12-month priority window.

## How to File

### Step 1: IP Australia Online

Go to: https://www.ipaustralia.gov.au/patents/applying-for-a-patent/filing-a-provisional-application

Use the TM Headstart / eServices portal, or file via the Patents Form system.

### Step 2: What You Need

- **Form P1 (Request for Grant of Patent)** — select "Provisional"
- **The specification document** — upload `provisional-specification.md` as a PDF (convert first)
- **Applicant details:**
  - Name: Matthew Searles
  - ABN: 86 932 245 930 (Individual/Sole Trader, active from 03 Feb 2026)
  - Address: [your address]
  - Location: NSW 2260
  - Country: Australia
- **Inventor:** Matthew Searles (same as applicant)
- **Title of Invention:** Autonomous Cognitive Agent System with Hash-Chained Causal Event Graph and Primitive Communication Protocol

### Step 3: Fee

**$100 AUD** for online filing, **$200 AUD** by post (as of 2026 fee schedule).

### Step 4: What Happens Next

- You get a **filing date** — this is your **priority date**.
- The provisional is NOT examined. It just establishes priority.
- You have **12 months** to file a **complete (standard) application** claiming priority from this provisional.
- During those 12 months you can:
  - Refine the specification
  - Add more detail to claims
  - Split into multiple applications
  - File in other countries (PCT, US, etc.) claiming this priority date
  - Get a patent attorney to polish the claims

## Important Notes

### What a Provisional Does
- Establishes a priority date (26 Feb 2026)
- Lets you say "patent pending"
- Gives you 12 months to decide on complete filing
- Does NOT give you enforceable patent rights (that requires the complete application)

### What Makes a Good Provisional
- **Detailed enough** that a person skilled in the art could implement the invention — the spec above covers this
- **Broad enough** to support various claim formulations in the complete application
- **All inventive concepts disclosed** — you can't add new matter in the complete application

### Splitting Strategy
When filing complete applications, consider:
- **Application A:** System patent (eventgraph + primitives + tick engine + decision trees) — claims 1-7
- **Application B:** Intra-system protocol patent (communication protocol + knowledge architecture) — claims 8-10
- **Application C:** Inter-system protocol patent (EGIP — cross-graph references, integrity proofs, treaties, trust) — claims 11-16. **This is arguably the most commercially valuable patent.**
- **Application D:** Methods patent (ontology derivation + convergence validation) — claims 17-20

Or file as one if the examiner doesn't require division.

### Prior Art to Be Aware Of
The spec differentiates from:
- **Blockchain/DLT:** Our hash chain is single-writer with causal DAG overlay, not distributed consensus
- **Multi-agent systems (JADE, FIPA):** Our primitives are LLM-native with evolving decision trees, not rule-based
- **A2A (Google/Linux Foundation):** Relies on central registries; no causal linking, no integrity proofs, no federated governance
- **ACP (IBM):** Central broker model; no hash chain integrity, no bilateral audit trail
- **MCP (Anthropic):** Tool discovery for single agent, not inter-system protocol
- **Cross-chain bridges:** Move assets, not causal semantics; no trust accumulation through observation
- **Pub/sub systems:** Our events are hash-chained with causal links, not fire-and-forget
- **RAFT/Paxos:** We're not doing distributed consensus — single event graph with integrity verification

### Timeline
- **Today (26 Feb 2026):** File provisional
- **By 26 Feb 2027:** File complete application(s)
- **Examination:** ~12-24 months after complete filing
- **Grant:** If claims allowed, patent granted ~2-4 years from provisional

## Convert to PDF

Before filing, convert the specification to PDF. On the Fly machine:

```bash
# If pandoc available:
pandoc provisional-specification.md -o provisional-specification.pdf

# Or use any markdown-to-PDF tool locally
```

## Cost Estimate (Total Path)

| Stage | Cost (AUD) |
|-------|-----------|
| Provisional filing | ~$240 |
| Complete application (within 12 months) | ~$470 |
| Examination request | ~$490 |
| Acceptance fee | ~$250 |
| Grant/sealing | ~$250 |
| **Total (no attorney, single application)** | **~$1,700** |

Add ~$3,000-8,000 if you engage an attorney for the complete application (recommended for claims drafting).

PCT international filing (if you want protection outside Australia) adds ~$2,500-5,000.
