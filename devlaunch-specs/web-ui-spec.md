# Web UI Spec

## But

Fournir une interface locale simple pour piloter les projets.

## Portée v1

La web UI doit être minimale.

Affichage:

- grille de projets
- nom
- chemin
- statut

Actions:

- `Start`
- `Stop`
- `Open Folder`
- `Refresh`

## Source de données

La web UI lit le registry global via le runtime.
Elle est servie localement par le runtime Go.

## Règles

- `localhost` uniquement en v1
- pas d'auth en v1
- la web UI ne réimplémente pas `start/stop/status`
- elle appelle seulement l'API du runtime
- toutes les actions UI doivent aussi exister en CLI
- un rendu SSR HTML simple est suffisant en v1
- Tailwind CSS peut être utilisé pour styliser l'interface

## UX

Le rendu peut être simple:

```text
+-------------------------------------------------------------------+
| devlaunch                                                         |
+-------------------------------------------------------------------+
| lebonplan        C:\PROJETS\lebonplan        running   [Start][Stop]
| other-project    C:\PROJETS\other-project    stopped   [Start][Stop]
+-------------------------------------------------------------------+
```
