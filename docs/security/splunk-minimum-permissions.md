# Splunk Minimum-Permission Access Model

## Separation of duties

ProofLayer uses three distinct local identities in the PoC:

| Identity | Purpose | Allowed scope |
|---|---|---|
| Splunk `admin` | One-time lab provisioning | Create the test index, HEC input, role, and observer user |
| `prooflayer_lab` HEC token | Windows telemetry ingestion | Submit events only to `prooflayer_test` |
| `prooflayer_observer` | Runtime observation | Execute searches only against `prooflayer_test` |

The Runner never receives the Splunk administrator or observer credential. The
Observer never receives the HEC token.

## Observer role

The local role grants only:

- Capability: `search`
- Allowed index: `prooflayer_test`
- Default index: `prooflayer_test`

The role does not inherit `admin`, `power`, or `user`. Its negative acceptance
test confirms that an `_internal` search cannot return events.

## Secret handling

- Local credentials are generated with OpenSSL and stored only in ignored
  `deployments/splunk/.env`.
- Scripts never print secret values.
- Splunk ports bind to loopback.
- The Windows guest receives only the dedicated HEC token on ignored read-only
  lab media.
- Customer deployments must use a secret manager and trusted TLS; local `.env`
  and self-signed certificate exceptions do not carry forward.

## Runtime query constraints

- The Observer searches an exact canonical correlation ID.
- The search window comes from the signed job plus bounded clock skew.
- Queries specify `index=prooflayer_test`; no all-index search is permitted.
- Search results return minimum evidence fields, not raw event bodies.
- Query timeout, result count, and polling frequency are bounded.

The Go connector additionally enforces HTTPS, TLS 1.2 or newer, a 10-second
connection-check deadline, and a 64 KiB response cap. Its connection check uses
only `search index=prooflayer_test | stats count`; no raw event is returned.

The isolated loopback lab may explicitly allow its self-signed certificate.
That exception is rejected for non-loopback endpoints and is not a customer
deployment default.

## Validation command

```bash
cd deployments/splunk
./configure-observer-role.sh
```

The script proves an allowed `prooflayer_test` search and a denied `_internal`
data return.
