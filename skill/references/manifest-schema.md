# Devlaunch Manifest Reference

- Top-level keys: `version`, `project`, `terminal`, `apps`, `services`
- Only `apps` and `services` may describe runtime resources
- `docker compose up -d` must be modeled as a `service`
