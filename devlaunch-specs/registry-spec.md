# Registry Spec

## Principe

Le registry global référence les projets déjà configurés.

Il ne stocke pas la config complète du projet.

Le registry n'est pas un namespace CLI public.
La CLI expose ces données via le namespace `project`.

## Emplacement

Sous `%USERPROFILE%`, dans un dossier global `devlaunch`.

Exemple:

```text
%USERPROFILE%\.devlaunch\
  registry\
  logs\
  runtime\
```

## Contenu minimal

```json
{
  "version": 1,
  "projects": [
    {
      "id": "lebonplan",
      "name": "lebonplan",
      "rootPath": "C:\\PROJETS\\lebonplan",
      "manifestPath": "C:\\PROJETS\\lebonplan\\.devlaunch\\manifest.json",
      "lastSeenAt": "2026-03-04T08:10:00.000Z",
      "lastKnownStatus": "running"
    }
  ]
}
```

## Champs

- `id`
- `name`
- `rootPath`
- `manifestPath`
- `lastSeenAt`
- `lastKnownStatus`

## Alimentation

Le registry est mis à jour par:

- l'assistant `init`
- le runtime lors de `start`
- le runtime lors de `stop`
- le runtime lors de `status`

## Règles

- un projet est identifié par `rootPath`
- `id` doit être stable
- pas de scan automatique global obligatoire en v1
- pas d'édition manuelle UI en v1
