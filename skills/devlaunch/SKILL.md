---
name: devlaunch
description: Configure a repository for devlaunch by generating manifest and state files that match the runtime contract.
---

# Devlaunch Skill

Use this skill when a repository must be initialized for `devlaunch`.

## Workflow

Always start with the developer, not with repo guessing.

Order is mandatory:

1. Ask the developer which tools they use to work on the project.
2. Ask how they usually launch the project day to day.
3. Ask which tools should stay open or be treated as optional.
4. Only after that, inspect the repository to confirm and complete the picture.
5. If the repo is not initialized yet, run `devlaunch init` first to create the baseline files.
6. Then adapt `.devlaunch/manifest.json` and `.devlaunch/state.local.json`.
7. Never generate custom config first and run `devlaunch init` after.
8. Register the project through `devlaunch init` or the runtime.

## Questions To Ask First

Ask explicitly:

1. Which developer tools do you need open for this project?
2. Which commands do you usually run, and in what order?
3. Which commands are interactive long-running services?
4. Which external apps are required on your machine?
5. Which items should be optional rather than auto-started?

Do not skip these questions unless the developer already answered them in the current conversation.

## Then Inspect The Repo

After the developer answers:

1. Inspect the repository.
2. Detect likely local developer commands and external apps.
3. Compare repo evidence with the developer's answers.
4. If there is a mismatch, prefer asking a follow-up question rather than guessing.

## Use The Local Skill Assets

This skill includes local supporting files that must be used explicitly.

Use them with this intent:

1. `assets/templates/`
   Use these as the generation base for new config files.

2. `assets/examples/`
   Use these to understand what a realistic final manifest or state file should look like.

3. `references/`
   Use these to confirm schema expectations and runtime constraints.

Do not rely only on memory when these files are available.
Check the local assets before generating the final output.

## Goals

1. Capture the real developer workflow first.
2. Confirm it against the repository.
3. Run `devlaunch init` first when `.devlaunch` is missing.
4. Adapt `.devlaunch/manifest.json`.
5. Adapt `.devlaunch/state.local.json`.
6. Register the project through `devlaunch init` or the runtime.

## Constraints

- The manifest may only contain `apps` and `services`.
- `docker compose up -d` must be modeled as a `service`.
- The target platform is Windows.
- The shell is PowerShell.
- Keep the manifest aligned with the runtime schema.
- Prefer explicit developer answers over inference when defining startup flow.
- `devlaunch init` must be treated as safe on existing projects and must not silently destroy existing `.devlaunch` files.
