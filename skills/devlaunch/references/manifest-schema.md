# Devlaunch Manifest Reference

Normative source:

- `references/manifest.v1.schema.json`

Rules:

- `version` must be `1`
- top-level keys are only `version`, `project`, `terminal`, `apps`, `services`
- every `app` and `service` must declare:
  - `startPolicy`
  - `stopPolicy`
  - `checks.start`
  - `checks.status`
- every check group must declare `mode`
- unsupported keys are invalid
- missing required keys are invalid

This Markdown is a human guide only. The JSON schema is the strict contract.
