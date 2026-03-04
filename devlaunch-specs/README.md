# Devlaunch Specs

## But

Ce dossier décrit le projet `C:\PROJETS\devlaunch`.

Le prochain agent doit travailler dans ce repo séparé, pas dans `lebonplan`.

`lebonplan` sert seulement de projet de test pour valider `devlaunch`.

## Résumé

`devlaunch` est un projet unique qui contient:

- le runtime
- la web UI
- la CLI
- le skill source

Conceptuellement:

- le skill aide à configurer un projet
- la CLI et la web UI pilotent le runtime
- le runtime exécute réellement les lancements et arrêts

## Contraintes verrouillées

- commande globale: `devlaunch`
- backend/runtime: `Go`
- moteur CLI: `Cobra`
- OS cible v1: `Windows`
- shell cible v1: `PowerShell`
- terminal cible v1: `Windows Terminal`
- manifest projet: `<repo>\.devlaunch\manifest.json`
- state projet: `<repo>\.devlaunch\state.local.json`
- seulement `apps` et `services` dans le manifest
- `docker compose up -d` est un `service`, pas une `app`
- le skill vit dans le même repo `devlaunch`
- `devlaunch skill install` copie le skill local puis appelle `npx skills`
- la web UI locale est servie par le binaire Go
- le namespace public pour les actions projet est `project`

## Docs de ce dossier

- [architecture.md](/C:/PROJETS/devlaunch/devlaunch-specs/architecture.md)
- [manifest-spec.md](/C:/PROJETS/devlaunch/devlaunch-specs/manifest-spec.md)
- [state-spec.md](/C:/PROJETS/devlaunch/devlaunch-specs/state-spec.md)
- [registry-spec.md](/C:/PROJETS/devlaunch/devlaunch-specs/registry-spec.md)
- [runtime-spec.md](/C:/PROJETS/devlaunch/devlaunch-specs/runtime-spec.md)
- [cli-spec.md](/C:/PROJETS/devlaunch/devlaunch-specs/cli-spec.md)
- [skill-spec.md](/C:/PROJETS/devlaunch/devlaunch-specs/skill-spec.md)
- [web-ui-spec.md](/C:/PROJETS/devlaunch/devlaunch-specs/web-ui-spec.md)
- [lebonplan-example.md](/C:/PROJETS/devlaunch/devlaunch-specs/lebonplan-example.md)
