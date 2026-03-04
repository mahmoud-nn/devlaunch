# Lebonplan Example

## Rôle

`lebonplan` est le projet de test principal pour valider `devlaunch`.

## Manifest cible

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

## Ordre de start

1. vérifier Docker Desktop
2. lancer Docker Desktop si besoin
3. attendre `docker info`
4. lancer `docker compose up -d`
5. lancer les services `pnpm dev:{service}`
6. lancer le frontend

## Ordre de stop

1. arrêter les services projet
2. demander explicitement:
   - faut-il faire `docker compose down` ?
   - faut-il laisser Docker Desktop ouvert ?

## Règles importantes

- `docker compose` reste un `service`
- le manifest de `lebonplan` ne doit pas introduire d'autre concept que `apps/services`
- le state doit tolérer les fermetures brutales de terminaux
