# Runtime Spec

## Rôle

Le runtime exécute les vrais flux:

- `start`
- `stop`
- `status`
- `register`
- `reconcile`

## Implémentation

Le runtime doit être implémenté en `Go`.

Le même binaire Go peut:

- exécuter les commandes CLI
- héberger la web UI locale
- exposer l'API HTTP locale

## Responsabilités

### `start`

Ordre:

1. lire `manifest.json`
2. lire `state.local.json`
3. réconcilier
4. lancer les `apps`
5. attendre la readiness des `apps`
6. lancer les `services` dans l'ordre topologique
7. ouvrir Windows Terminal pour les services interactifs
8. mettre à jour state + registry

### `stop`

Ordre:

1. lire manifest + state
2. réconcilier
3. arrêter les `services`
4. demander explicitement quoi faire des `apps`
5. mettre à jour state + registry

### `status`

Le runtime doit renvoyer:

- état calculé des ressources
- divergence éventuelle avec le state
- dernier démarrage connu

### `register`

Le runtime ajoute ou met à jour le projet dans le registry global.

### `reconcile`

Le runtime doit supporter:

- state absent
- state invalide
- tabs fermés à la main
- arrêt brutal machine

## Readiness v1

Supporter:

- `command`
- `port`
- `process`
- `fixed-delay`

## API locale minimale

- `GET /projects`
- `GET /projects/:id/status`
- `POST /projects/:id/start`
- `POST /projects/:id/stop`
- `POST /projects/:id/open-folder`
- `POST /projects/register`

Cette API est locale, sur `localhost`, et sert surtout la web UI.

## Process interactifs

Un `service` interactif:

- est lancé dans Windows Terminal
- avec un nom d'onglet
- dans le bon répertoire

Un `service` non interactif:

- peut être lancé sans conserver un onglet

## Principes

- le runtime ne doit pas dépendre de la web UI
- la CLI et la web UI utilisent le même runtime
- la web UI est une couche de confort; la CLI doit exposer les mêmes actions métier
