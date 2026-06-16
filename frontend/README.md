# MailHive — Frontend

Interface web de gestion et monitoring de la plateforme MailHive, construite avec React 19, TypeScript et Tailwind CSS v4.

---

## Stack

| Outil | Version | Rôle |
| ------- | --------- | ------ |
| React | 19 | Framework UI |
| TypeScript | 5.9 | Typage statique |
| Vite | 7 | Bundler et serveur de dev |
| Tailwind CSS | v4 | Styles utilitaires |
| React Router | v7 | Routage SPA |
| Axios | 1.x | Client HTTP |
| Recharts | 3 | Graphiques interactifs |

---

## Structure

```txt
src/
├── api/
│   └── client.ts            # Instance Axios + types API
├── contexts/
│   └── AuthContext.tsx       # Contexte d'authentification (JWT)
├── components/
│   ├── Layout.tsx            # Barre latérale + contenu principal
│   └── ProtectedRoute.tsx    # Garde d'authentification
├── pages/
│   ├── LoginPage.tsx         # Connexion par clé API
│   ├── DashboardPage.tsx     # Tableau de bord + monitoring queues
│   ├── MailsPage.tsx         # Liste des mails (pagination, filtres)
│   ├── MailDetailPage.tsx    # Détail d'un mail
│   ├── TemplatesPage.tsx     # Gestion des templates
│   ├── SMTPConfigsPage.tsx   # Gestion des configs SMTP
│   └── TenantsPage.tsx       # Gestion des tenants (admin)
├── App.tsx                   # Routeur principal
└── main.tsx                  # Point d'entrée
```

---

## Pages

### Connexion (`/login`)

Formulaire de saisie de la clé API du tenant. Le token JWT retourné est stocké dans `localStorage`.

### Tableau de bord (`/`)

- **Cartes de résumé** : total mails, envoyés, échoués, nombre de tenants
- **Graphique en barres** : répartition par statut (envoyés, en attente, en cours, échoués, annulés)
- **Graphique circulaire** : vue d'ensemble avec taux de succès
- **Stats par tenant** : barres empilées + répartition circulaire (admin)
- **Monitoring queues Asynq** : tableau en temps réel (rafraîchi toutes les 5s) avec :
  - Tâches actives, en attente, planifiées, en retry, archivées
  - Compteurs traités/échoués du jour
  - Latence par queue
  - Pastille verte/rouge (active/en pause)
  - Barre de progression colorée
- **Derniers mails** : 5 mails les plus récents

### Mails (`/mails`)

Liste paginée avec filtrage par statut. Chaque mail affiche le sujet, l'expéditeur, le statut et la date.

### Détail mail (`/mails/:id`)

Informations complètes du mail : expéditeur, sujet, corps, destinataires, statut, tentatives, dates.

### Templates (`/templates`)

CRUD complet avec prévisualisation du rendu. Définition de variables pour les templates dynamiques.

### Configs SMTP (`/smtp-configs`)

CRUD + test de connexion SMTP en un clic.

### Tenants (`/tenants`)

Gestion des comptes multi-tenant (création, activation/désactivation, paramètres de rate limiting). Réservé aux administrateurs.

---

## Client API

Le fichier `src/api/client.ts` fournit :

- **Instance Axios** configurée sur `/api/v1`
- **Intercepteur requête** : ajoute automatiquement le header `Authorization: Bearer {token}`
- **Intercepteur réponse** : redirige vers `/login` en cas de 401
- **Types TypeScript** : `Tenant`, `Mail`, `MailRecipient`, `Template`, `SMTPConfig`, `QueueInfo`, `MailStats`, `TenantMailStats`, `PaginatedList<T>`, `APIResponse<T>`

---

## Authentification

Le contexte `AuthContext` gère le cycle de vie JWT :

1. L'utilisateur saisit sa clé API sur `/login`
2. `POST /api/v1/auth/token` retourne un JWT
3. Le token est stocké dans `localStorage`
4. Toutes les requêtes API l'incluent via l'intercepteur Axios
5. Un 401 déclenche la déconnexion automatique

Les routes protégées sont enveloppées par `<ProtectedRoute>` qui vérifie la présence du token.

---

## Développement

```bash
# Installation des dépendances
npm install

# Serveur de développement (port 5173)
npm run dev

# Vérification TypeScript
npx tsc -b

# Build production
npm run build

# Lint
npm run lint
```

Le serveur Vite proxifie `/api` vers `http://localhost:8080` en développement.

---

## Production

Le `Dockerfile` frontend utilise un build multi-stage :

1. **Build** (Node 22 Alpine) : `npm install` + `npm run build`
2. **Runtime** (Nginx Alpine) : sert les fichiers statiques

La configuration Nginx :

- Proxifie `/api/` vers le service API (`http://api:8080`)
- Fallback SPA : toutes les routes non-statiques servent `index.html`
- Cache des assets (JS, CSS, images, fonts) : 1 an
