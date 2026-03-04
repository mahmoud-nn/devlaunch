# Devlaunch State Reference

Normative source:

- `references/state.v1.schema.json`

Rules:

- `version` must be `1`
- top-level keys are only `version`, `projectName`, `lastStartAt`, `lastStopAt`, `resources`
- resource statuses are only `running`, `stopped`, `unknown`, `failed`
- unsupported keys are invalid
- missing required keys are invalid

The state is local and recoverable, but when a state file exists it must still match the strict schema to be trusted.
