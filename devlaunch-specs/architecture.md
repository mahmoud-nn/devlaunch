# Architecture

## Projet cible

- `C:\PROJETS\devlaunch`

## Structure logique

Le repo contient 4 briques:

1. `runtime`
Le moteur d'orchestration.

2. `web UI`
Une interface locale simple servie par Go qui appelle le runtime.

3. `CLI`
La commande globale `devlaunch`, implémentée en Go avec Cobra.

4. `skill`
Le skill source, stocké dans le même repo.

## Séparation des responsabilités

### Runtime

Responsable de:

- lire les manifests
- lire les states
- réconcilier l'état avec la réalité système
- lancer les `apps`
- attendre la readiness
- lancer les `services`
- stopper les `services`
- demander quoi faire des `apps`
- maintenir le registry global
- exposer l'API HTTP locale utilisée par la web UI

### Web UI

Responsable de:

- lister les projets enregistrés
- afficher un statut
- appeler `start`
- appeler `stop`
- ouvrir le dossier projet

La web UI ne réimplémente pas l'orchestration.
Elle est un convenience layer au-dessus du runtime.

### CLI

Responsable de:

- exposer les commandes utilisateur
- appeler le runtime
- lancer l'assistant de configuration
- gérer l'installation du skill
- exposer les mêmes actions métier que la web UI

### Skill

Responsable de:

- analyser un projet
- poser des questions
- détecter les outils système
- générer le manifest
- initialiser le state
- enregistrer le projet dans le registry

## Structure repo recommandée

```text
C:\PROJETS\devlaunch
├── cmd/
│   └── devlaunch/
├── internal/
│   ├── app/
│   ├── manifest/
│   ├── registry/
│   ├── runtime/
│   ├── state/
│   ├── web/
│   └── windows/
├── packages/
│   └── skill/
├── scripts/
│   └── ps1/
└── data/
```

## Principes

- le runtime est la source de vérité d'exécution
- le manifest est la source de vérité de configuration projet
- le state est local et non autoritaire
- le registry global référence les projets déjà configurés

## V1

V1 vise:

- Windows seulement
- PowerShell seulement
- Windows Terminal seulement
- localhost seulement pour la web UI
- runtime, CLI et serveur web local dans le même binaire Go
