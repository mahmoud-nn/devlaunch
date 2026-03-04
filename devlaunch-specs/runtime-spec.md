# Runtime Spec

## Rôle

Le runtime exécute les vrais flux:

- `validate`
- `start`
- `stop`
- `status`
- `register`
- `reconcile`

## Contrat de validation

Avant toute action métier:

1. lire `manifest.json`
2. valider strictement contre `manifest.v1.schema.json`
3. lire `state.local.json` si présent
4. valider strictement contre `state.v1.schema.json`
5. seulement ensuite continuer

Si le manifest est invalide:

- l'action échoue

Si le state est invalide:

- il est ignoré
- un state vide valide est reconstruit en mémoire
- un warning est renvoyé

## `start`

Ordre:

1. valider les fichiers
2. réconcilier le state avec la réalité système
3. résoudre `startPolicy`
4. lancer les `apps`
5. attendre `checks.start`
6. lancer les `services` dans l'ordre topologique
7. attendre `checks.start`
8. mettre à jour state + registry

## `stop`

Ordre:

1. valider les fichiers
2. réconcilier le state avec la réalité système
3. arrêter les `services` selon `stopPolicy`
4. demander explicitement quoi faire des `apps` selon `stopPolicy`
5. mettre à jour state + registry

## `status`

Le runtime doit renvoyer:

- état calculé des ressources
- état mémorisé dans le state
- divergence éventuelle
- mécanisme d'observation utilisé
- warnings éventuels

## Observation réelle

Le runtime ne doit pas déduire l'état d'une ressource uniquement depuis le state.

Il doit observer la réalité avec:

- `checks.status`
- le PID mémorisé si encore crédible

`fixed-delay` peut servir au démarrage mais ne peut pas suffire à conclure `running` en `status`.

## API locale minimale

- `GET /projects`
- `GET /projects/:id/status`
- `POST /projects/:id/start`
- `POST /projects/:id/stop`
- `POST /projects/:id/open-folder`

Les appels `start` et `stop` acceptent des options d'exécution.

Par défaut la web UI utilise le mode non interactif et peut fournir plus tard des décisions explicites pour les ressources en `ask`.

## Principes

- le runtime ne dépend pas de la web UI
- la CLI et la web UI appellent le même runtime
- la différence CLI/UI passe uniquement par les options d'exécution
