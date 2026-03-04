# Manifest Spec

## Chemin

Chaque projet configuré possède:

- `<repo>\.devlaunch\manifest.json`

## Principe

Le manifest décrit comment lancer un projet.

Il ne contient que deux collections:

- `apps`
- `services`

## Définition

### `apps`

Ressources externes au projet.

Exemples:

- Docker Desktop
- Laragon
- Cursor
- IntelliJ
- Android Studio

### `services`

Commandes propres au projet.

Exemples:

- `docker compose up -d`
- `pnpm dev:payments`
- `bun dev:full`
- worker
- backend
- frontend

## Dépendances

Règles:

- une `app` peut dépendre d'une `app`
- un `service` peut dépendre d'une `app`
- un `service` peut dépendre d'un `service`

## Schéma conceptuel

```json
{
  "version": 1,
  "project": {
    "name": "lebonplan",
    "rootPath": "C:\\PROJETS\\lebonplan",
    "platform": "windows"
  },
  "terminal": {
    "engine": "windows-terminal",
    "reuseWindow": false
  },
  "apps": [],
  "services": []
}
```

## Schéma `app`

```json
{
  "id": "docker-desktop",
  "kind": "desktop-app",
  "enabled": true,
  "dependsOn": [],
  "launch": {
    "strategy": "executable",
    "path": "C:\\Program Files\\Docker\\Docker\\Docker Desktop.exe"
  },
  "readiness": {
    "type": "command",
    "command": "docker info",
    "retry": {
      "maxAttempts": 60,
      "delayMs": 2000
    }
  },
  "stopPolicy": {
    "defaultAction": "ask"
  }
}
```

## Schéma `service`

```json
{
  "id": "docker-compose",
  "kind": "project-command",
  "interactive": false,
  "tabName": null,
  "workingDirectory": "C:\\PROJETS\\lebonplan",
  "command": "docker compose up -d",
  "dependsOn": ["docker-desktop"],
  "readiness": {
    "type": "command",
    "command": "docker compose ps"
  },
  "startWhen": {
    "delayMs": 0
  },
  "stopPolicy": {
    "defaultAction": "ask",
    "command": "docker compose down"
  }
}
```

## Champs attendus

### `project`

- `name`
- `rootPath`
- `platform`

### `terminal`

- `engine`
- `reuseWindow`

### `apps[*]`

- `id`
- `kind`
- `enabled`
- `dependsOn`
- `launch`
- `readiness`
- `stopPolicy`

### `services[*]`

- `id`
- `kind`
- `interactive`
- `tabName`
- `workingDirectory`
- `command`
- `dependsOn`
- `readiness`
- `startWhen`
- `stopPolicy`

## Règles de validation

- `id` unique globalement dans le manifest
- `dependsOn` doit référencer des ids existants
- pas de cycle de dépendance
- `docker compose` reste un `service`
- un `service` interactif a normalement un `tabName`
- un `service` non interactif peut avoir `tabName = null`

## Types de readiness v1

- `command`
- `port`
- `process`
- `fixed-delay`

## Types de stop policy v1

- `ask`
- `never`
- `always`

## Exemple minimal

```json
{
  "version": 1,
  "project": {
    "name": "demo",
    "rootPath": "C:\\PROJETS\\demo",
    "platform": "windows"
  },
  "terminal": {
    "engine": "windows-terminal",
    "reuseWindow": false
  },
  "apps": [
    {
      "id": "docker-desktop",
      "kind": "desktop-app",
      "enabled": true,
      "dependsOn": [],
      "launch": {
        "strategy": "executable",
        "path": "C:\\Program Files\\Docker\\Docker\\Docker Desktop.exe"
      },
      "readiness": {
        "type": "command",
        "command": "docker info",
        "retry": {
          "maxAttempts": 60,
          "delayMs": 2000
        }
      },
      "stopPolicy": {
        "defaultAction": "ask"
      }
    }
  ],
  "services": [
    {
      "id": "docker-compose",
      "kind": "project-command",
      "interactive": false,
      "tabName": null,
      "workingDirectory": "C:\\PROJETS\\demo",
      "command": "docker compose up -d",
      "dependsOn": ["docker-desktop"],
      "readiness": {
        "type": "command",
        "command": "docker compose ps"
      },
      "startWhen": {
        "delayMs": 0
      },
      "stopPolicy": {
        "defaultAction": "ask",
        "command": "docker compose down"
      }
    }
  ]
}
```
