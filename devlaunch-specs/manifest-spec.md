# Manifest Spec

## Chemin

- `<repo>\.devlaunch\manifest.json`

## Principe

Le manifest v1 décrit le contrat strict de lancement du projet.

Il ne contient que:

- `apps`
- `services`

La référence normative est:

- `skills/devlaunch/references/manifest.v1.schema.json`

## Top-level

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
  "apps": [],
  "services": []
}
```

## `app`

```json
{
  "id": "docker-desktop",
  "kind": "desktop-app",
  "dependsOn": [],
  "launch": {
    "strategy": "executable",
    "path": "C:\\Program Files\\Docker\\Docker\\Docker Desktop.exe"
  },
  "startPolicy": {
    "defaultAction": "always"
  },
  "stopPolicy": {
    "defaultAction": "ask"
  },
  "checks": {
    "start": {
      "mode": "all",
      "items": [
        {
          "type": "command",
          "command": "docker info",
          "retry": {
            "maxAttempts": 60,
            "delayMs": 2000
          }
        }
      ]
    },
    "status": {
      "mode": "all",
      "items": [
        {
          "type": "command",
          "command": "docker info"
        }
      ]
    }
  }
}
```

## `service`

```json
{
  "id": "frontend",
  "kind": "project-command",
  "interactive": true,
  "tabName": "frontend",
  "workingDirectory": "C:\\PROJETS\\demo",
  "command": "pnpm dev",
  "dependsOn": ["docker-compose"],
  "startPolicy": {
    "defaultAction": "ask"
  },
  "stopPolicy": {
    "defaultAction": "always"
  },
  "checks": {
    "start": {
      "mode": "all",
      "items": [
        {
          "type": "fixed-delay",
          "delayMs": 2000
        }
      ]
    },
    "status": {
      "mode": "any",
      "items": [
        {
          "type": "process",
          "name": "node"
        }
      ]
    }
  }
}
```

## Policies v1

- `startPolicy.defaultAction`
- `stopPolicy.defaultAction`

Valeurs supportées:

- `always`
- `ask`
- `never`

## Checks v1

Chaque groupe de checks doit déclarer:

- `mode`
- `items`

Modes supportés:

- `all`
- `any`

Types de check supportés:

- `command`
- `port`
- `process`
- `http`
- `fixed-delay`

## Règles de validation métier

- `id` unique globalement dans le manifest
- `dependsOn` doit référencer des ids existants
- pas de cycle de dépendance
- une `app` peut dépendre uniquement d'une `app`
- un `service` peut dépendre d'une `app` ou d'un `service`
- un `service` interactif doit avoir un `tabName`
- `checks.status` ne peut pas être composé uniquement de `fixed-delay`
- aucun champ hors schéma n'est supporté
