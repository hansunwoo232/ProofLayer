# Dashboard

The dashboard will let operators start tests, inspect history and host state,
and view the complete pipeline:

`Generated → Collected → Ingested → Parsed → Detected → Alerted`

## Day 26 result-screen wireframe

`result-screen-wireframe.html` is a single-file, local-only product wireframe.
It presents one synthetic parser-failure run across all seven product stages,
including mandatory cleanup. PASS, FAIL, and NOT TESTED states use text and
symbols in addition to color. Raw endpoint and customer event bodies are not
embedded or displayed.

Open the file directly in a browser. It has no external runtime, font, image,
script, network, analytics, or persistence dependency.
