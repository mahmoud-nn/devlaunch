# Skill Spec

## Principe

Le skill vit dans le même repo `devlaunch`.

Il n'est pas un projet séparé.

Mais conceptuellement, il reste distinct:

- il aide à configurer
- il n'exécute pas tout le runtime applicatif

## Rôle

Le skill doit permettre à un agent de:

- analyser un repo
- découvrir comment il se lance
- détecter les outils externes
- poser les bonnes questions
- générer le manifest
- initialiser le state
- enregistrer le projet

## Flux attendu

1. lire le repo courant
2. détecter:
   - `package.json`
   - scripts dev
   - compose
   - outils machine probables
3. poser les questions nécessaires
4. produire:
   - `.devlaunch/manifest.json`
   - `.devlaunch/state.local.json`
5. appeler le runtime ou la CLI pour enregistrer le projet

## Installation

Le skill n'est pas installé manuellement.

La commande cible est:

- `devlaunch skill install`

Elle doit:

1. copier le skill source depuis le repo `devlaunch`
2. placer cette version dans le bon dossier utilisateur
3. appeler `npx skills`

Le fait que la CLI et le runtime soient en Go ne change pas cette règle.

## Contenu minimal du skill

- `SKILL.md`
- références de schéma
- templates de manifest
- templates de state

## Contraintes

- rester aligné avec le runtime réel
- ne pas inventer une config différente du manifest réel
