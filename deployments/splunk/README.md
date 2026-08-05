# Local Splunk Lab

This deployment runs a single-node Splunk Enterprise lab for ProofLayer's
endpoint-to-SIEM validation work.

## Architecture note

The official Splunk image is currently published for AMD64. Apple Silicon runs
this image through Docker's `linux/amd64` emulation. This is acceptable for the
local proof of concept, but it is not the target production architecture.
The Compose file pins the verified Splunk 10.4.2 image digest for repeatability.

## Start

The operator must review and accept the applicable Splunk terms before running
the setup script. The local acceptance flag is stored in the ignored `.env`
file.

```bash
cd deployments/splunk
chmod +x setup.sh wait-until-ready.sh create-test-index.sh configure-hec.sh configure-observer-role.sh smoke-test.sh verify-latest-windows-event.sh
./setup.sh
./wait-until-ready.sh
./create-test-index.sh
./configure-hec.sh
./configure-observer-role.sh
./smoke-test.sh
```

Open <http://localhost:18000> and sign in as `admin`. The generated local
password is stored in `deployments/splunk/.env` and must not be committed.

## Endpoints

| Service | Local endpoint |
|---|---|
| Splunk Web | `http://127.0.0.1:18000` |
| HTTP Event Collector | `https://127.0.0.1:8088` |
| Management API | `https://127.0.0.1:8089` |

All published ports bind to loopback only. The test index is
`prooflayer_test`, capped at 512 MB for this lab.

The smoke test sends synthetic text through Splunk's authenticated management
receiver, then verifies that a Splunk search returns the event. It does not use
customer data.

`configure-hec.sh` creates a dedicated `prooflayer_lab` HEC input restricted to
the `prooflayer_test` index and stores its generated token only in the ignored
local `.env` file.

After the Windows Day 6 script runs, verify the latest canonical Sysmon event
and write ignored local evidence with:

```bash
./verify-latest-windows-event.sh
```

After three runs, verify unique IDs, required parsed fields, and the ingestion
latency distribution:

```bash
./verify-windows-event-series.sh
```

Verify one exact canonical correlation ID with a fixed 24-hour local PoC window
and exactly-one-match requirement:

```bash
./verify-windows-event-by-id.sh PL-<32-uppercase-hex-characters>
```

The verifier returns field names and counts, not command-line or username
values.

## Stop and resume

```bash
docker compose stop
docker compose start
```

The named volumes preserve the local configuration and indexed test data. Never
edit Splunk's internal password database in place. If local authentication state
becomes unrecoverable, preserve the existing volumes and rotate to new,
explicitly named lab volumes before recreating synthetic test data.
