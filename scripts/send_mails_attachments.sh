#!/usr/bin/env bash
# Script d'exemple : envoi en masse de mails avec une PIÈCE JOINTE IDENTIQUE pour
# tous les destinataires. Objectif : exercer la déduplication des pièces jointes
# (même contenu -> 1 seul blob stocké, N références).
#
# Usage :
#   TEMPLATE_ID=<uuid> [API_KEY=<clé>] [BASE_URL=<url>] \
#     ./scripts/send_mails_attachments.sh [NB_MAILS] [TAILLE_PJ_KO]
#
# Paramètres (variables d'environnement) :
#   TEMPLATE_ID   (obligatoire) UUID du template à utiliser
#   API_KEY       (défaut: admin-dev-key) clé API du TENANT émetteur — c'est elle
#                 qui détermine le tenant (les mails sont créés sous ce tenant)
#   BASE_URL      (défaut: http://localhost:8080) URL de base de l'API
#
# Arguments positionnels :
#   NB_MAILS      (défaut 2000)        nombre de mails à envoyer
#   TAILLE_PJ_KO  (défaut 1024 = 1 Mo) taille de la pièce jointe en Ko
#
# Exemple :
#   TEMPLATE_ID=636ac3a6-122d-447e-805e-beb7b5c48d54 \
#     ./scripts/send_mails_attachments.sh 500 2048

set -u

TEMPLATE_ID="${TEMPLATE_ID:?TEMPLATE_ID requis : UUID du template (voir en-tête du script)}"
API_KEY="${API_KEY:-admin-dev-key}"
BASE_URL="${BASE_URL:-http://localhost:8080}"

COUNT="${1:-2000}"
ATTACH_KB="${2:-1024}"
PARALLEL=20

ATTACH_FILENAME="document.pdf"
ATTACH_CONTENT_TYPE="application/pdf"

# Authentification : le token (donc le tenant émetteur) découle de l'API_KEY.
TOKEN=$(curl -s -X POST "$BASE_URL/api/v1/auth/token" \
  -H "Content-Type: application/json" \
  -d "{\"api_key\": \"$API_KEY\"}" | jq -r '.data.token')
if [ -z "$TOKEN" ] || [ "$TOKEN" = "null" ]; then
  echo "Échec d'authentification (API_KEY=$API_KEY, BASE_URL=$BASE_URL)" >&2
  exit 1
fi

# Répertoire de travail temporaire (corps des requêtes + pièce jointe).
# Les corps de requête sont écrits dans des fichiers et envoyés via `curl -d @fichier`
# pour éviter "Argument list too long" : la PJ base64 (~1,3x la taille brute) dépasse
# sinon ARG_MAX quand elle est passée en argument de ligne de commande.
WORKDIR=$(mktemp -d)
trap 'rm -rf "$WORKDIR"' EXIT

# Générer une pièce jointe déterministe (même contenu à chaque exécution -> dédup
# aussi entre deux runs), encodée en base64 une seule fois, stockée dans un fichier.
echo "Génération de la pièce jointe (${ATTACH_KB} Ko)..."
ATTACH_FILE="$WORKDIR/attachment.b64"
head -c $((ATTACH_KB * 1024)) /dev/zero | base64 | tr -d '\n' > "$ATTACH_FILE"
echo "Pièce jointe prête : $(wc -c < "$ATTACH_FILE") caractères base64."
echo "Template: $TEMPLATE_ID — Tenant (via API_KEY): $API_KEY — Envoi de $COUNT mails."

FIRST_NAMES=("Alice" "Bob" "Charlie" "Diana" "Emma" "Francois" "Gabriel" "Hannah" "Ines" "Julien" "Karim" "Laura" "Marie" "Nicolas" "Olivia" "Pierre" "Quentin" "Rachel" "Sophie" "Thomas")
LAST_NAMES=("Martin" "Bernard" "Dubois" "Thomas" "Robert" "Richard" "Petit" "Durand" "Leroy" "Moreau" "Simon" "Laurent" "Lefebvre" "Michel" "Garcia" "David" "Bertrand" "Roux" "Vincent" "Fournier")
DOMAINS=("gmail.com" "yahoo.fr" "hotmail.com" "outlook.fr" "orange.fr")

for i in $(seq 1 "$COUNT"); do
  (
    FIRST=${FIRST_NAMES[$((RANDOM % ${#FIRST_NAMES[@]}))]}
    LAST=${LAST_NAMES[$((RANDOM % ${#LAST_NAMES[@]}))]}
    DOMAIN=${DOMAINS[$((RANDOM % ${#DOMAINS[@]}))]}
    FIRST_LOWER=$(echo "$FIRST" | tr '[:upper:]' '[:lower:]')
    LAST_LOWER=$(echo "$LAST" | tr '[:upper:]' '[:lower:]')
    EMAIL="${FIRST_LOWER}.${LAST_LOWER}${RANDOM}@${DOMAIN}"
    TOKEN_VAL=$(LC_ALL=C tr -dc 'a-z0-9' < /dev/urandom | head -c 32)
    ACTIVATION_URL="https://monactivation.com/activate?token=${TOKEN_VAL}"

    # Construire le corps de la requête dans un fichier (printf/cat sont des builtins
    # ou lisent un fichier : pas de limite ARG_MAX). La PJ base64 est insérée via cat.
    BODY="$WORKDIR/body_${i}.json"
    {
      printf '{"to":[{"email":"%s","name":"%s %s"}],"template_id":"%s","subject":"Ouverture de compte informatique pour {{.name}}","template_data":{"name":"%s","activation_url":"%s"},"attachments":[{"filename":"%s","content_type":"%s","content":"' \
        "$EMAIL" "$FIRST" "$LAST" "$TEMPLATE_ID" "$FIRST" "$ACTIVATION_URL" "$ATTACH_FILENAME" "$ATTACH_CONTENT_TYPE"
      cat "$ATTACH_FILE"
      printf '"}]}'
    } > "$BODY"

    curl -s -o /dev/null -w "[${i}] %{http_code}\n" \
      -X POST "$BASE_URL/api/v1/mails" \
      -H "Content-Type: application/json" \
      -H "Authorization: Bearer ${TOKEN}" \
      --data-binary @"$BODY"

    rm -f "$BODY"
  ) &

  # Limiter à $PARALLEL jobs simultanés
  if (( $(jobs -rp | wc -l) >= PARALLEL )); then
    wait -f %% 2>/dev/null || wait %%
  fi

  if (( i % 100 == 0 )); then
    echo "--- Progression: ${i}/${COUNT} ---"
  fi
done

wait
echo "Terminé ! ${COUNT} mails envoyés avec la même pièce jointe de ${ATTACH_KB} Ko."
