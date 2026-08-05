# Weekly sprint and decision cadence

## Operating principle

Three evidence streams move together every week:

1. **Product evidence:** the smallest working, tested end-to-end slice.
2. **Customer evidence:** an interview, demo, or pilot learning.
3. **Company evidence:** a decision, risk, brand, deployment, or funding step.

A week of code alone is not considered successful.

## Capacity

- 50% product and testing
- 30% customer discovery or demos
- 20% security, documentation, and operations
- At most three active work items at once
- At least one end-to-end product proof per sprint

Customer outreach remains subject to ADR-0002 and begins only after the MVP
demonstration gate.

## Rituals

### Monday — 45-minute planning

Review the scorecard and blockers, write one measurable weekly outcome, select
at most five items with owners and acceptance criteria, and name what leaves
scope if new work enters.

### Daily — 10-minute check-in

Record what finished, today's most important outcome, the blocker that must be
resolved within 24 hours, and any decision or risk-log update.

### Wednesday — 30-minute customer learning

After outreach begins, score interviews, extract repeated evidence and
contradictions, add only evidence-backed problems to the backlog, and assign
follow-ups.

### Friday — 45-minute demo and close

Demonstrate the working result, check acceptance criteria, update metrics, run
a short continue/stop/start retrospective, and record new decisions.

## Weekly scorecard

| Area | Target |
|---|---|
| Critical product outcome | 1 |
| Customer interviews | 2–3 after MVP gate |
| Strong new problem signals | 1+ after MVP gate |
| Live demo or pilot step | 1 when applicable |
| Closed P0/P1 security risks | As required |
| Aging active work | 0 |

## Workflow and done criteria

`Backlog → Ready → In progress → Review/Test → Done`

Blocked work keeps its workflow status and adds a reason, owner, and resolution
date. Done requires passed acceptance criteria, a tested failure path, reviewed
security impact, current documentation, demonstrable evidence, and a named next
step.

## ADR trigger

Create an ADR for trust-boundary or data-flow changes, technology choices,
breaking API/event-model changes, MVP scope changes, customer-data handling, or
accepted security risk.
