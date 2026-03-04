# CLI Spec

## Commande

La commande globale doit s'appeler:

- `devlaunch`

## Implémentation

La CLI doit être implémentée en `Go` avec `Cobra`.

## Structure v1

Les actions projet doivent utiliser le namespace public:

- `project`

Le mot `registry` reste un détail interne d'implémentation.

## Commandes v1

### `devlaunch ui`

Lance la web UI locale.

Modes utiles:

- attaché
- détaché

### `devlaunch init`

Lance l'assistant de configuration dans le dossier courant.

But:

- analyser le projet
- poser les questions
- générer `.devlaunch/manifest.json`
- générer `.devlaunch/state.local.json`
- enregistrer le projet

### `devlaunch project list`

Liste les projets connus.

### `devlaunch project start [project-id]`

Lance le projet courant ou un projet connu via son `project-id`.

### `devlaunch project stop [project-id]`

Stoppe le projet courant ou un projet connu via son `project-id`.

### `devlaunch project status [project-id]`

Affiche le statut du projet courant ou d'un projet connu via son `project-id`.

### `devlaunch project open [project-id]`

Ouvre le dossier du projet courant ou d'un projet connu via son `project-id`.

### `devlaunch skill install`

But:

1. prendre le skill embarqué dans le repo `devlaunch`
2. le copier dans le bon dossier utilisateur
3. appeler `npx skills` pour l'installer

## Règles CLI

- une commande doit pouvoir fonctionner depuis un repo projet
- une commande doit aussi pouvoir cibler un projet du registry
- la CLI doit rester un proxy fin vers le runtime
- la logique d'orchestration doit rester dans le runtime
- la web UI ne doit exposer aucune action absente de la CLI
- si `project-id` est absent, la commande cible le dossier courant
