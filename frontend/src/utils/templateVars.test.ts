import { test } from 'node:test';
import assert from 'node:assert/strict';

import { extractVariables, undeclaredVariables } from './templateVars.ts';

test('extractVariables', async (t) => {
  const cases: { name: string; sources: string[]; expected: string[] }[] = [
    { name: 'plusieurs variables dans leur ordre d\'apparition', sources: ['<p>Bonjour {{.Prenom}} {{.Nom}}</p>'], expected: ['Prenom', 'Nom'] },
    { name: 'dédoublonne', sources: ['{{.Prenom}} et {{.Prenom}}'], expected: ['Prenom'] },
    { name: 'ne retient que la racine d\'une chaîne de champs', sources: ['{{.Client.Adresse.Ville}}'], expected: ['Client'] },
    { name: 'ignore les mots-clés Go template', sources: ['{{if .Actif}}oui{{end}}{{range .Items}}x{{end}}'], expected: ['Actif', 'Items'] },
    { name: 'tolère les marqueurs de découpage d\'espaces', sources: ['{{-  .Prenom  -}}'], expected: ['Prenom'] },
    { name: 'tolère un pipeline', sources: ['{{ .Nom | upper }}'], expected: ['Nom'] },
    { name: 'tolère une action multiligne', sources: ['{{if\n  .Actif}}x{{end}}'], expected: ['Actif'] },
    { name: 'ignore les champs d\'une variable locale', sources: ['{{range $i, $v := .Items}}{{$v.Nom}}{{end}}'], expected: ['Items'] },
    { name: 'lit la source d\'une affectation locale', sources: ['{{$local := .Prenom}}{{$local}}'], expected: ['Prenom'] },
    { name: 'sans point, ce n\'est pas une variable', sources: ['{{Prenom}}'], expected: [] },
    { name: 'texte sans action', sources: ['<p>Bonjour</p>'], expected: [] },
    { name: 'fusionne les sources', sources: ['{{.A}}', '{{.B}}', '{{.A}}'], expected: ['A', 'B'] },
  ];

  for (const { name, sources, expected } of cases) {
    await t.test(name, () => {
      assert.deepEqual(extractVariables(...sources), expected);
    });
  }
});

test('undeclaredVariables ne garde que les variables absentes de la déclaration', () => {
  assert.deepEqual(
    undeclaredVariables({ Prenom: 'Prénom' }, '{{.Prenom}} {{.Societe}}'),
    ['Societe'],
  );
});
