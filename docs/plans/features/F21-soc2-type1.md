# F21 — SOC 2 Type I

> Ship in **Milestone 4** (audit kickoff) → **Milestone 6** (report). Co-owners: `security-compliance` +
> `infra-sre`.

## Spec

Vanta (or Drata) as the control-evidence automation platform.

Control areas (per SOC 2 Trust Services Criteria):

| Area | Owner | Evidence source |
|---|---|---|
| Security / CC1–CC9 | `security-compliance` + `infra-sre` | Vanta + Vault + audit logs |
| Availability | `infra-sre` | uptime dashboards, runbooks, incident reports |
| Processing integrity | `credit-ledger` + `infra-sre` | hash-chain audit, reconciliation, replay |
| Confidentiality | `infra-sre` | KMS, Vault, network policies |
| Privacy | `security-compliance` | PII inventory, data retention, access controls |

Concrete controls to demonstrate:
- Access control: SSO for all infra tools; role separation; quarterly access review.
- Change management: PR review + CI gates + deploy approval.
- Encryption: at-rest (DB, S3) + in-transit (TLS 1.3 + mTLS internal).
- Logging: every credit/order/admin action audited; logs retained 1y minimum.
- Backups + DR: weekly DR drill; restore tested monthly.
- Incident response: documented runbook; tabletop exercise.
- Vendor management: registry of subprocessors.
- HR: background check, signed AUP, offboarding within 24h.

## Owning agents

- `security-compliance`: control authorship, evidence quality, audit firm liaison.
- `infra-sre`: technical control implementation + evidence pipeline.

## Contracts consumed / produced

- None directly. Produces `docs/soc2-controls.md` (control matrix) and Vanta evidence streams.

## Dependencies

- F01–F19 (controls exist where services exist).
- F22 (legal sign-off on regulatory framing influences a few controls).

## Sync points

- M1 — Vanta engaged.
- M2 — initial control mapping; evidence pipeline started.
- M4 — audit firm engaged; kickoff meeting.
- M5 — gap remediation.
- M6 — report achieved.

## Acceptance criteria

- [ ] SOC 2 Type I report achieved by end of Milestone 6.
- [ ] Control gaps closed before audit window opens.
- [ ] Quarterly access review process established.
- [ ] DR drill passed.

## Milestone

- M4 (Gate 4): kickoff.
- M6 (Gate 6): report.
