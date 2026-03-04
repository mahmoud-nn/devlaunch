# Devlaunch Handoff

## Objectif

Créer un projet séparé `C:\PROJETS\devlaunch` avec dépôt git:

- `https://github.com/mahmoud-nn/devlaunch.git`

La commande principale doit s'appeler:

- `devlaunch`

`lebonplan` sert seulement de terrain de test pour le manifest et les flux de lancement.

## Décisions verrouillées

### Structure générale

Le projet `devlaunch` contient dans le même repo:

- le runtime
- la web UI
- la CLI custom
- le skill source

Mais conceptuellement:

- le skill reste distinct du runtime/web UI
- le runtime fait le vrai travail
- la web UI et la CLI appellent le runtime

### Stack technique

- backend/runtime: `Go`
- CLI: `Go`
- moteur CLI: `Cobra`
- web UI locale: servie par `Go`
- scripts système: `PowerShell`
- terminal: `Windows Terminal`
- plateforme v1: `Windows`

La web UI locale peut être une interface SSR simple avec Tailwind CSS.

### Arborescence logique attendue

Projet dédié:

- `C:\PROJETS\devlaunch`

Manifest projet:

- `<repo>\\.devlaunch\\manifest.json`

State local projet:

- `<repo>\\.devlaunch\\state.local.json`

Registry global:

- sous `%USERPROFILE%`

### Modèle manifest

Le manifest ne contient que:

- `apps`
- `services`

Règles:

- `apps` = tout ce qui est externe à la machine/projet
- `services` = tout ce qui est propre au projet

Exemples:

- `Docker Desktop` = `app`
- `Laragon` = `app`
- `Cursor` = `app`
- `Android Studio` = `app`
- `docker compose up -d` = `service`
- `pnpm dev:users` = `service`
- `bun dev:full` = `service`

Relations:

- une `app` peut dépendre d'une `app`
- un `service` peut dépendre d'une `app`
- un `service` peut dépendre d'un `service`

### Runtime attendu

Le runtime doit savoir:

- lire `manifest.json`
- lire `state.local.json`
- réconcilier le state avec la réalité système
- lancer les `apps`
- attendre leur readiness réelle
- lancer les `services` dans l'ordre
- ouvrir des onglets Windows Terminal pour les services interactifs
- lancer les services non interactifs sans onglet persistant si nécessaire
- stopper les services
- demander explicitement s'il faut aussi stopper les `apps`

### Web UI

La web UI doit être très simple:

- grille de projets
- nom du projet
- chemin du projet
- statut
- bouton `Start`
- bouton `Stop`
- bouton `Open Folder`

La web UI appelle seulement le runtime.

Règles:

- elle est servie localement par le même binaire Go
- elle reste optionnelle
- toute action UI doit aussi exister en CLI

### CLI

La CLI globale doit s'appeler:

- `devlaunch`

Commandes attendues:

- `devlaunch ui`
- `devlaunch init`
- `devlaunch project list`
- `devlaunch project start [project-id]`
- `devlaunch project stop [project-id]`
- `devlaunch project status [project-id]`
- `devlaunch project open [project-id]`
- `devlaunch skill install`

Règle de résolution:

- si `project-id` est fourni, la commande cible un projet du registry
- sinon, elle cible le projet du dossier courant

### Skill

Le skill source doit vivre dans le même repo `devlaunch`.

Le but de `devlaunch skill install`:

1. prendre la version du skill embarquée dans le repo `devlaunch`
2. la copier dans le bon dossier utilisateur
3. appeler `npx skills` pour installer ce skill globalement côté agents

Important:

- on ne réinvente pas un installateur de skill séparé
- on proxy `npx skills`

### Assistant de configuration

La CLI doit aussi permettre de lancer un assistant de configuration dans le dossier courant pour:

- analyser le projet
- poser des questions
- générer `.devlaunch/manifest.json`
- initialiser `.devlaunch/state.local.json`
- enregistrer le projet dans le registry global

L'assistant peut s'appuyer sur un agent supporté.

## Exemple de cas cible: lebonplan

Pour `C:\PROJETS\lebonplan`, le manifest cible doit au minimum pouvoir représenter:

### Apps

- `docker-desktop`

### Services

- `docker-compose`
- `users-service`
- `payments-service`
- `logistics-service`
- `notifications-service`
- `announcements-service`
- `escrow-sales-service`
- `admin-app`
- `frontend`

Ordre logique:

1. vérifier/lancer Docker Desktop
2. attendre readiness Docker
3. lancer `docker compose up -d`
4. lancer les `pnpm dev:{service}`
5. lancer le frontend

Stop logique:

1. arrêter les services projet
2. demander explicitement:
   - faut-il faire `docker compose down` ?
   - faut-il laisser Docker Desktop ouvert ?

## Ce qui a déjà été fait

Un début de structure a été créé dans:

- `C:\PROJETS\devlaunch`

Dossiers présents au moment du handoff:

- `apps/web/src`
- `packages/cli/src`
- `packages/runtime/src/core`
- `packages/runtime/src/server`
- `packages/runtime/src/types`
- `packages/skill/scripts`
- `packages/skill/references`
- `packages/skill/assets/templates`
- `scripts/ps1`
- `data`

Mais le scaffold complet n'a pas été écrit proprement, car la session courante n'a pas `C:\PROJETS\devlaunch` comme dossier autorisé en écriture normale.

## Pourquoi changer de session

La session actuelle est sandboxée sur:

- `C:\PROJETS\lebonplan`

Donc travailler dans `C:\PROJETS\devlaunch` oblige à passer par des commandes PowerShell escaladées, ce qui ralentit beaucoup et rend les créations de fichiers fragiles.

La bonne suite est:

1. ouvrir une nouvelle session directement dans `C:\PROJETS\devlaunch`
2. continuer l'implémentation depuis là

## Plan d'exécution recommandé pour le nouvel agent

1. Initialiser proprement le repo `C:\PROJETS\devlaunch`
2. Créer le monorepo interne:
   - `apps/web`
   - `packages/runtime`
   - `packages/cli`
   - `packages/skill`
   - `scripts/ps1`
3. Écrire le socle workspace:
   - `package.json`
   - `pnpm-workspace.yaml`
   - `tsconfig`
   - `.gitignore`
4. Implémenter `@devlaunch/runtime` avec:
   - types `manifest/state/registry`
   - lecture/écriture fichiers
   - réconciliation state
   - API HTTP locale Go
5. Implémenter la CLI Go avec:
   - `ui`
   - `init`
   - `project list`
   - `project start`
   - `project stop`
   - `project status`
   - `project open`
   - `skill install`
6. Implémenter les scripts PowerShell globaux
7. Implémenter la web UI locale simple servie par Go
8. Intégrer le skill dans `packages/skill`
9. Faire fonctionner `devlaunch skill install`
10. Tester avec `lebonplan`

## Contraintes à respecter

- rester succinct dans les explications
- ne pas réintroduire un modèle `external/resources`; seulement `apps/services`
- `docker compose` doit rester un `service`
- le manifest doit s'appeler exactement `manifest.json`
- le state doit s'appeler exactement `state.local.json`
- la commande globale doit s'appeler exactement `devlaunch`
- le namespace public des actions projet doit être `project`
