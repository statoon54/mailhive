#!/bin/bash
# Variante de send_2000_mails.sh avec une PIÈCE JOINTE IDENTIQUE pour tous les
# destinataires. Objectif : exercer la déduplication des pièces jointes
# (même contenu -> 1 seul blob stocké, N références).
#
# Usage : ./send_2000_mails_attachments.sh [NB_MAILS] [TAILLE_PJ_KO]
#   NB_MAILS      : nombre de mails à envoyer (défaut 2000)
#   TAILLE_PJ_KO  : taille de la pièce jointe en Ko (défaut 1024 = 1 Mo)

set -u

COUNT=${1:-2000}
ATTACH_KB=${2:-1024}
PARALLEL=20

TOKEN=$(curl -s -X POST http://localhost:8080/api/v1/auth/token -H "Content-Type: application/json" -d '{"api_key": "admin-dev-key"}' | jq -r '.data.token')
TEMPLATE_ID="636ac3a6-122d-447e-805e-beb7b5c48d54"
ATTACH_FILENAME="document.pdf"
ATTACH_CONTENT_TYPE="application/pdf"

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
      -X POST http://localhost:8080/api/v1/mails \
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
