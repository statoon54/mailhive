#!/bin/sh
# Extrait d'un CHANGELOG au format Keep a Changelog la section d'une version.
#
# Sert à alimenter l'en-tête des notes de release GoReleaser
# (--release-header) : sans lui, les notes se réduisent à une liste de sujets de
# commits, où une rupture de contrat passe inaperçue.
#
# Usage : scripts/changelog-section.sh 0.2.0 [CHANGELOG.md]
set -eu

version="${1:?usage: changelog-section.sh VERSION [FICHIER]}"
file="${2:-CHANGELOG.md}"

# Le tag peut être passé avec ou sans « v ».
version="${version#v}"

[ -f "$file" ] || { echo "changelog introuvable : $file" >&2; exit 1; }

section=$(awk -v v="$version" '
  # Début de la section recherchée : « ## [1.2.3] - AAAA-MM-JJ ».
  $0 ~ "^## \\[" v "\\]" { found = 1; next }
  # Toute autre entête de version termine la section.
  found && /^## / { exit }
  # Le bloc de définitions de liens en fin de fichier aussi : sans cette
  # condition, la dernière version du changelog les absorberait.
  found && /^\[[^]]+\]: / { exit }
  found { print }
' "$file")

# Retire les lignes vides en tête et en queue (awk plutôt que sed : les idiomes
# de rognage sed diffèrent entre BSD et GNU, et la CI tourne sous dash).
section=$(printf '%s\n' "$section" | awk '
  { lines[NR] = $0 }
  END {
    first = 1; last = NR
    while (first <= NR && lines[first] ~ /^[[:space:]]*$/) first++
    while (last >= first && lines[last] ~ /^[[:space:]]*$/) last--
    for (i = first; i <= last; i++) print lines[i]
  }
')

if [ -z "$section" ]; then
  echo "aucune section pour la version $version dans $file" >&2
  exit 1
fi

printf '%s\n' "$section"
