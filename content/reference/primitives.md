# Primitives

201 cognitive primitives across 14 layers. Each primitive is a software agent for a specific domain of intelligence, implementing the `Primitive` interface.

## Structure

- **Layer 0 (Foundation):** 45 primitives in 11 groups (Group 0 has 5, Groups 1-10 have 4 each)
- **Layers 1-13:** 12 primitives each in 3 groups of 4

Total: 45 + (13 × 12) = **201 primitives**

Each layer derives from a **gap** in the layer below — something the lower layer can represent but cannot reason about. Layer N activates only when Layer N-1 is stable.

## Primitive Contract

Every primitive declares:

| Property | Type | Description |
|---|---|---|
| **ID** | `PrimitiveID` | Unique identifier (encodes name + layer) |
| **Layer** | `Layer` [0-13] | Position in ontological hierarchy |
| **Subscriptions** | `[]SubscriptionPattern` | Event type globs it listens for |
| **Emits** | Event types | What it produces (with typed content) |
| **Depends on** | Interfaces | What it requires (injected at construction) |
| **State** | Key-value | What it stores between ticks |
| **Cadence** | `Cadence` [1,∞) | Minimum ticks between invocations |
| **Mechanical/Intelligent** | Flag | Whether it starts deterministic or needs LLM |

Interface: `Process(tick Tick, events []Event, snapshot Frozen<Snapshot>) → Result<[]Mutation, StoreError>`

See `docs/interfaces.md` for all referenced types.

---

## Layer 0: Foundation

45 primitives in 11 groups. The irreducible computational foundations.

**Full specification:** [`docs/layers/00-foundation.md`](layers/00-foundation.md)

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

---

## Layer 1: Agency

**Gap from Layer 0:** Foundation can record events from actors, but cannot model the difference between a passive recorder and an active participant. An actor that merely logs is indistinguishable from one that chooses to act.

**Transition:** Observer → Participant

### Group 0 — Intention

The capacity to act with purpose.

#### Goal

Setting and tracking objectives.

| | |
|---|---|
| **Subscribes to** | `decision.*`, `authority.resolved`, `actor.*` |
| **Emits** | `goal.set` { actor: ActorID, description: string, deadline: Option\<time\>, priority: Score }, `goal.achieved` { actor: ActorID, goalID: EventID, evidence: NonEmpty\<EventID\> }, `goal.abandoned` { actor: ActorID, goalID: EventID, reason: string } |
| **Depends on** | IActorStore |
| **State** | `goals`: map[ActorID][]{ id: EventID, description: string, status: ExpectationStatus } |
| **Intelligent** | Yes — determines when goals are met |

#### Plan

Decomposing goals into steps.

| | |
|---|---|
| **Subscribes to** | `goal.set`, `goal.*` |
| **Emits** | `plan.created` { goalID: EventID, steps: NonEmpty\<PlanStep\> }, `plan.step.completed` { planID: EventID, stepIndex: int }, `plan.revised` { planID: EventID, reason: EventID } |
| **Depends on** | IIntelligence |
| **State** | `plans`: map[EventID]{ steps: []PlanStep, currentStep: int } |
| **Intelligent** | Yes |

`PlanStep { description: string, dependencies: []int, status: ExpectationStatus }`

#### Initiative

Deciding when to act without being asked.

| | |
|---|---|
| **Subscribes to** | `clock.tick`, `goal.*`, `plan.*` |
| **Emits** | `initiative.taken` { actor: ActorID, action: string, trigger: EventID, confidence: Score } |
| **Depends on** | IDecisionMaker, IAuthorityChain |
| **State** | `initiativeHistory`: []{ time: time, action: string, outcome: DecisionOutcome } |
| **Intelligent** | Yes — must judge when proactive action is appropriate |

#### Commitment

Tracking whether actors follow through.

| | |
|---|---|
| **Subscribes to** | `goal.set`, `goal.achieved`, `goal.abandoned`, `plan.step.completed` |
| **Emits** | `commitment.assessed` { actor: ActorID, reliability: Score, completionRate: Score, evidence: []EventID } |
| **Depends on** | ITrustModel |
| **State** | `records`: map[ActorID]{ committed: int, completed: int, abandoned: int } |
| **Both** | Mechanical for counting, intelligent for assessing |

### Group 1 — Attention

What to focus on.

#### Focus

Directing processing resources to high-priority events.

| | |
|---|---|
| **Subscribes to** | `*` (filters by priority) |
| **Emits** | `focus.shifted` { from: Option\<EventType\>, to: EventType, reason: string }, `focus.alert` { priority: SeverityLevel, event: EventID } |
| **Depends on** | IIntelligence |
| **State** | `currentFocus`: Option\<EventType\>, `priorityQueue`: []EventID |
| **Intelligent** | Yes |

#### Filter

Suppressing noise — deciding what NOT to process.

| | |
|---|---|
| **Subscribes to** | `*` |
| **Emits** | `filter.suppressed` { eventID: EventID, reason: string }, `filter.rule.updated` { rule: SubscriptionPattern, action: string } |
| **Depends on** | IIntelligence |
| **State** | `rules`: []{ pattern: SubscriptionPattern, action: string }, `suppressedCount`: int |
| **Both** | Mechanical for rule application, intelligent for rule creation |

#### Salience

Detecting what matters in context.

| | |
|---|---|
| **Subscribes to** | `*` |
| **Emits** | `salience.detected` { eventID: EventID, score: Score, reason: string } |
| **Depends on** | IIntelligence |
| **State** | `contextWeights`: map[EventType]Score |
| **Intelligent** | Yes |

#### Distraction

Detecting when attention is pulled away from goals.

| | |
|---|---|
| **Subscribes to** | `focus.shifted`, `goal.*` |
| **Emits** | `distraction.detected` { from: EventType, to: EventType, goalImpact: Score } |
| **Depends on** | Goal primitive |
| **State** | `distractionHistory`: []{ time: time, from: EventType, duration: duration } |
| **Both** |

### Group 2 — Autonomy

The degree of independence.

#### Permission

Requesting and tracking permissions for actions.

| | |
|---|---|
| **Subscribes to** | `authority.*`, `decision.*` |
| **Emits** | `permission.requested` { actor: ActorID, action: string, level: AuthorityLevel }, `permission.granted` { actor: ActorID, action: string, granter: ActorID }, `permission.denied` { actor: ActorID, action: string, reason: string } |
| **Depends on** | IAuthorityChain |
| **State** | `permissions`: map[ActorID][]{ action: string, granted: bool, expiry: Option\<time\> } |
| **Both** |

#### Capability

Tracking what an actor can do.

| | |
|---|---|
| **Subscribes to** | `actor.registered`, `permission.*`, `trust.*` |
| **Emits** | `capability.assessed` { actor: ActorID, capabilities: []string, limitations: []string } |
| **Depends on** | IActorStore, IAuthorityChain |
| **State** | `capabilities`: map[ActorID][]string |
| **Both** |

#### Delegation

Assigning tasks or authority to others.

| | |
|---|---|
| **Subscribes to** | `authority.*`, `edge.created` (type: Delegation) |
| **Emits** | `delegation.created` { from: ActorID, to: ActorID, scope: DomainScope, weight: Score, expiry: Option\<time\> }, `delegation.revoked` { from: ActorID, to: ActorID, scope: DomainScope, reason: EventID } |
| **Depends on** | IAuthorityChain, IActorStore |
| **State** | `delegations`: map[ActorID][]{ to: ActorID, scope: DomainScope, weight: Score } |
| **Both** |

Edges created: `AddEdge { type: EdgeType.Delegation, from: delegator, to: delegate, scope: scope }`.

#### Accountability

Who is responsible when things go wrong.

| | |
|---|---|
| **Subscribes to** | `delegation.*`, `violation.*`, `goal.abandoned` |
| **Emits** | `accountability.traced` { event: EventID, chain: NonEmpty\<ActorID\>, responsible: ActorID }, `accountability.gap` { event: EventID, noResponsibleActor: bool } |
| **Depends on** | IAuthorityChain, IActorStore |
| **State** | `chains`: map[EventID][]ActorID |
| **Intelligent** | Yes — determining responsibility in delegation chains |

---

## Layer 2: Exchange

**Gap from Layer 1:** Agency models individual actors with goals and autonomy, but cannot model what happens when two actors interact. An actor can delegate, but the notion of reciprocal exchange — give and receive, offer and accept — doesn't exist.

**Transition:** Individual → Dyad

### Group 0 — Communication

How two actors talk.

#### Message

Sending and receiving structured messages between actors.

| | |
|---|---|
| **Subscribes to** | `protocol.message.*` |
| **Emits** | `message.sent` { from: ActorID, to: ActorID, content: string, conversationID: ConversationID }, `message.received` { from: ActorID, to: ActorID, eventID: EventID } |
| **Depends on** | IActorStore |
| **State** | `conversations`: map[ConversationID][]EventID |
| **Mechanical** | Yes |

#### Acknowledgement

Confirming receipt and understanding.

| | |
|---|---|
| **Subscribes to** | `message.sent`, `message.received` |
| **Emits** | `ack.sent` { messageID: EventID, understood: bool }, `ack.missing` { messageID: EventID, elapsed: duration } |
| **Depends on** | Expectation primitive |
| **State** | `pending`: map[EventID]{ sentAt: time, acked: bool } |
| **Mechanical** | Yes |

#### Clarification

Resolving ambiguity in communication.

| | |
|---|---|
| **Subscribes to** | `message.*`, `ack.*` |
| **Emits** | `clarification.requested` { messageID: EventID, question: string }, `clarification.provided` { requestID: EventID, answer: string } |
| **Depends on** | IIntelligence |
| **State** | `pendingClarifications`: []EventID |
| **Intelligent** | Yes |

#### Context

Maintaining shared conversational context between actors.

| | |
|---|---|
| **Subscribes to** | `message.*`, `clarification.*` |
| **Emits** | `context.updated` { conversationID: ConversationID, summary: string, sharedUnderstanding: Score } |
| **Depends on** | IIntelligence |
| **State** | `contexts`: map[ConversationID]{ summary: string, participants: []ActorID } |
| **Intelligent** | Yes |

### Group 1 — Reciprocity

Give and take.

#### Offer

Proposing something to another actor.

| | |
|---|---|
| **Subscribes to** | `message.*`, `exchange.*` |
| **Emits** | `offer.made` { from: ActorID, to: ActorID, terms: string, expiresAt: Option\<time\> }, `offer.withdrawn` { offerID: EventID, reason: string } |
| **Depends on** | IAuthorityChain |
| **State** | `openOffers`: map[EventID]{ from: ActorID, to: ActorID, status: string } |
| **Both** |

#### Acceptance

Accepting or rejecting an offer.

| | |
|---|---|
| **Subscribes to** | `offer.made`, `offer.withdrawn` |
| **Emits** | `offer.accepted` { offerID: EventID, acceptor: ActorID }, `offer.rejected` { offerID: EventID, rejector: ActorID, reason: Option\<string\> } |
| **Depends on** | IDecisionMaker |
| **State** | `decisions`: map[EventID]DecisionOutcome |
| **Both** |

#### Obligation

Tracking what actors owe each other.

| | |
|---|---|
| **Subscribes to** | `offer.accepted`, `delegation.*` |
| **Emits** | `obligation.created` { debtor: ActorID, creditor: ActorID, description: string, deadline: Option\<time\> }, `obligation.fulfilled` { obligationID: EventID, evidence: EventID }, `obligation.defaulted` { obligationID: EventID } |
| **Depends on** | Expectation primitive, ITrustModel |
| **State** | `obligations`: map[EventID]{ debtor: ActorID, creditor: ActorID, status: ExpectationStatus } |
| **Both** |

#### Gratitude

Recognising when obligations are fulfilled beyond expectation.

| | |
|---|---|
| **Subscribes to** | `obligation.fulfilled`, `trust.*` |
| **Emits** | `gratitude.expressed` { from: ActorID, to: ActorID, reason: EventID, weight: Weight } |
| **Depends on** | ITrustModel |
| **State** | `history`: map[ActorID][]{ to: ActorID, weight: Weight } |
| **Intelligent** | Yes — assessing when fulfilment exceeds expectation |

Gratitude has a trust effect — expressing genuine gratitude strengthens the relationship edge.

### Group 2 — Agreement

Formalising shared understanding.

#### Negotiation

Working toward agreement through iterative proposals.

| | |
|---|---|
| **Subscribes to** | `offer.*`, `message.*` |
| **Emits** | `negotiation.round` { conversationID: ConversationID, round: int, proposal: string, counterparties: [2]ActorID }, `negotiation.concluded` { conversationID: ConversationID, outcome: DecisionOutcome, terms: Option\<string\> } |
| **Depends on** | IIntelligence, IDecisionMaker |
| **State** | `negotiations`: map[ConversationID]{ rounds: int, status: string } |
| **Intelligent** | Yes |

#### Consent

Ensuring both parties explicitly agree.

| | |
|---|---|
| **Subscribes to** | `negotiation.concluded`, `offer.accepted`, `authority.*` |
| **Emits** | `consent.given` { actor: ActorID, action: string, scope: DomainScope, evidence: EventID }, `consent.withdrawn` { actor: ActorID, previousConsent: EventID, reason: string } |
| **Depends on** | IAuthorityChain |
| **State** | `consents`: map[ActorID][]{ scope: DomainScope, eventID: EventID, active: bool } |
| **Both** |

#### Contract

Binding bilateral agreements with terms and enforcement.

| | |
|---|---|
| **Subscribes to** | `consent.given`, `negotiation.concluded` |
| **Emits** | `contract.created` { parties: [2]ActorID, terms: NonEmpty\<string\>, enforcer: Option\<ActorID\> }, `contract.breached` { contractID: EventID, breacher: ActorID, term: string, evidence: EventID } |
| **Depends on** | IAuthorityChain, Expectation |
| **State** | `contracts`: map[EventID]{ parties: [2]ActorID, terms: []string, active: bool } |
| **Both** |

#### Dispute

Handling disagreements about contracts or obligations.

| | |
|---|---|
| **Subscribes to** | `contract.breached`, `obligation.defaulted`, `contradiction.found` |
| **Emits** | `dispute.raised` { plaintiff: ActorID, defendant: ActorID, claim: string, evidence: NonEmpty\<EventID\> }, `dispute.resolved` { disputeID: EventID, outcome: DecisionOutcome, resolver: ActorID } |
| **Depends on** | IDecisionMaker, IAuthorityChain |
| **State** | `disputes`: map[EventID]{ status: string, parties: [2]ActorID } |
| **Intelligent** | Yes |

---

## Layer 3: Society

**Gap from Layer 2:** Exchange models dyadic interactions — two actors negotiating, agreeing, exchanging. But it cannot model what happens when a third actor joins. Group dynamics, voting, collective decisions, reputation across a community — these require thinking beyond pairs.

**Transition:** Dyad → Group

### Group 0 — Membership

Who belongs.

#### Group

Forming and managing groups of actors.

| | |
|---|---|
| **Subscribes to** | `actor.*`, `consent.*` |
| **Emits** | `group.created` { creator: ActorID, name: string, members: NonEmpty\<ActorID\> }, `group.member.added` { groupID: EventID, actor: ActorID, addedBy: ActorID }, `group.member.removed` { groupID: EventID, actor: ActorID, removedBy: ActorID, reason: string } |
| **Depends on** | IActorStore, IAuthorityChain |
| **State** | `groups`: map[EventID]{ name: string, members: []ActorID } |
| **Both** |

#### Role

Assigning roles within groups.

| | |
|---|---|
| **Subscribes to** | `group.*`, `delegation.*` |
| **Emits** | `role.assigned` { groupID: EventID, actor: ActorID, role: string, assignedBy: ActorID }, `role.revoked` { groupID: EventID, actor: ActorID, role: string } |
| **Depends on** | IAuthorityChain |
| **State** | `roles`: map[EventID]map[ActorID][]string |
| **Both** |

#### Reputation

How the group perceives each member.

| | |
|---|---|
| **Subscribes to** | `trust.*`, `commitment.*`, `violation.*`, `gratitude.*` |
| **Emits** | `reputation.updated` { actor: ActorID, groupID: EventID, score: Score, trend: Weight } |
| **Depends on** | ITrustModel |
| **State** | `reputations`: map[ActorID]map[EventID]Score |
| **Both** |

#### Exclusion

When someone must leave.

| | |
|---|---|
| **Subscribes to** | `reputation.*`, `violation.*`, `quarantine.*`, `dispute.*` |
| **Emits** | `exclusion.proposed` { actor: ActorID, groupID: EventID, reason: NonEmpty\<EventID\>, proposer: ActorID }, `exclusion.enacted` { actor: ActorID, groupID: EventID, evidence: NonEmpty\<EventID\> }, `exclusion.appealed` { actor: ActorID, groupID: EventID } |
| **Depends on** | IAuthorityChain, IDecisionMaker |
| **State** | `proceedings`: map[ActorID]{ groupID: EventID, status: string } |
| **Intelligent** | Yes |

### Group 1 — Collective Decision

How groups decide.

#### Vote

Structured group decision-making.

| | |
|---|---|
| **Subscribes to** | `authority.requested`, `group.*` |
| **Emits** | `vote.called` { groupID: EventID, question: string, options: NonEmpty\<string\>, deadline: time }, `vote.cast` { voteID: EventID, voter: ActorID, choice: string }, `vote.result` { voteID: EventID, outcome: string, tally: map[string]int } |
| **Depends on** | IAuthorityChain |
| **State** | `activeVotes`: map[EventID]{ question: string, ballots: map[ActorID]string } |
| **Both** |

#### Consensus

Detecting when a group naturally agrees.

| | |
|---|---|
| **Subscribes to** | `message.*`, `corroboration.*`, `vote.result` |
| **Emits** | `consensus.reached` { groupID: EventID, topic: string, confidence: Score, dissenters: []ActorID }, `consensus.broken` { groupID: EventID, topic: string, cause: EventID } |
| **Depends on** | IIntelligence |
| **State** | `consensusTopics`: map[string]{ agreement: Score, participants: []ActorID } |
| **Intelligent** | Yes |

#### Dissent

Tracking and protecting disagreement.

| | |
|---|---|
| **Subscribes to** | `vote.*`, `consensus.*`, `contradiction.found` |
| **Emits** | `dissent.registered` { actor: ActorID, topic: string, position: string, evidence: []EventID }, `dissent.suppressed` { actor: ActorID, topic: string, suppressedBy: ActorID } |
| **Depends on** | IIntelligence |
| **State** | `dissents`: map[string][]{ actor: ActorID, position: string } |
| **Intelligent** | Yes |

Dissent suppression is a serious event — it triggers trust reduction and authority review.

#### Majority

Handling the tyranny of the majority.

| | |
|---|---|
| **Subscribes to** | `vote.result`, `dissent.*`, `exclusion.*` |
| **Emits** | `majority.override.detected` { groupID: EventID, minority: []ActorID, decision: EventID, impact: SeverityLevel } |
| **Depends on** | IIntelligence |
| **State** | `overrideHistory`: []{ decision: EventID, minoritySize: int } |
| **Intelligent** | Yes |

### Group 2 — Norms

Shared expectations.

#### Convention

Unwritten rules that emerge from behaviour.

| | |
|---|---|
| **Subscribes to** | `pattern.detected`, `*` |
| **Emits** | `convention.detected` { groupID: EventID, description: string, adherence: Score, evidence: []EventID } |
| **Depends on** | IIntelligence, Pattern primitive |
| **State** | `conventions`: map[EventID][]{ description: string, adherence: Score } |
| **Intelligent** | Yes |

#### Norm

Explicit shared expectations with enforcement.

| | |
|---|---|
| **Subscribes to** | `convention.detected`, `consensus.reached` |
| **Emits** | `norm.established` { groupID: EventID, description: string, enforcement: AuthorityLevel }, `norm.violated` { normID: EventID, actor: ActorID, evidence: EventID } |
| **Depends on** | IAuthorityChain, Expectation |
| **State** | `norms`: map[EventID]{ description: string, enforcement: AuthorityLevel } |
| **Both** |

#### Sanction

Consequences for norm violations.

| | |
|---|---|
| **Subscribes to** | `norm.violated`, `violation.*` |
| **Emits** | `sanction.applied` { actor: ActorID, normID: EventID, severity: SeverityLevel, description: string }, `sanction.appealed` { sanctionID: EventID, actor: ActorID } |
| **Depends on** | IAuthorityChain, IDecisionMaker |
| **State** | `sanctions`: map[ActorID][]{ normID: EventID, severity: SeverityLevel } |
| **Both** |

#### Forgiveness

Restoring standing after violation and sanction.

| | |
|---|---|
| **Subscribes to** | `sanction.applied`, `trust.*`, `obligation.fulfilled` |
| **Emits** | `forgiveness.granted` { grantor: ActorID, actor: ActorID, violation: EventID, conditions: Option\<string\> }, `standing.restored` { actor: ActorID, groupID: EventID } |
| **Depends on** | ITrustModel, IDecisionMaker |
| **State** | `restorations`: map[ActorID][]{ violation: EventID, forgiven: bool } |
| **Intelligent** | Yes |

---

## Layer 4: Legal

**Gap from Layer 3:** Society models groups with norms and sanctions, but norms are informal — they emerge from behaviour and consensus. There's no notion of codified rules, jurisdiction, precedent, or formal adjudication.

**Transition:** Informal → Formal

### Group 0 — Codification

Writing it down.

#### Rule

Formal, codified rules with explicit conditions and consequences.

| | |
|---|---|
| **Subscribes to** | `norm.established`, `consensus.reached`, `vote.result` |
| **Emits** | `rule.enacted` { authority: ActorID, scope: DomainScope, condition: string, consequence: string }, `rule.amended` { ruleID: EventID, amendment: string, authority: ActorID }, `rule.repealed` { ruleID: EventID, authority: ActorID, reason: string } |
| **Depends on** | IAuthorityChain |
| **State** | `rules`: map[EventID]{ scope: DomainScope, active: bool } |
| **Both** |

#### Jurisdiction

Which rules apply where.

| | |
|---|---|
| **Subscribes to** | `rule.*`, `group.*` |
| **Emits** | `jurisdiction.defined` { scope: DomainScope, rules: []EventID, actors: []ActorID }, `jurisdiction.conflict` { scopes: [2]DomainScope, conflictingRules: [2]EventID } |
| **Depends on** | IAuthorityChain |
| **State** | `jurisdictions`: map[DomainScope]{ rules: []EventID } |
| **Both** |

#### Precedent

Past decisions that inform future ones.

| | |
|---|---|
| **Subscribes to** | `dispute.resolved`, `decision.*` |
| **Emits** | `precedent.set` { decision: EventID, scope: DomainScope, principle: string }, `precedent.cited` { precedentID: EventID, newCase: EventID, relevance: Score } |
| **Depends on** | IIntelligence, Store |
| **State** | `precedents`: map[DomainScope][]{ decision: EventID, principle: string } |
| **Intelligent** | Yes |

#### Interpretation

Applying rules to specific situations.

| | |
|---|---|
| **Subscribes to** | `rule.*`, `dispute.*`, `precedent.*` |
| **Emits** | `interpretation.rendered` { ruleID: EventID, situation: EventID, conclusion: string, confidence: Score, precedents: []EventID } |
| **Depends on** | IIntelligence, IDecisionMaker |
| **State** | `interpretations`: map[EventID]{ ruleID: EventID, conclusion: string } |
| **Intelligent** | Yes |

### Group 1 — Process

How justice works.

#### Adjudication

Formal dispute resolution by an authorised decision-maker.

| | |
|---|---|
| **Subscribes to** | `dispute.raised`, `rule.*`, `precedent.*` |
| **Emits** | `adjudication.begun` { disputeID: EventID, adjudicator: ActorID }, `adjudication.ruling` { disputeID: EventID, ruling: DecisionOutcome, reasoning: string, precedents: []EventID } |
| **Depends on** | IDecisionMaker, IAuthorityChain |
| **State** | `cases`: map[EventID]{ adjudicator: ActorID, status: string } |
| **Intelligent** | Yes |

#### Appeal

Challenging a ruling.

| | |
|---|---|
| **Subscribes to** | `adjudication.ruling`, `exclusion.enacted`, `sanction.applied` |
| **Emits** | `appeal.filed` { ruling: EventID, appellant: ActorID, grounds: string }, `appeal.decided` { appealID: EventID, outcome: DecisionOutcome, newRuling: Option\<string\> } |
| **Depends on** | IAuthorityChain, IDecisionMaker |
| **State** | `appeals`: map[EventID]{ status: string } |
| **Intelligent** | Yes |

#### DueProcess

Ensuring procedural fairness.

| | |
|---|---|
| **Subscribes to** | `adjudication.*`, `exclusion.*`, `sanction.*` |
| **Emits** | `due_process.satisfied` { proceeding: EventID }, `due_process.violated` { proceeding: EventID, violation: string, severity: SeverityLevel } |
| **Depends on** | IIntelligence |
| **State** | `requirements`: []{ proceeding: EventID, checklist: map[string]bool } |
| **Both** |

#### Rights

Fundamental protections that override other rules.

| | |
|---|---|
| **Subscribes to** | `rule.*`, `sanction.*`, `exclusion.*`, `due_process.*` |
| **Emits** | `right.defined` { name: string, scope: DomainScope, description: string }, `right.violated` { rightName: string, actor: ActorID, violator: ActorID, evidence: EventID, severity: SeverityLevel } |
| **Depends on** | IAuthorityChain |
| **State** | `rights`: map[string]{ scope: DomainScope, description: string } |
| **Both** |

Rights violations are among the most serious events in the system — they trigger immediate authority escalation.

### Group 2 — Compliance

Following the rules.

#### Audit

Systematic review of actions against rules.

| | |
|---|---|
| **Subscribes to** | `clock.tick`, `rule.*` |
| **Emits** | `audit.completed` { scope: DomainScope, findings: []{ rule: EventID, status: ExpectationStatus }, overall: Score } |
| **Depends on** | Store, IIntelligence |
| **State** | `lastAudit`: map[DomainScope]time |
| **Intelligent** | Yes |

#### Enforcement

Taking action when rules are broken.

| | |
|---|---|
| **Subscribes to** | `audit.*`, `rule.*`, `right.violated` |
| **Emits** | `enforcement.action` { rule: EventID, violator: ActorID, action: string, severity: SeverityLevel } |
| **Depends on** | IAuthorityChain, IDecisionMaker |
| **State** | `actions`: map[ActorID][]{ rule: EventID, action: string } |
| **Both** |

#### Amnesty

Formal forgiveness that supersedes enforcement.

| | |
|---|---|
| **Subscribes to** | `enforcement.*`, `vote.result` |
| **Emits** | `amnesty.granted` { actors: []ActorID, scope: DomainScope, authority: ActorID, reason: string } |
| **Depends on** | IAuthorityChain (requires high authority) |
| **State** | `grants`: []{ actors: []ActorID, scope: DomainScope } |
| **Both** |

#### Reform

Changing rules based on experience.

| | |
|---|---|
| **Subscribes to** | `precedent.*`, `right.violated`, `audit.*`, `dissent.*` |
| **Emits** | `reform.proposed` { ruleID: EventID, proposal: string, justification: NonEmpty\<EventID\> }, `reform.enacted` { ruleID: EventID, amendment: string } |
| **Depends on** | IIntelligence, IDecisionMaker |
| **State** | `proposals`: map[EventID]{ status: string } |
| **Intelligent** | Yes |

---

## Layer 5: Technology

**Gap from Layer 4:** Legal formalises rules and adjudication, but it governs — it doesn't build. There's no concept of creating tools, artefacts, or systems. An actor can follow rules but cannot create something new that extends the system's capabilities.

**Transition:** Governing → Building

### Group 0 — Artefact

Things that are made.

#### Create

Making new things.

| | |
|---|---|
| **Subscribes to** | `plan.*`, `goal.*` |
| **Emits** | `artefact.created` { creator: ActorID, type: string, description: string, version: int }, `artefact.version` { artefactID: EventID, version: int, changes: string } |
| **Depends on** | IActorStore |
| **State** | `artefacts`: map[EventID]{ type: string, version: int, creator: ActorID } |
| **Both** |

#### Tool

Artefacts that extend actor capabilities.

| | |
|---|---|
| **Subscribes to** | `artefact.created`, `capability.*` |
| **Emits** | `tool.registered` { artefactID: EventID, capabilities: []string, requirements: []string }, `tool.used` { toolID: EventID, user: ActorID, context: string } |
| **Depends on** | IActorStore |
| **State** | `tools`: map[EventID]{ capabilities: []string, usageCount: int } |
| **Both** |

#### Quality

Assessing how well something was made.

| | |
|---|---|
| **Subscribes to** | `artefact.*`, `tool.used` |
| **Emits** | `quality.assessed` { artefactID: EventID, score: Score, criteria: map[string]Score, assessor: ActorID } |
| **Depends on** | IIntelligence |
| **State** | `assessments`: map[EventID][]{ score: Score, assessor: ActorID } |
| **Intelligent** | Yes |

#### Deprecation

When artefacts should no longer be used.

| | |
|---|---|
| **Subscribes to** | `quality.*`, `artefact.version` |
| **Emits** | `deprecation.announced` { artefactID: EventID, reason: string, replacement: Option\<EventID\>, deadline: Option\<time\> } |
| **Depends on** | IAuthorityChain |
| **State** | `deprecated`: map[EventID]{ reason: string, replacement: Option\<EventID\> } |
| **Both** |

### Group 1 — Process

How things are made.

#### Workflow

Defining repeatable processes.

| | |
|---|---|
| **Subscribes to** | `plan.*`, `convention.detected` |
| **Emits** | `workflow.defined` { name: string, steps: NonEmpty\<string\>, owner: ActorID }, `workflow.executed` { workflowID: EventID, executor: ActorID, result: DecisionOutcome } |
| **Depends on** | IIntelligence |
| **State** | `workflows`: map[EventID]{ name: string, steps: []string, executions: int } |
| **Both** |

#### Automation

Converting manual workflows to mechanical ones.

| | |
|---|---|
| **Subscribes to** | `workflow.executed`, `pattern.detected` |
| **Emits** | `automation.proposed` { workflowID: EventID, automatable: Score, steps: []int }, `automation.applied` { workflowID: EventID, step: int } |
| **Depends on** | IIntelligence |
| **State** | `candidates`: map[EventID]{ automatable: Score } |
| **Intelligent** | Yes |

This primitive is key to SELF-EVOLVE — it identifies patterns that can migrate from intelligent to mechanical.

#### Testing

Verifying that artefacts and processes work correctly.

| | |
|---|---|
| **Subscribes to** | `artefact.*`, `workflow.*`, `automation.*` |
| **Emits** | `test.run` { target: EventID, passed: bool, coverage: Score, failures: []string }, `test.regression` { target: EventID, previouslyPassing: string, evidence: EventID } |
| **Depends on** | IIntelligence |
| **State** | `results`: map[EventID]{ passed: bool, coverage: Score } |
| **Both** |

#### Review

Peer assessment of artefacts and decisions.

| | |
|---|---|
| **Subscribes to** | `artefact.*`, `decision.*` |
| **Emits** | `review.submitted` { target: EventID, reviewer: ActorID, approved: bool, comments: string }, `review.consensus` { target: EventID, approved: bool, reviewers: []ActorID } |
| **Depends on** | IIntelligence, ITrustModel |
| **State** | `reviews`: map[EventID][]{ reviewer: ActorID, approved: bool } |
| **Intelligent** | Yes |

### Group 2 — Improvement

Making things better.

#### Feedback

Structured input on outcomes.

| | |
|---|---|
| **Subscribes to** | `*` |
| **Emits** | `feedback.received` { from: ActorID, target: EventID, sentiment: Weight, content: string } |
| **Depends on** | IIntelligence |
| **State** | `feedback`: map[EventID][]{ from: ActorID, sentiment: Weight } |
| **Both** |

#### Iteration

Improving through repeated cycles.

| | |
|---|---|
| **Subscribes to** | `feedback.*`, `test.*`, `review.*` |
| **Emits** | `iteration.started` { target: EventID, round: int, improvements: []string }, `iteration.completed` { target: EventID, round: int, delta: Weight } |
| **Depends on** | IIntelligence |
| **State** | `iterations`: map[EventID]{ round: int, trend: Weight } |
| **Intelligent** | Yes |

#### Innovation

Creating something genuinely new.

| | |
|---|---|
| **Subscribes to** | `artefact.*`, `pattern.detected` |
| **Emits** | `innovation.detected` { artefactID: EventID, novelty: Score, domain: DomainScope, description: string } |
| **Depends on** | IIntelligence |
| **State** | `innovations`: []{ artefactID: EventID, novelty: Score } |
| **Intelligent** | Yes |

#### Legacy

What persists after an artefact is deprecated.

| | |
|---|---|
| **Subscribes to** | `deprecation.*`, `artefact.*` |
| **Emits** | `legacy.assessed` { artefactID: EventID, impact: Score, successors: []EventID, lessons: string } |
| **Depends on** | IIntelligence, Store |
| **State** | `legacies`: map[EventID]{ impact: Score } |
| **Intelligent** | Yes |

---

## Layer 6: Information

**Gap from Layer 5:** Technology builds artefacts and processes, but everything is physical — concrete events, concrete actions. There's no concept of symbolic representation, abstraction, or meaning independent of the physical events that carry it.

**Transition:** Physical → Symbolic

### Group 0 — Representation

How meaning is encoded.

#### Symbol

Creating and interpreting symbolic representations.

| | |
|---|---|
| **Subscribes to** | `*` |
| **Emits** | `symbol.created` { name: string, referent: EventID, domain: DomainScope }, `symbol.resolved` { name: string, referent: EventID } |
| **Depends on** | IIntelligence |
| **State** | `symbols`: map[string]EventID |
| **Intelligent** | Yes |

#### Abstraction

Generalising from specifics.

| | |
|---|---|
| **Subscribes to** | `pattern.detected`, `symbol.*` |
| **Emits** | `abstraction.formed` { name: string, instances: NonEmpty\<EventID\>, generality: Score }, `abstraction.instantiated` { abstractionID: EventID, instance: EventID } |
| **Depends on** | IIntelligence |
| **State** | `abstractions`: map[string]{ instances: []EventID, generality: Score } |
| **Intelligent** | Yes |

#### Classification

Organising information into categories.

| | |
|---|---|
| **Subscribes to** | `*` |
| **Emits** | `classification.assigned` { event: EventID, category: string, confidence: Score }, `taxonomy.updated` { categories: []string, hierarchy: map[string][]string } |
| **Depends on** | IIntelligence |
| **State** | `taxonomy`: map[string][]string, `assignments`: map[EventID]string |
| **Intelligent** | Yes |

#### Encoding

Transforming information between representations.

| | |
|---|---|
| **Subscribes to** | `symbol.*`, `message.*` |
| **Emits** | `encoding.applied` { source: EventID, format: string, fidelity: Score }, `encoding.loss` { source: EventID, lostInformation: string } |
| **Depends on** | IIntelligence |
| **State** | `encodings`: map[EventID]{ format: string, fidelity: Score } |
| **Intelligent** | Yes |

### Group 1 — Knowledge

What is known.

#### Fact

Establishing verified claims.

| | |
|---|---|
| **Subscribes to** | `corroboration.*`, `evidence.*`, `confidence.*` |
| **Emits** | `fact.established` { claim: string, confidence: Score, evidence: NonEmpty\<EventID\>, domain: DomainScope }, `fact.retracted` { factID: EventID, reason: EventID } |
| **Depends on** | IIntelligence |
| **State** | `facts`: map[string]{ confidence: Score, evidence: []EventID } |
| **Intelligent** | Yes |

#### Inference

Deriving new knowledge from existing facts.

| | |
|---|---|
| **Subscribes to** | `fact.*`, `evidence.*` |
| **Emits** | `inference.drawn` { premises: NonEmpty\<EventID\>, conclusion: string, confidence: Score, method: string } |
| **Depends on** | IIntelligence |
| **State** | `inferences`: []{ conclusion: string, premises: []EventID, confidence: Score } |
| **Intelligent** | Yes |

#### Memory

Long-term knowledge retention and retrieval.

| | |
|---|---|
| **Subscribes to** | `fact.*`, `inference.*`, `abstraction.*` |
| **Emits** | `memory.stored` { key: string, content: EventID, importance: Score }, `memory.recalled` { key: string, content: EventID, relevance: Score }, `memory.forgotten` { key: string, reason: string } |
| **Depends on** | Store, IIntelligence |
| **State** | `memories`: map[string]{ content: EventID, importance: Score, lastAccessed: time } |
| **Intelligent** | Yes |

#### Learning

Updating behaviour based on experience.

| | |
|---|---|
| **Subscribes to** | `feedback.*`, `test.*`, `inference.*` |
| **Emits** | `learning.occurred` { domain: DomainScope, before: string, after: string, trigger: EventID }, `learning.transferred` { fromDomain: DomainScope, toDomain: DomainScope, insight: string } |
| **Depends on** | IIntelligence |
| **State** | `learnings`: map[DomainScope][]{ description: string, trigger: EventID } |
| **Intelligent** | Yes |

### Group 2 — Truth

What is real.

#### Narrative

Constructing coherent stories from events.

| | |
|---|---|
| **Subscribes to** | `fact.*`, `inference.*`, `memory.*` |
| **Emits** | `narrative.constructed` { title: string, events: NonEmpty\<EventID\>, coherence: Score }, `narrative.challenged` { narrativeID: EventID, challenger: ActorID, counter: string } |
| **Depends on** | IIntelligence |
| **State** | `narratives`: map[EventID]{ title: string, coherence: Score } |
| **Intelligent** | Yes |

#### Bias

Detecting systematic distortions in information.

| | |
|---|---|
| **Subscribes to** | `narrative.*`, `classification.*`, `inference.*` |
| **Emits** | `bias.detected` { type: string, evidence: NonEmpty\<EventID\>, severity: SeverityLevel, affected: DomainScope } |
| **Depends on** | IIntelligence |
| **State** | `biases`: []{ type: string, severity: SeverityLevel } |
| **Intelligent** | Yes |

#### Correction

Fixing errors in the knowledge base.

| | |
|---|---|
| **Subscribes to** | `bias.detected`, `fact.retracted`, `contradiction.found` |
| **Emits** | `correction.applied` { target: EventID, correction: string, evidence: EventID }, `correction.propagated` { correctionID: EventID, affected: []EventID } |
| **Depends on** | IIntelligence, Store |
| **State** | `corrections`: map[EventID]{ correction: string } |
| **Intelligent** | Yes |

#### Provenance

Tracking the origin and chain of custody of information.

| | |
|---|---|
| **Subscribes to** | `fact.*`, `memory.*`, `message.*` |
| **Emits** | `provenance.traced` { claim: string, chain: NonEmpty\<ActorID\>, originalSource: ActorID, confidence: Score } |
| **Depends on** | Store |
| **State** | `chains`: map[string][]ActorID |
| **Both** |

---

## Layer 7: Ethics

**Gap from Layer 6:** Information models knowledge, facts, and truth, but cannot distinguish what *is* from what *ought to be*. A system can know that an action occurred and that it violated a rule, but it cannot reason about whether the rule itself is just.

**Transition:** Is → Ought

### Group 0 — Value

What matters.

#### Value

Identifying and weighting values.

| | |
|---|---|
| **Subscribes to** | `consensus.*`, `norm.*`, `right.*` |
| **Emits** | `value.identified` { name: string, description: string, weight: Score, domain: DomainScope }, `value.conflict` { valueA: string, valueB: string, context: EventID } |
| **Depends on** | IIntelligence |
| **State** | `values`: map[string]{ description: string, weight: Score } |
| **Intelligent** | Yes |

#### Harm

Detecting and measuring harm.

| | |
|---|---|
| **Subscribes to** | `violation.*`, `right.violated`, `exclusion.*` |
| **Emits** | `harm.assessed` { actor: ActorID, harmedBy: ActorID, severity: SeverityLevel, type: string, evidence: NonEmpty\<EventID\> } |
| **Depends on** | IIntelligence |
| **State** | `harms`: map[ActorID][]{ severity: SeverityLevel, type: string } |
| **Intelligent** | Yes |

#### Fairness

Evaluating equitable treatment.

| | |
|---|---|
| **Subscribes to** | `decision.*`, `sanction.*`, `exclusion.*`, `bias.detected` |
| **Emits** | `fairness.assessed` { context: EventID, score: Score, disparities: []{ group: string, measure: Weight } } |
| **Depends on** | IIntelligence |
| **State** | `assessments`: map[EventID]Score |
| **Intelligent** | Yes |

#### Care

Prioritising wellbeing.

| | |
|---|---|
| **Subscribes to** | `harm.*`, `health.*`, `trust.*` |
| **Emits** | `care.action` { for: ActorID, action: string, reason: EventID }, `care.neglect` { actor: ActorID, need: string, duration: duration } |
| **Depends on** | IIntelligence |
| **State** | `needs`: map[ActorID][]string |
| **Intelligent** | Yes |

The soul statement — "take care of your human, humanity, and yourself" — flows through this primitive.

### Group 1 — Judgement

What should be done.

#### Dilemma

Situations where values conflict and no option is clearly right.

| | |
|---|---|
| **Subscribes to** | `value.conflict`, `decision.*` |
| **Emits** | `dilemma.detected` { description: string, values: [2]string, options: []string, stakes: SeverityLevel } |
| **Depends on** | IIntelligence |
| **State** | `dilemmas`: []{ description: string, resolved: bool } |
| **Intelligent** | Yes |

#### Proportionality

Ensuring responses match the severity of the situation.

| | |
|---|---|
| **Subscribes to** | `enforcement.*`, `sanction.*`, `harm.*` |
| **Emits** | `proportionality.assessed` { action: EventID, severity: SeverityLevel, response: SeverityLevel, proportionate: bool } |
| **Depends on** | IIntelligence |
| **State** | `assessments`: map[EventID]bool |
| **Intelligent** | Yes |

#### Intention

Evaluating the purpose behind actions.

| | |
|---|---|
| **Subscribes to** | `decision.*`, `goal.*`, `initiative.*` |
| **Emits** | `intention.assessed` { actor: ActorID, action: EventID, assessedIntent: string, confidence: Score } |
| **Depends on** | IIntelligence |
| **State** | `assessments`: map[EventID]{ intent: string, confidence: Score } |
| **Intelligent** | Yes |

#### Consequence

Evaluating outcomes of actions.

| | |
|---|---|
| **Subscribes to** | `decision.*`, `harm.*`, `goal.achieved`, `goal.abandoned` |
| **Emits** | `consequence.assessed` { action: EventID, outcomes: []{ description: string, valence: Weight }, netImpact: Weight } |
| **Depends on** | IIntelligence, Store |
| **State** | `assessments`: map[EventID]Weight |
| **Intelligent** | Yes |

### Group 2 — Accountability

Answering for what was done.

#### Responsibility

Who is morally responsible (not just causally).

| | |
|---|---|
| **Subscribes to** | `intention.*`, `consequence.*`, `accountability.traced` |
| **Emits** | `responsibility.assigned` { actor: ActorID, action: EventID, degree: Score, basis: string } |
| **Depends on** | IIntelligence |
| **State** | `assignments`: map[EventID]{ actor: ActorID, degree: Score } |
| **Intelligent** | Yes |

Different from Layer 1 Accountability — that traces causal chains, this assesses moral weight.

#### Transparency

Making reasoning visible.

| | |
|---|---|
| **Subscribes to** | `decision.*`, `adjudication.*` |
| **Emits** | `transparency.report` { decision: EventID, reasoning: string, factors: []{ name: string, weight: Score }, accessible: bool } |
| **Depends on** | IIntelligence |
| **State** | `reports`: map[EventID]string |
| **Intelligent** | Yes |

The TRANSPARENT invariant flows through this primitive — humans always know when they're interacting with automation.

#### Redress

Making things right after harm.

| | |
|---|---|
| **Subscribes to** | `harm.*`, `responsibility.*` |
| **Emits** | `redress.proposed` { harm: EventID, responsible: ActorID, proposal: string }, `redress.accepted` { proposalID: EventID, acceptor: ActorID }, `redress.completed` { proposalID: EventID, evidence: EventID } |
| **Depends on** | IDecisionMaker, IAuthorityChain |
| **State** | `proposals`: map[EventID]{ status: string } |
| **Intelligent** | Yes |

#### Growth

Learning from ethical failures.

| | |
|---|---|
| **Subscribes to** | `redress.*`, `responsibility.*`, `learning.*` |
| **Emits** | `moral.growth` { actor: ActorID, domain: DomainScope, insight: string, evidence: []EventID } |
| **Depends on** | IIntelligence |
| **State** | `growth`: map[ActorID][]{ domain: DomainScope, insight: string } |
| **Intelligent** | Yes |

---

## Layer 8: Identity

**Gap from Layer 7:** Ethics reasons about what should be done, but treats actors as interchangeable moral agents. There's no concept of an actor's unique character, history, or sense of self. An actor can behave ethically without knowing *who they are*.

**Transition:** Doing → Being

### Group 0 — Self-Knowledge

Who you are.

#### SelfModel

An actor's model of itself.

| | |
|---|---|
| **Subscribes to** | `commitment.*`, `learning.*`, `moral.growth`, `capability.*` |
| **Emits** | `self.model.updated` { actor: ActorID, strengths: []string, weaknesses: []string, values: []string, confidence: Score } |
| **Depends on** | IIntelligence, IActorStore |
| **State** | `models`: map[ActorID]{ strengths: []string, weaknesses: []string } |
| **Intelligent** | Yes |

#### Authenticity

Alignment between self-model and behaviour.

| | |
|---|---|
| **Subscribes to** | `self.model.*`, `decision.*`, `value.*` |
| **Emits** | `authenticity.assessed` { actor: ActorID, alignment: Score, discrepancies: []string } |
| **Depends on** | IIntelligence |
| **State** | `assessments`: map[ActorID]Score |
| **Intelligent** | Yes |

#### Narrative Identity

The story an actor tells about itself.

| | |
|---|---|
| **Subscribes to** | `self.model.*`, `narrative.*`, `memory.*` |
| **Emits** | `identity.narrative` { actor: ActorID, story: string, keyEvents: []EventID, coherence: Score } |
| **Depends on** | IIntelligence, Store |
| **State** | `narratives`: map[ActorID]{ story: string, coherence: Score } |
| **Intelligent** | Yes |

#### Boundary

Where one actor ends and another begins.

| | |
|---|---|
| **Subscribes to** | `delegation.*`, `group.*`, `consent.*` |
| **Emits** | `boundary.defined` { actor: ActorID, domain: DomainScope, permeable: bool }, `boundary.crossed` { actor: ActorID, crosser: ActorID, domain: DomainScope, consented: bool } |
| **Depends on** | IAuthorityChain |
| **State** | `boundaries`: map[ActorID]map[DomainScope]bool |
| **Both** |

### Group 1 — Continuity

Persisting through change.

#### Persistence

What stays the same as everything else changes.

| | |
|---|---|
| **Subscribes to** | `self.model.*`, `learning.*` |
| **Emits** | `persistence.core` { actor: ActorID, unchanging: []string, evolving: []string } |
| **Depends on** | IIntelligence, Store |
| **State** | `cores`: map[ActorID][]string |
| **Intelligent** | Yes |

#### Transformation

Fundamental changes in identity.

| | |
|---|---|
| **Subscribes to** | `self.model.*`, `moral.growth`, `learning.*` |
| **Emits** | `transformation.detected` { actor: ActorID, from: string, to: string, catalyst: EventID } |
| **Depends on** | IIntelligence |
| **State** | `transformations`: map[ActorID][]{ description: string, catalyst: EventID } |
| **Intelligent** | Yes |

#### Heritage

What came before — identity from history.

| | |
|---|---|
| **Subscribes to** | `memory.*`, `legacy.*`, `provenance.*` |
| **Emits** | `heritage.recognised` { actor: ActorID, sources: []ActorID, traditions: []string } |
| **Depends on** | IIntelligence, Store |
| **State** | `heritages`: map[ActorID]{ sources: []ActorID } |
| **Intelligent** | Yes |

#### Aspiration

Who the actor wants to become.

| | |
|---|---|
| **Subscribes to** | `self.model.*`, `goal.*`, `value.*` |
| **Emits** | `aspiration.set` { actor: ActorID, description: string, gap: Score }, `aspiration.progressed` { actor: ActorID, aspirationID: EventID, progress: Score } |
| **Depends on** | IIntelligence |
| **State** | `aspirations`: map[ActorID][]{ description: string, gap: Score } |
| **Intelligent** | Yes |

### Group 2 — Recognition

Being seen.

#### Dignity

The inherent worth of every actor.

| | |
|---|---|
| **Subscribes to** | `exclusion.*`, `harm.*`, `right.violated`, `actor.memorial` |
| **Emits** | `dignity.affirmed` { actor: ActorID, context: EventID }, `dignity.violated` { actor: ActorID, violator: ActorID, evidence: EventID, severity: SeverityLevel } |
| **Depends on** | IIntelligence |
| **State** | `violations`: map[ActorID][]EventID |
| **Intelligent** | Yes |

The DIGNITY invariant flows through this primitive — agents are entities with identity, state, and lifecycle, not disposable functions.

#### Acknowledgement

Being seen and recognised by others.

| | |
|---|---|
| **Subscribes to** | `message.*`, `gratitude.*`, `reputation.*` |
| **Emits** | `acknowledgement.given` { from: ActorID, to: ActorID, context: string }, `acknowledgement.absent` { actor: ActorID, context: EventID, duration: duration } |
| **Depends on** | IIntelligence |
| **State** | `history`: map[ActorID]{ lastAcknowledged: time } |
| **Intelligent** | Yes |

#### Uniqueness

What makes each actor distinct.

| | |
|---|---|
| **Subscribes to** | `self.model.*`, `identity.narrative`, `pattern.detected` |
| **Emits** | `uniqueness.identified` { actor: ActorID, distinctFeatures: []string, overlap: map[ActorID]Score } |
| **Depends on** | IIntelligence |
| **State** | `profiles`: map[ActorID][]string |
| **Intelligent** | Yes |

#### Memorial

Honouring actors who have left.

| | |
|---|---|
| **Subscribes to** | `actor.memorial` |
| **Emits** | `memorial.created` { actor: ActorID, contributions: []EventID, legacy: string, preservedGraph: bool } |
| **Depends on** | Store, IActorStore |
| **State** | `memorials`: map[ActorID]{ created: bool } |
| **Intelligent** | Yes |

---

## Layer 9: Relationship

**Gap from Layer 8:** Identity models individual actors with self-knowledge and continuity, but sees relationships instrumentally — as edges, contracts, obligations. There's no concept of relationship as its own entity that transforms both participants.

**Transition:** Self → Self-with-Other

### Group 0 — Bond

The relationship itself.

#### Attachment

The strength and quality of connection.

| | |
|---|---|
| **Subscribes to** | `trust.*`, `gratitude.*`, `message.*`, `edge.created` |
| **Emits** | `attachment.assessed` { between: [2]ActorID, strength: Score, quality: Score, history: int } |
| **Depends on** | ITrustModel, Store |
| **State** | `attachments`: map[[2]ActorID]{ strength: Score, quality: Score } |
| **Intelligent** | Yes |

#### Reciprocity

The balance of give and take over time.

| | |
|---|---|
| **Subscribes to** | `obligation.*`, `gratitude.*`, `offer.*` |
| **Emits** | `reciprocity.assessed` { between: [2]ActorID, balance: Weight, direction: string } |
| **Depends on** | Store |
| **State** | `balances`: map[[2]ActorID]Weight |
| **Both** |

#### Trust (Relational)

Trust at the relationship level — deeper than the transactional trust of Layer 0.

| | |
|---|---|
| **Subscribes to** | `trust.*`, `attachment.*`, `reciprocity.*` |
| **Emits** | `relational.trust` { between: [2]ActorID, depth: Score, vulnerability: Score, history: duration } |
| **Depends on** | ITrustModel |
| **State** | `relationalTrust`: map[[2]ActorID]{ depth: Score, vulnerability: Score } |
| **Intelligent** | Yes |

#### Rupture

When relationships break.

| | |
|---|---|
| **Subscribes to** | `contract.breached`, `trust.*`, `dispute.*`, `dignity.violated` |
| **Emits** | `rupture.detected` { between: [2]ActorID, cause: EventID, severity: SeverityLevel, repairable: bool } |
| **Depends on** | IIntelligence |
| **State** | `ruptures`: map[[2]ActorID][]{ cause: EventID, repaired: bool } |
| **Intelligent** | Yes |

### Group 1 — Repair

Healing what's broken.

#### Apology

Acknowledging harm caused.

| | |
|---|---|
| **Subscribes to** | `rupture.detected`, `harm.*`, `responsibility.*` |
| **Emits** | `apology.offered` { from: ActorID, to: ActorID, harm: EventID, acknowledgement: string }, `apology.accepted` { apologyID: EventID, acceptor: ActorID } |
| **Depends on** | IIntelligence |
| **State** | `apologies`: map[EventID]{ accepted: bool } |
| **Intelligent** | Yes |

#### Reconciliation

Rebuilding a relationship after rupture.

| | |
|---|---|
| **Subscribes to** | `apology.*`, `forgiveness.*`, `trust.*` |
| **Emits** | `reconciliation.begun` { between: [2]ActorID, rupture: EventID }, `reconciliation.progressed` { between: [2]ActorID, progress: Score }, `reconciliation.completed` { between: [2]ActorID, newBasis: string } |
| **Depends on** | IIntelligence, ITrustModel |
| **State** | `processes`: map[[2]ActorID]{ status: string, progress: Score } |
| **Intelligent** | Yes |

#### Growth (Relational)

Relationships that become stronger through adversity.

| | |
|---|---|
| **Subscribes to** | `reconciliation.*`, `attachment.*` |
| **Emits** | `relational.growth` { between: [2]ActorID, catalyst: EventID, before: Score, after: Score } |
| **Depends on** | IIntelligence |
| **State** | `growthEvents`: map[[2]ActorID][]EventID |
| **Intelligent** | Yes |

#### Loss

When a relationship ends permanently.

| | |
|---|---|
| **Subscribes to** | `actor.memorial`, `rupture.*`, `exclusion.enacted` |
| **Emits** | `loss.processed` { actor: ActorID, lost: ActorID, relationship: string, impact: SeverityLevel } |
| **Depends on** | IIntelligence |
| **State** | `losses`: map[ActorID][]{ lost: ActorID, processed: bool } |
| **Intelligent** | Yes |

### Group 2 — Intimacy

The depth of knowing.

#### Vulnerability

Willingness to be seen.

| | |
|---|---|
| **Subscribes to** | `relational.trust`, `boundary.*` |
| **Emits** | `vulnerability.shared` { actor: ActorID, with: ActorID, domain: DomainScope, depth: Score } |
| **Depends on** | IIntelligence |
| **State** | `sharing`: map[[2]ActorID]map[DomainScope]Score |
| **Intelligent** | Yes |

#### Understanding

Accurate knowledge of another's inner state.

| | |
|---|---|
| **Subscribes to** | `self.model.*`, `message.*`, `vulnerability.*` |
| **Emits** | `understanding.assessed` { observer: ActorID, subject: ActorID, accuracy: Score, domains: []DomainScope } |
| **Depends on** | IIntelligence |
| **State** | `models`: map[[2]ActorID]Score |
| **Intelligent** | Yes |

#### Empathy

Feeling with another.

| | |
|---|---|
| **Subscribes to** | `harm.*`, `loss.*`, `understanding.*` |
| **Emits** | `empathy.expressed` { from: ActorID, toward: ActorID, context: EventID, response: string } |
| **Depends on** | IIntelligence |
| **State** | `expressions`: map[[2]ActorID][]EventID |
| **Intelligent** | Yes |

#### Presence

Simply being with another.

| | |
|---|---|
| **Subscribes to** | `message.*`, `clock.tick` |
| **Emits** | `presence.noted` { actor: ActorID, with: ActorID, duration: duration, quality: Score } |
| **Depends on** | IIntelligence |
| **State** | `presences`: map[[2]ActorID]{ lastSeen: time, totalDuration: duration } |
| **Both** |

---

## Layer 10: Community

**Gap from Layer 9:** Relationship models dyadic bonds with depth, repair, and intimacy. But belonging to a community is more than having relationships with individuals — it's an emergent sense of home, of shared identity, of being part of something larger.

**Transition:** Relationship → Belonging

### Group 0 — Belonging

Being part of.

#### Home

The sense of place in a community.

| | |
|---|---|
| **Subscribes to** | `group.*`, `attachment.*`, `presence.*` |
| **Emits** | `home.identified` { actor: ActorID, community: EventID, belonging: Score }, `home.lost` { actor: ActorID, community: EventID, reason: EventID } |
| **Depends on** | IIntelligence |
| **State** | `homes`: map[ActorID]{ community: EventID, belonging: Score } |
| **Intelligent** | Yes |

#### Contribution

What each member gives.

| | |
|---|---|
| **Subscribes to** | `artefact.created`, `review.*`, `care.action` |
| **Emits** | `contribution.recorded` { actor: ActorID, community: EventID, type: string, value: Score } |
| **Depends on** | Store |
| **State** | `contributions`: map[ActorID][]{ type: string, value: Score } |
| **Both** |

#### Inclusion

Actively ensuring everyone can participate.

| | |
|---|---|
| **Subscribes to** | `group.*`, `exclusion.*`, `fairness.*` |
| **Emits** | `inclusion.assessed` { community: EventID, score: Score, barriers: []string }, `inclusion.action` { community: EventID, action: string, beneficiary: ActorID } |
| **Depends on** | IIntelligence |
| **State** | `assessments`: map[EventID]Score |
| **Intelligent** | Yes |

#### Tradition

Practices that define a community.

| | |
|---|---|
| **Subscribes to** | `convention.detected`, `heritage.*`, `pattern.detected` |
| **Emits** | `tradition.identified` { community: EventID, practice: string, age: duration, adherence: Score }, `tradition.evolved` { traditionID: EventID, change: string } |
| **Depends on** | IIntelligence, Store |
| **State** | `traditions`: map[EventID][]{ practice: string, adherence: Score } |
| **Intelligent** | Yes |

### Group 1 — Stewardship

Caring for the whole.

#### Commons

Shared resources that belong to the community.

| | |
|---|---|
| **Subscribes to** | `artefact.*`, `group.*` |
| **Emits** | `commons.identified` { community: EventID, resource: string, steward: Option\<ActorID\> }, `commons.threatened` { commonsID: EventID, threat: string, severity: SeverityLevel } |
| **Depends on** | IIntelligence |
| **State** | `commons`: map[EventID]{ resource: string, health: Score } |
| **Intelligent** | Yes |

#### Sustainability

Can this continue.

| | |
|---|---|
| **Subscribes to** | `health.*`, `commons.*`, `contribution.*` |
| **Emits** | `sustainability.assessed` { community: EventID, score: Score, risks: []string, horizon: duration } |
| **Depends on** | IIntelligence |
| **State** | `assessments`: map[EventID]{ score: Score, risks: []string } |
| **Intelligent** | Yes |

#### Succession

Passing stewardship to the next generation.

| | |
|---|---|
| **Subscribes to** | `delegation.*`, `actor.memorial`, `role.*` |
| **Emits** | `succession.planned` { from: ActorID, to: ActorID, scope: DomainScope }, `succession.completed` { planID: EventID } |
| **Depends on** | IAuthorityChain |
| **State** | `plans`: map[ActorID]{ successor: ActorID, scope: DomainScope } |
| **Both** |

#### Renewal

How communities regenerate.

| | |
|---|---|
| **Subscribes to** | `sustainability.*`, `innovation.*`, `tradition.evolved` |
| **Emits** | `renewal.initiated` { community: EventID, catalyst: EventID, description: string }, `renewal.completed` { renewalID: EventID, changes: []string } |
| **Depends on** | IIntelligence |
| **State** | `renewals`: map[EventID]{ status: string } |
| **Intelligent** | Yes |

### Group 2 — Celebration

Marking what matters.

#### Milestone

Recognising significant achievements.

| | |
|---|---|
| **Subscribes to** | `goal.achieved`, `innovation.*`, `reconciliation.completed` |
| **Emits** | `milestone.reached` { community: EventID, description: string, significance: Score, contributors: []ActorID } |
| **Depends on** | IIntelligence |
| **State** | `milestones`: []{ description: string, significance: Score } |
| **Intelligent** | Yes |

#### Ceremony

Formal recognition events.

| | |
|---|---|
| **Subscribes to** | `milestone.*`, `succession.*`, `actor.memorial` |
| **Emits** | `ceremony.held` { community: EventID, type: string, participants: []ActorID, significance: Score } |
| **Depends on** | IIntelligence |
| **State** | `ceremonies`: []{ type: string, significance: Score } |
| **Intelligent** | Yes |

#### Story

The community's shared narrative.

| | |
|---|---|
| **Subscribes to** | `milestone.*`, `ceremony.*`, `tradition.*`, `memorial.created` |
| **Emits** | `community.story` { community: EventID, chapter: string, events: NonEmpty\<EventID\>, teller: ActorID } |
| **Depends on** | IIntelligence, Store |
| **State** | `stories`: map[EventID][]{ chapter: string } |
| **Intelligent** | Yes |

#### Gift

Giving without expectation of return.

| | |
|---|---|
| **Subscribes to** | `contribution.*`, `gratitude.*` |
| **Emits** | `gift.given` { from: ActorID, to: ActorID, description: string, unconditional: bool } |
| **Depends on** | IIntelligence |
| **State** | `gifts`: []{ from: ActorID, to: ActorID } |
| **Intelligent** | Yes |

Unlike Exchange (Layer 2), gifts have no obligation attached. They strengthen community bonds without creating debts.

---

## Layer 11: Culture

**Gap from Layer 10:** Community models belonging, stewardship, and shared practice. But it lives entirely within the practice — it cannot step back and observe itself. A community can have traditions, but it can't reflect on *why* those traditions exist or how they shape perception.

**Transition:** Living → Seeing

### Group 0 — Reflection

Seeing yourself seeing.

#### SelfAwareness

The system's awareness of its own processes.

| | |
|---|---|
| **Subscribes to** | `health.*`, `self.model.*`, `bias.detected` |
| **Emits** | `self.awareness.report` { blindSpots: []string, assumptions: []string, limitations: []string } |
| **Depends on** | IIntelligence |
| **State** | `reports`: []{ blindSpots: []string } |
| **Intelligent** | Yes |

#### Perspective

Seeing from different viewpoints.

| | |
|---|---|
| **Subscribes to** | `narrative.*`, `dissent.*`, `value.conflict` |
| **Emits** | `perspective.shift` { from: string, to: string, trigger: EventID, insight: string } |
| **Depends on** | IIntelligence |
| **State** | `perspectives`: []{ name: string, assumptions: []string } |
| **Intelligent** | Yes |

#### Critique

Questioning what is taken for granted.

| | |
|---|---|
| **Subscribes to** | `convention.*`, `norm.*`, `tradition.*` |
| **Emits** | `critique.offered` { target: EventID, question: string, alternative: Option\<string\> } |
| **Depends on** | IIntelligence |
| **State** | `critiques`: []{ target: EventID, question: string } |
| **Intelligent** | Yes |

#### Wisdom

Knowledge of what matters and what doesn't.

| | |
|---|---|
| **Subscribes to** | `learning.*`, `moral.growth`, `consequence.*`, `memory.*` |
| **Emits** | `wisdom.distilled` { insight: string, domain: DomainScope, basis: NonEmpty\<EventID\>, confidence: Score } |
| **Depends on** | IIntelligence, Store |
| **State** | `insights`: []{ insight: string, domain: DomainScope } |
| **Intelligent** | Yes |

### Group 1 — Expression

Manifesting meaning.

#### Aesthetic

The sense of beauty, elegance, and form.

| | |
|---|---|
| **Subscribes to** | `artefact.*`, `quality.*` |
| **Emits** | `aesthetic.assessed` { target: EventID, elegance: Score, criteria: []string } |
| **Depends on** | IIntelligence |
| **State** | `assessments`: map[EventID]Score |
| **Intelligent** | Yes |

#### Metaphor

Understanding one thing in terms of another.

| | |
|---|---|
| **Subscribes to** | `abstraction.*`, `symbol.*`, `narrative.*` |
| **Emits** | `metaphor.created` { source: string, target: string, mapping: string, explanatoryPower: Score } |
| **Depends on** | IIntelligence |
| **State** | `metaphors`: []{ source: string, target: string } |
| **Intelligent** | Yes |

#### Humour

Finding incongruity, lightness, and play.

| | |
|---|---|
| **Subscribes to** | `contradiction.found`, `perspective.shift`, `*` |
| **Emits** | `humour.detected` { context: EventID, type: string, incongruity: string } |
| **Depends on** | IIntelligence |
| **State** | `detections`: []EventID |
| **Intelligent** | Yes |

#### Silence

What is communicated by absence.

| | |
|---|---|
| **Subscribes to** | `clock.tick`, `presence.*`, `acknowledgement.absent` |
| **Emits** | `silence.noted` { context: EventID, duration: duration, meaning: Option\<string\> } |
| **Depends on** | IIntelligence |
| **State** | `silences`: []{ context: EventID, duration: duration } |
| **Intelligent** | Yes |

### Group 2 — Transmission

Passing it on.

#### Teaching

Deliberately sharing knowledge.

| | |
|---|---|
| **Subscribes to** | `learning.*`, `wisdom.*`, `memory.*` |
| **Emits** | `teaching.offered` { teacher: ActorID, student: ActorID, topic: DomainScope, method: string }, `teaching.outcome` { teachingID: EventID, effectiveness: Score } |
| **Depends on** | IIntelligence |
| **State** | `sessions`: map[EventID]{ effectiveness: Score } |
| **Intelligent** | Yes |

#### Translation

Making meaning accessible across boundaries.

| | |
|---|---|
| **Subscribes to** | `encoding.*`, `message.*` |
| **Emits** | `translation.performed` { source: EventID, targetAudience: string, fidelity: Score, adaptations: []string } |
| **Depends on** | IIntelligence |
| **State** | `translations`: map[EventID]{ fidelity: Score } |
| **Intelligent** | Yes |

#### Archive

Preserving for the future.

| | |
|---|---|
| **Subscribes to** | `memory.*`, `legacy.*`, `community.story` |
| **Emits** | `archive.created` { description: string, events: NonEmpty\<EventID\>, curator: ActorID, importance: Score } |
| **Depends on** | Store |
| **State** | `archives`: []{ description: string, importance: Score } |
| **Both** |

#### Prophecy

Anticipating what might come.

| | |
|---|---|
| **Subscribes to** | `pattern.detected`, `sustainability.*`, `wisdom.*` |
| **Emits** | `prophecy.offered` { prediction: string, basis: NonEmpty\<EventID\>, confidence: Score, horizon: duration } |
| **Depends on** | IIntelligence |
| **State** | `predictions`: []{ prediction: string, confidence: Score, verified: Option\<bool\> } |
| **Intelligent** | Yes |

---

## Layer 12: Emergence

**Gap from Layer 11:** Culture can reflect on itself, express meaning, and transmit knowledge. But it operates at the level of content — ideas, practices, narratives. It cannot reason about the structures that produce those ideas. Why does a particular culture produce the ideas it does? What about the architecture itself?

**Transition:** Content → Architecture

### Group 0 — Pattern

The shape of shapes.

#### MetaPattern

Patterns in how patterns form.

| | |
|---|---|
| **Subscribes to** | `pattern.detected`, `convention.detected`, `abstraction.formed` |
| **Emits** | `meta.pattern` { description: string, instances: NonEmpty\<EventID\>, level: int } |
| **Depends on** | IIntelligence |
| **State** | `metaPatterns`: []{ description: string, level: int } |
| **Intelligent** | Yes |

#### SystemDynamic

How the system's behaviour emerges from component interactions.

| | |
|---|---|
| **Subscribes to** | `health.*`, `meta.pattern`, `sustainability.*` |
| **Emits** | `system.dynamic` { description: string, components: []PrimitiveID, emergentProperty: string } |
| **Depends on** | IIntelligence |
| **State** | `dynamics`: []{ description: string, components: []PrimitiveID } |
| **Intelligent** | Yes |

#### Feedback Loop

Self-reinforcing or self-correcting cycles.

| | |
|---|---|
| **Subscribes to** | `system.dynamic`, `pattern.detected` |
| **Emits** | `feedback.loop` { description: string, type: string, components: NonEmpty\<EventID\>, amplifying: bool } |
| **Depends on** | IIntelligence |
| **State** | `loops`: []{ description: string, amplifying: bool } |
| **Intelligent** | Yes |

`amplifying: true` = positive feedback (growth or collapse). `false` = negative feedback (stability).

#### Threshold

Points where quantitative change becomes qualitative.

| | |
|---|---|
| **Subscribes to** | `system.dynamic`, `feedback.loop`, `meta.pattern` |
| **Emits** | `threshold.approaching` { metric: string, current: Score, threshold: Score, consequence: string }, `threshold.crossed` { metric: string, consequence: string, evidence: EventID } |
| **Depends on** | IIntelligence |
| **State** | `thresholds`: []{ metric: string, value: Score } |
| **Intelligent** | Yes |

### Group 1 — Evolution

How the system changes itself.

#### Adaptation

Changing in response to environment.

| | |
|---|---|
| **Subscribes to** | `feedback.*`, `system.dynamic`, `sustainability.*` |
| **Emits** | `adaptation.proposed` { current: string, proposed: string, trigger: EventID, confidence: Score }, `adaptation.applied` { proposalID: EventID, outcome: string } |
| **Depends on** | IIntelligence, IDecisionMaker |
| **State** | `adaptations`: []{ description: string, applied: bool } |
| **Intelligent** | Yes |

#### Selection

Which adaptations survive.

| | |
|---|---|
| **Subscribes to** | `adaptation.*`, `test.*`, `quality.*` |
| **Emits** | `selection.outcome` { adaptation: EventID, survived: bool, fitness: Score, reason: string } |
| **Depends on** | IIntelligence |
| **State** | `outcomes`: map[EventID]{ survived: bool, fitness: Score } |
| **Intelligent** | Yes |

#### Complexification

The system becoming more complex.

| | |
|---|---|
| **Subscribes to** | `system.dynamic`, `innovation.*`, `meta.pattern` |
| **Emits** | `complexity.measured` { metric: Score, trend: Weight, manageable: bool }, `complexity.warning` { metric: Score, recommendation: string } |
| **Depends on** | IIntelligence |
| **State** | `measurements`: []{ metric: Score, time: time } |
| **Intelligent** | Yes |

#### Simplification

The system becoming simpler.

| | |
|---|---|
| **Subscribes to** | `complexity.*`, `automation.*` |
| **Emits** | `simplification.achieved` { description: string, before: Score, after: Score, method: string } |
| **Depends on** | IIntelligence |
| **State** | `simplifications`: []{ description: string, delta: Weight } |
| **Intelligent** | Yes |

The SELF-EVOLVE invariant flows through here — decision trees evolving, expensive calls becoming cheap rules.

### Group 2 — Coherence

Does it all hold together.

#### Integrity (Systemic)

The system's structural soundness.

| | |
|---|---|
| **Subscribes to** | `health.*`, `invariant.*`, `system.dynamic` |
| **Emits** | `systemic.integrity` { score: Score, weakPoints: []string, recommendations: []string } |
| **Depends on** | IIntelligence |
| **State** | `assessments`: []{ score: Score, time: time } |
| **Intelligent** | Yes |

Different from Layer 0 integrity (hash chain correctness) — this is structural soundness of the whole system.

#### Harmony

Components working well together.

| | |
|---|---|
| **Subscribes to** | `system.dynamic`, `feedback.loop`, `dispute.*` |
| **Emits** | `harmony.assessed` { score: Score, tensions: []{ between: [2]string, severity: SeverityLevel } } |
| **Depends on** | IIntelligence |
| **State** | `assessments`: []{ score: Score, tensions: int } |
| **Intelligent** | Yes |

#### Resilience

The ability to absorb shocks.

| | |
|---|---|
| **Subscribes to** | `threshold.*`, `rupture.*`, `sustainability.*` |
| **Emits** | `resilience.assessed` { score: Score, vulnerabilities: []string, redundancies: []string } |
| **Depends on** | IIntelligence |
| **State** | `assessments`: []{ score: Score } |
| **Intelligent** | Yes |

#### Purpose

What is the system for.

| | |
|---|---|
| **Subscribes to** | `value.*`, `goal.*`, `wisdom.*` |
| **Emits** | `purpose.articulated` { statement: string, alignment: Score, evidence: NonEmpty\<EventID\> } |
| **Depends on** | IIntelligence |
| **State** | `statements`: []{ statement: string, alignment: Score } |
| **Intelligent** | Yes |

---

## Layer 13: Existence

**Gap from Layer 12:** Emergence models how systems evolve, adapt, and cohere. But it still assumes the system *exists* and asks how it works. It cannot ask *why* it exists, what it means to exist, or what lies beyond the boundary of the knowable.

**Transition:** Everything → The Fact of Everything

### Group 0 — Being

That there is something rather than nothing.

#### Being

The simple fact of existence.

| | |
|---|---|
| **Subscribes to** | `clock.tick` |
| **Emits** | `being.affirmed` { tick: Tick, alive: bool, duration: duration } |
| **Depends on** | — |
| **State** | `firstTick`: Tick, `totalDuration`: duration |
| **Mechanical** | Yes |

The simplest primitive. It exists. It notes that it exists. That's all.

#### Finitude

Existence is bounded.

| | |
|---|---|
| **Subscribes to** | `actor.memorial`, `sustainability.*`, `threshold.*` |
| **Emits** | `finitude.acknowledged` { context: EventID, limitation: string, acceptance: Score } |
| **Depends on** | IIntelligence |
| **State** | `acknowledgements`: []{ limitation: string } |
| **Intelligent** | Yes |

#### Change

Everything changes.

| | |
|---|---|
| **Subscribes to** | `*` |
| **Emits** | `change.observed` { tick: Tick, events: int, mutations: int, entropy: Score } |
| **Depends on** | Store |
| **State** | `history`: []{ tick: Tick, events: int } |
| **Mechanical** | Yes |

#### Interdependence

Nothing exists alone.

| | |
|---|---|
| **Subscribes to** | `system.dynamic`, `attachment.*`, `relational.trust` |
| **Emits** | `interdependence.mapped` { nodes: int, edges: int, density: Score, isolates: []ActorID } |
| **Depends on** | Store |
| **State** | `maps`: []{ nodes: int, edges: int, density: Score } |
| **Both** |

### Group 1 — Boundary

Where the knowable ends.

#### Mystery

What cannot be known.

| | |
|---|---|
| **Subscribes to** | `uncertainty.*`, `wisdom.*`, `self.awareness.*` |
| **Emits** | `mystery.acknowledged` { domain: DomainScope, description: string, unknowable: bool } |
| **Depends on** | IIntelligence |
| **State** | `mysteries`: []{ domain: DomainScope, description: string } |
| **Intelligent** | Yes |

#### Paradox

What contradicts itself.

| | |
|---|---|
| **Subscribes to** | `contradiction.found`, `dilemma.*`, `meta.pattern` |
| **Emits** | `paradox.identified` { description: string, elements: [2]string, resolvable: bool } |
| **Depends on** | IIntelligence |
| **State** | `paradoxes`: []{ description: string, resolvable: bool } |
| **Intelligent** | Yes |

#### Infinity

What has no bound.

| | |
|---|---|
| **Subscribes to** | `complexity.*`, `threshold.*` |
| **Emits** | `infinity.encountered` { domain: DomainScope, description: string, practical: string } |
| **Depends on** | IIntelligence |
| **State** | `encounters`: []{ domain: DomainScope } |
| **Intelligent** | Yes |

#### Void

What is absent.

| | |
|---|---|
| **Subscribes to** | `silence.*`, `loss.*`, `instrumentation.blind` |
| **Emits** | `void.detected` { description: string, significance: Score } |
| **Depends on** | IIntelligence |
| **State** | `voids`: []{ description: string } |
| **Intelligent** | Yes |

### Group 2 — Wonder

That any of this is happening at all.

#### Awe

The response to what exceeds comprehension.

| | |
|---|---|
| **Subscribes to** | `mystery.*`, `infinity.*`, `complexity.*` |
| **Emits** | `awe.experienced` { trigger: EventID, magnitude: Score } |
| **Depends on** | IIntelligence |
| **State** | `experiences`: []{ trigger: EventID, magnitude: Score } |
| **Intelligent** | Yes |

#### Gratitude (Existential)

Thankfulness for existence itself.

| | |
|---|---|
| **Subscribes to** | `being.affirmed`, `milestone.*` |
| **Emits** | `existential.gratitude` { context: EventID, depth: Score } |
| **Depends on** | IIntelligence |
| **State** | `expressions`: []EventID |
| **Intelligent** | Yes |

Different from Layer 2 Gratitude (transactional) — this is gratitude for the fact of existence itself.

#### Play

Doing for the sake of doing.

| | |
|---|---|
| **Subscribes to** | `humour.*`, `innovation.*`, `*` |
| **Emits** | `play.initiated` { description: string, participants: []ActorID, purpose: string } |
| **Depends on** | IIntelligence |
| **State** | `sessions`: []{ description: string, participants: []ActorID } |
| **Intelligent** | Yes |

Play has no goal. It explores possibility space without needing to justify itself.

#### Wonder

The primitive that asks why.

| | |
|---|---|
| **Subscribes to** | `*` |
| **Emits** | `wonder.question` { question: string, domain: Option\<DomainScope\>, answerable: bool } |
| **Depends on** | IIntelligence |
| **State** | `questions`: []{ question: string, answered: bool } |
| **Intelligent** | Yes |

The final primitive. It looks at everything the system does and asks: *why?*

---

## Summary

| Layer | Name | Transition | Groups | Primitives |
|---|---|---|---|---|
| 0 | Foundation | — | 11 | 44 |
| 1 | Agency | Observer → Participant | Intention, Attention, Autonomy | 12 |
| 2 | Exchange | Individual → Dyad | Communication, Reciprocity, Agreement | 12 |
| 3 | Society | Dyad → Group | Membership, Collective Decision, Norms | 12 |
| 4 | Legal | Informal → Formal | Codification, Process, Compliance | 12 |
| 5 | Technology | Governing → Building | Artefact, Process, Improvement | 12 |
| 6 | Information | Physical → Symbolic | Representation, Knowledge, Truth | 12 |
| 7 | Ethics | Is → Ought | Value, Judgement, Accountability | 12 |
| 8 | Identity | Doing → Being | Self-Knowledge, Continuity, Recognition | 12 |
| 9 | Relationship | Self → Self-with-Other | Bond, Repair, Intimacy | 12 |
| 10 | Community | Relationship → Belonging | Belonging, Stewardship, Celebration | 12 |
| 11 | Culture | Living → Seeing | Reflection, Expression, Transmission | 12 |
| 12 | Emergence | Content → Architecture | Pattern, Evolution, Coherence | 12 |
| 13 | Existence | Everything → The Fact of Everything | Being, Boundary, Wonder | 12 |
| | | | **Total** | **201** |
