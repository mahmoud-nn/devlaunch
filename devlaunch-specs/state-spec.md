# State Spec

## Chemin

- `<repo>\.devlaunch\state.local.json`

## Principe

Le state local mémorise ce que le runtime croit avoir lancé.

Ce n'est jamais une source de vérité forte.

Il doit rester:

- local
- non versionné
- tolérant aux corruptions et extinctions sauvages

## Schéma conceptuel

```json
{
  "version": 1,
  "projectName": "lebonplan",
  "lastStartAt": "2026-03-04T08:00:00.000Z",
  "lastStopAt": null,
  "resources": {
    "docker-desktop": {
      "type": "app",
      "status": "running",
      "startedByRuntime": true,
      "lastKnownPid": 12345
    },
    "docker-compose": {
      "type": "service",
      "status": "running",
      "startedByRuntime": true
    },
    "payments-service": {
      "type": "service",
      "status": "running",
      "startedByRuntime": true,
      "terminalTabName": "payments-service"
    }
  }
}
```

## Champs

- `version`
- `projectName`
- `lastStartAt`
- `lastStopAt`
- `resources`

## Ressource

Champs utiles v1:

- `type`
- `status`
- `startedByRuntime`
- `lastKnownPid`
- `terminalTabName`
- `lastSeenAt`

## Statuts v1

- `running`
- `stopped`
- `unknown`
- `failed`

## Règles

- fichier absent: autorisé
- JSON invalide: autorisé, le runtime doit survivre
- info partielle: autorisée
- le runtime doit réconcilier avec la réalité avant `start`, `stop`, `status`

## Réconciliation

La réconciliation doit:

- vérifier si un process existe encore
- vérifier si une app est réellement disponible
- vérifier si un service non interactif a réellement été lancé
- nettoyer les entrées mortes
- conserver ce qui reste crédible
