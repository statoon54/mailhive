// Actions Go template, y compris multilignes : {{ ... }}, avec les marqueurs de
// découpage d'espaces optionnels ({{- ... -}}).
const ACTION_RE = /\{\{-?([\s\S]*?)-?\}\}/g;

// Chaîne de champs à l'intérieur d'une action, dont seul le premier segment est
// capturé : « .Client.Nom » donne « Client », et consommer la chaîne entière
// évite de compter « Nom » comme une seconde variable.
//
// Exiger le point initial écarte les mots-clés (if, range, end...) et les noms
// de fonctions ; la lookbehind écarte les champs d'une variable locale
// (« {{$v.Nom}} »), qui ne viennent pas de template_data.
const FIELD_RE = /(?<![\w$)\]])\.([A-Za-z_][A-Za-z0-9_]*)(?:\.[A-Za-z_][A-Za-z0-9_]*)*/g;

/**
 * extractVariables liste les variables référencées dans les corps fournis, dans
 * leur ordre d'apparition et sans doublon. Seul le premier segment est retenu :
 * {{.Client.Nom}} donne « Client ».
 */
export function extractVariables(...sources: string[]): string[] {
  const found = new Set<string>();
  for (const source of sources) {
    for (const [, action] of source.matchAll(ACTION_RE)) {
      for (const [, name] of action.matchAll(FIELD_RE)) {
        found.add(name);
      }
    }
  }
  return [...found];
}

/** undeclaredVariables liste les variables utilisées qui ne sont pas déclarées. */
export function undeclaredVariables(declared: Record<string, string>, ...sources: string[]): string[] {
  return extractVariables(...sources).filter((name) => !(name in declared));
}
