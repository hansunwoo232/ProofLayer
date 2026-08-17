# Dashboard

The dashboard will let operators start tests, inspect history and host state,
and view the complete pipeline:

`Generated → Collected → Ingested → Parsed → Detected → Alerted`

## Day 27 local result surface

`result-screen-wireframe.html` is a single-file, local-only product wireframe.
It presents one synthetic parser-failure run across all seven product stages,
including mandatory cleanup. PASS, FAIL, and NOT TESTED states use text and
symbols in addition to color. Raw endpoint and customer event bodies are not
embedded or displayed.

Opening the file directly still shows the representative result without an
external runtime, font, image, analytics, or persistence dependency. The Run
Test button stays unavailable in file mode.

When served by the Day 27 local Control Plane at `http://127.0.0.1:8787`, the
button obtains a same-origin CSRF session and queues the fixed allowlisted
process-marker job. It disables synchronously and reuses one idempotency key if
the request must be retried. Queueing does not yet execute the Runner or replace
the representative pipeline result.
