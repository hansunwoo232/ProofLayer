# Customer discovery interview guide

**Purpose:** Learn from past behavior about the frequency, impact, current
workaround, owner, and budget path of detection-pipeline failures. Do not sell
the product during discovery.  
**Duration:** 30 minutes  
**Roles:** SOC Manager, Detection or SIEM Engineer, MSSP leader  
**Recording:** Never record without explicit consent. Do not retain sensitive
company, incident, or customer data in notes.

## Interview flow

| Time | Section | Goal |
|---:|---|---|
| 0–3 min | Context and consent | Confirm purpose, confidentiality, and timing |
| 3–15 min | Past events | Understand the latest real failure and discovery path |
| 15–23 min | Current process | Measure people, tools, time, frequency, and gaps |
| 23–27 min | Ownership and budget | Identify decision owner, pilot gate, and buying path |
| 27–30 min | Close | Ask permission to follow up and request a referral |

## Opening

“I am researching a focused problem: proving that detections work from endpoint
telemetry through the SIEM alert, not merely that rules exist. This is not a
product pitch. I would like to understand real examples from the last six
months and how your team handles them today. Please do not share sensitive
company or incident details.”

## Core questions

1. In the last six months, did a log source, parser, or detection fail silently?
2. Who noticed it, how, and after how long?
3. What was the impact: visibility loss, false negative, analysis time, or audit
   risk?
4. How is a new detection tested before production?
5. How often is the complete endpoint-to-alert path tested?
6. How many people and how much time does this process consume?
7. Which tools, scripts, or services are used today?
8. Which part of the current approach creates the least confidence?
9. What is retested after a migration or configuration change?
10. Who owns the technical problem and who owns the budget?
11. What security, legal, and access conditions would a controlled pilot need?
12. Who else understands this problem particularly well?

## Role-specific prompts

### SOC Manager

- How are detection coverage and health reported to management?
- How do you prove the difference between “enabled” and “working”?
- How do you find every rule affected by a parser change?
- What evidence is shown to an auditor?
- What would make a pilot successful?

### Detection or SIEM Engineer

- How do you generate and distinguish test events?
- Which event fields and timestamps are verified?
- Is regression testing triggered by rule, parser, or sourcetype changes?
- Can failures be isolated to telemetry, ingestion, parsing, and rule stages?
- How easy is it to rerun an identical test after a fix?

### MSSP leader

- How is detection health standardized across customers?
- How are shared parser or content regressions contained?
- Does the customer SLA cover rule existence or verified behavior?
- How are customer approvals and tenant separation handled?
- What commercial model would fit an existing managed service?

## Budget and decision path

- Has the organization spent time or money on this problem in the last year?
- Who last evaluated a similar tool and how did that process work?
- Who approves a pilot and which security documents are required?
- Which budget is the closest fit: SOC/SIEM, detection engineering, consulting,
  or innovation?
- What risk most often delays evaluation?

## Avoid

- “Would you use this product?”
- “Do you like the idea?”
- “Would you want AI?”
- “How much would you pay?” before learning the real process and past spend
- Long leading questions that defend the product

## Problem-signal score

Score each dimension from 0 to 2.

| Dimension | 0 | 1 | 2 |
|---|---|---|---|
| Frequency | Never | Annual or rare | Monthly or more |
| Impact | Negligible | Operational burden | Visibility, incident, or audit risk |
| Current spend | None | Manual time | Tool, project, or budget |
| Urgency | Unknown | This year | Within 90 days |
| Access | No follow-up | Demo accepted | Pilot owner and date |

`0–3 weak`, `4–6 monitor`, `7–8 strong problem`, `9–10 design-partner candidate`.

## Closing

“I will use what you shared only as anonymized product research. May I send you
a short learning summary and a future demo? Could you introduce me to someone
who experiences the same problem?”
