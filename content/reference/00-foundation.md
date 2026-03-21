# Foundation (Layer 0)

The irreducible computational foundations. 45 primitives in 11 groups.

## Gap

None — this is the base layer. The graph itself.

## Transition

Nothing → Something. The capacity to record, verify, and query what happened.

## Primitives (45)

| Group | Primitives | Domain |
|---|---|---|
| 0 — Core | Event, EventStore, Clock, Hash, Self | The graph itself |
| 1 — Causality | CausalLink, Ancestry, Descendancy, FirstCause | Why things happen |
| 2 — Identity | ActorID, ActorRegistry, Signature, Verify | Who does things |
| 3 — Expectations | Expectation, Timeout, Violation, Severity | What should happen |
| 4 — Trust | TrustScore, TrustUpdate, Corroboration, Contradiction | Who to believe |
| 5 — Confidence | Confidence, Evidence, Revision, Uncertainty | How sure we are |
| 6 — Instrumentation | InstrumentationSpec, CoverageCheck, Gap, Blind | What we're watching |
| 7 — Query | PathQuery, SubgraphExtract, Annotate, Timeline | How to find things |
| 8 — Integrity | HashChain, ChainVerify, Witness, IntegrityViolation | Is the record true |
| 9 — Deception | Pattern, DeceptionIndicator, Suspicion, Quarantine | Is someone lying |
| 10 — Health | GraphHealth, Invariant, InvariantCheck, Bootstrap | Is the system well |

## What It Provides

Every layer above builds on these foundations:

- **Events** — immutable, hash-chained, causally linked records of what happened
- **Identity** — cryptographic actor identity with signatures and verification
- **Trust** — asymmetric, non-transitive, continuous 0.0–1.0 trust scores between actors
- **Expectations** — the ability to declare what *should* happen and detect violations
- **Integrity** — hash chains and witnesses that prove the record hasn't been tampered with
- **Query** — traversal, subgraph extraction, and timeline reconstruction

The social grammar (15 operations) provides the composition layer on top of these primitives. All higher-layer grammars compose the social grammar operations with layer-specific primitives.
