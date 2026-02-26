# Australian Provisional Patent — Filing Guide

## What You're Filing

**Title:** Autonomous Cognitive Agent System with Hash-Chained Causal Event Graph and Primitive Communication Protocol

**File:** `provisional-specification.md` in this directory

This single provisional covers three inventions:
1. The autonomous agent system (eventgraph + primitives + tick engine + decision trees + authority)
2. The primitive communication protocol (four-event vocabulary + listen/say + gateway routing + three-layer knowledge)
3. The layer ontology derivation method (gap-driven derivation + convergence validation)

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
  - Address: [your address]
  - Country: Australia
- **Inventor:** Matthew Searles (same as applicant)
- **Title of Invention:** Autonomous Cognitive Agent System with Hash-Chained Causal Event Graph and Primitive Communication Protocol

### Step 3: Fee

**$240 AUD** for online filing (as of 2025 fee schedule — verify current fee).

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
- **Application A:** System patent (eventgraph + primitives + tick engine + decision trees)
- **Application B:** Protocol patent (communication protocol + knowledge architecture)
- **Application C:** Methods patent (ontology derivation + convergence validation)

Or file as one if the examiner doesn't require division.

### Prior Art to Be Aware Of
The spec differentiates from:
- **Blockchain/DLT:** Our hash chain is single-writer with causal DAG overlay, not distributed consensus
- **Multi-agent systems (JADE, FIPA):** Our primitives are LLM-native with evolving decision trees, not rule-based
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
