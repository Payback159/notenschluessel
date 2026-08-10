# Datenintegrität, Anzeige und Export

Stand: 2026-08-09

## Problem

Die App liefert in mehreren Fällen still ein falsches oder unvollständiges Ergebnis. Für ein
Werkzeug, das Schulnoten erzeugt, ist das die schwerwiegendste Fehlerklasse: Eine Lehrkraft
hat keine Möglichkeit zu erkennen, dass etwas schiefgegangen ist.

Alle folgenden Befunde wurden gegen den Stand von `main` (Commit `4c15e11`) ausführbar
verifiziert.

### B1 — Punktzahlen oberhalb der Maximalpunktzahl werden gewertet

`src/parsers/csvParser.ts:112` prüft gegen `LIMITS.maxPoints` (1000), nicht gegen die von der
Lehrkraft eingegebene Maximalpunktzahl. Der Parser kennt diesen Wert nicht.

Bei `maxPoints = 100` ergibt die Zeile `Tippfehler,855` das Ergebnis
`{"name":"Tippfehler","points":855,"grade":1}` bei `ok = true` und leerer Fehlerliste. Ein
verrutschter Dezimalpunkt (855 statt 85,5) wird zur Bestnote.

### B2 — Verworfene Zeilen werden nie gemeldet

`parseCSVContent` liefert `skippedRows` zurück. `runCalculationWorkflow`
(`src/ui/workflow.ts:50-52`) übernimmt nur `result.errors` und verwirft das Feld.

Eingabe `Name,Punkte\nA,10\nKaputt\nB,abc\nC,20` ergibt im Parser `skippedRows = 2`, im
Workflow jedoch `ok = true`, zwei Schüler, `errors = []`. Wer 28 Namen importiert und 26
zurückbekommt, merkt es nur durch Nachzählen.

### B3 — Daten im inaktiven Eingabemodus werden ignoriert

`src/ui/workflow.ts:49-59` führt den CSV-Zweig nur bei `inputMode === "csv" && csvProvided`
aus, den manuellen entsprechend. Liegen die Daten im jeweils anderen Modus, greift kein Zweig
und keine Prüfung schlägt an.

Mit `inputMode: "csv"`, leerem `csvContent` und einem gefüllten `manualEntries` ergibt sich
`ok = true`, null Schüler, `errors = []`. Die Oberfläche umgeht das, indem sie beim Umschalten
die Felder des anderen Modus leert — die Kernlogik bleibt darauf angewiesen.

### B4 — Angezeigte Skala weicht von der berechneten ab

`src/app.ts:101` und `:111` formatieren mit `toFixed(1)`, unabhängig von der gewählten
Schrittweite. Bei Schrittweite 0,25 hat die berechnete Grenze zwei Nachkommastellen.

Für `maxPoints = 37`, `minPoints = 0,25`, `breakPointPercent = 60` beginnt Note 1 rechnerisch
bei **33,25**, angezeigt und ausgedruckt wird **33,3**. Der CSV-Export schreibt den Rohwert.
Bildschirm, Ausdruck und Export widersprechen sich.

Zusätzlich setzt `index.html:31` für dieses Feld `step="0.1"`. Eine Schrittweite von 0,25 wird
dadurch von der Browser-Validierung abgelehnt, obwohl die Kernlogik sie unterstützt und
`tests/unit/calculator.test.ts` sie testet.

### B5 — CSV-Injection-Schutz zerstört die Spaltenstruktur

`src/export/csvExport.ts:9-16` behandelt Apostroph-Präfix und Quoting als Alternativen: Nach
dem `return` im Sonderzeichen-Zweig wird nicht mehr auf Trennzeichen geprüft.

`{ name: "=Mueller, Hans", points: 90, grade: 1 }` exportiert als
`'=Mueller, Hans,90,1` — vier Felder statt drei.

### B6 — Export ist in deutschsprachigem Excel unbrauchbar

Der Export beginnt direkt mit `Name,Punkte,Note`, ohne UTF-8-BOM. Excel unter Windows liest
das als ANSI: aus „Müller" wird „MÃ¼ller". Betroffen sind auch die eigenen Überschriften
(„Schüler", „Notenschlüssel Export"). Zusätzlich erkennt der Import Semikolon als Trennzeichen,
der Export schreibt jedoch immer Komma — in deutscher Regionaleinstellung landet die gesamte
Zeile in einer Spalte.

### B7 — Fehlermeldungen erreichen Screenreader nicht

`#message` (`index.html:21`) besitzt weder `role="alert"` noch `aria-live`. `showMessage`
tauscht nur `textContent` und CSS-Klassen; assistive Technologie erfährt davon nichts.

### B8 — Kleinere Punkte

- `sanitizeName` ist in `csvParser.ts:4-14` und `manualParser.ts:4-14` wortgleich dupliziert.
- Schülernamen werden über `innerHTML` gerendert (`app.ts:101,111`). Aktuell durch
  `sanitizeName` entschärft, das `<` und `>` entfernt; aus `sessionStorage` wiederhergestellte
  Daten durchlaufen diese Bereinigung jedoch nicht erneut.
- `dockerfile:2` baut auf `node:26-alpine`, CI und `.nvmrc` verwenden Node 24.
- `README.md` nennt die Abhängigkeit `xlsx`; tatsächlich verwendet wird `xlsx-js-style`.
- Meldungstexte mischen „Ungueltige" und „Schülerdaten".
- Kein Linter konfiguriert.

## Gemeinsame Ursache

B1, B2 und B3 sind Symptome derselben Lücke: Die Kernlogik kennt Probleme, kann sie aber nicht
transportieren. `WorkflowResult` besitzt mit `errors: string[]` genau einen Kanal, und dieser
bedeutet immer Abbruch. Es fehlt der Zustand „gerechnet, aber mit Vorbehalt".

Das Design behebt diese Ursache, statt drei Symptome einzeln zu reparieren.

## Entscheidungen

| Frage | Entscheidung |
| :--- | :--- |
| Verhalten bei fehlerhaften Zeilen | Abgestuft: Strukturfehler warnen und weiterrechnen, Punkte über Maximum brechen ab |
| CSV-Trennzeichen im Export | Semikolon, Zahlen mit Dezimalkomma, BOM immer |
| Umbau von `app.ts` | Formatierung und Rendering herauslösen, keine jsdom-Tests |
| Fehlerbehandlung allgemein | Strukturierte Diagnosen, kein durchgehender `Result<T, E>`-Typ |

Begründung der ersten Zeile: Eine unlesbare oder unvollständige Zeile ist ein Datenproblem, das
die Lehrkraft sehen und selbst bewerten kann. Eine Punktzahl über dem Maximum ist dagegen fast
immer ein Tippfehler, der eine konkret falsche Note erzeugt — hier ist Weiterrechnen schädlich.

## Architektur

### Diagnose-Typ

Neu in `src/types.ts`:

```ts
export type Severity = "error" | "warning";

export interface Diagnostic {
    severity: Severity;
    code: DiagnosticCode;
    row?: number;      // 1-basierte Zeilennummer der Quelldatei, sofern zuordenbar
    message: string;   // fertiger deutscher Text für die Anzeige
}
```

`DiagnosticCode` ist ein String-Union über die unten aufgeführten Codes. Der Code ist die
testbare Zusicherung, `message` reine Präsentation — Tests prüfen den Code, nicht den Text.

### Betroffene Schnittstellen

`CSVParseResult` und `ManualParseResult` ersetzen `errors: string[]` und `skippedRows` durch
`diagnostics: Diagnostic[]`. `ValidationResult` und `GradeBoundsValidationResult` ebenso.

`WorkflowResult` wird zu:

```ts
export interface WorkflowResult {
    ok: boolean;                  // true, wenn keine Diagnose severity "error" hat
    diagnostics: Diagnostic[];
    gradeBounds: GradeBound[];
    students: Student[];
    averageGrade: number;
}
```

Das Feld `errors` entfällt ersatzlos. Das Projekt ist `private: true` und hat keine externen
Konsumenten; `src/index.ts` bleibt der einzige Sammelexport.

Wichtig: Bei `ok = true` dürfen Warnungen vorliegen. `gradeBounds` und `students` sind dann
gültig und werden angezeigt. Bei `ok = false` bleiben beide leer.

### Diagnose-Codes

**Abbruch (`error`)**

| Code | Auslöser |
| :--- | :--- |
| `invalid-core-input` | Maximalpunktzahl, Schrittweite oder Knickpunkt außerhalb der erlaubten Bereiche |
| `degenerate-scale` | `validateGradeBounds` schlägt fehl |
| `input-mode-conflict` | CSV **und** manuelle Eingabe gleichzeitig befüllt |
| `input-mode-mismatch` | Daten liegen ausschließlich im nicht gewählten Eingabemodus (B3) |
| `points-exceed-max` | Punktzahl größer als die eingegebene Maximalpunktzahl (B1) |
| `no-valid-students` | Eingabe vorhanden, aber keine einzige verwertbare Zeile |

**Warnung (`warning`)**

| Code | Auslöser |
| :--- | :--- |
| `row-too-few-columns` | Zeile hat weniger als zwei Spalten |
| `row-missing-name` | Namensfeld leer, Punktefeld befüllt |
| `row-missing-points` | Namensfeld befüllt, Punktefeld leer (nur manueller Modus) |
| `row-unparsable-points` | Punktefeld nicht als Zahl lesbar |
| `row-negative-points` | Punktzahl kleiner als 0 |
| `row-limit-reached` | Grenze `LIMITS.maxStudents` erreicht, weitere Zeilen wurden nicht gelesen |

Vollständig leere Zeilen erzeugen weiterhin keine Diagnose und werden übersprungen.

Das Zeilenlimit wird zwischen beiden Parsern vereinheitlicht. Heute bricht `csvParser` still ab
(`break`), während `manualParser` einen Fehler meldet und **alle** bereits gelesenen Schüler
verwirft. Künftig gilt für beide: Die ersten `LIMITS.maxStudents` Zeilen werden verarbeitet, der
Rest wird verworfen und über `row-limit-reached` gemeldet. Ein Warnung statt eines Fehlers ist
hier richtig, weil das Ergebnis für die gelesenen Zeilen korrekt bleibt und die Grenze mit
10 000 Zeilen weit oberhalb jeder realen Klassenliste liegt.

`points-exceed-max` ist bewusst ein Fehler und keine Warnung, obwohl es zeilenbezogen ist: Die
betroffene Zeile würde sonst mit einer falschen Note im Ergebnis erscheinen. Alle solchen
Zeilen werden gesammelt, damit die Lehrkraft sie in einem Durchgang korrigieren kann, statt sie
einzeln zu finden.

### Datenfluss

```
Eingabe
  → validateCoreInputs                    → error: invalid-core-input
  → calculateGradeBounds
  → validateGradeBounds                   → error: degenerate-scale
  → Modusprüfung                          → error: input-mode-conflict | input-mode-mismatch
  → Parser (CSV oder manuell)             → warning: row-*
  → validateStudentPoints(…, maxPoints)   → error: points-exceed-max      ← NEU
  → processStudents / calculateAverageGrade
  → WorkflowResult { ok, diagnostics, … }
  → format.ts  (reine Formatierung)
  → render.ts  (DOM-Ausgabe)
```

Die Prüfung gegen die Maximalpunktzahl ist eine eigene Stufe **nach** dem Parsen. Sie gehört
nicht in den Parser: Dieser kennt die Maximalpunktzahl nicht, was genau die Ursache von B1 ist.
Die absolute Obergrenze `LIMITS.maxPoints` bleibt zusätzlich im Parser bestehen.

Neue Datei `src/core/studentValidation.ts`:

```ts
export function validateStudentPoints(students: Student[], maxPoints: number): Diagnostic[];
```

Die Reihenfolge ist verbindlich: Kernvalidierung und Skalenprüfung laufen vor der Modusprüfung,
damit eine unbrauchbare Skala nicht durch eine Modusmeldung verdeckt wird.

### Modusprüfung

Ersetzt `validateInputModeExclusivity`. Aus `inputMode`, `csvProvided` und `manualProvided`
ergibt sich:

| `inputMode` | `csvProvided` | `manualProvided` | Ergebnis |
| :--- | :--- | :--- | :--- |
| beliebig | ja | ja | `input-mode-conflict` |
| `csv` | ja | nein | CSV parsen |
| `csv` | nein | ja | `input-mode-mismatch` |
| `manual` | nein | ja | Manuell parsen |
| `manual` | ja | nein | `input-mode-mismatch` |
| beliebig | nein | nein | Nur Skala berechnen, `ok = true` |

Die letzte Zeile erhält das bestehende, gewollte Verhalten aus
`tests/integration/inputModes.test.ts`: Ohne Schülerdaten wird lediglich die Notenskala erzeugt.

### Formatierung

Neu, `src/ui/format.ts`, ohne DOM-Zugriff und ohne Importe aus `app.ts`:

```ts
export function decimalsFor(minPoints: number): number;
export function formatPoints(value: number, minPoints: number): string;
export function formatRange(bound: GradeBound, minPoints: number): string;
export function formatNumberDe(value: number, decimals: number): string;
```

`decimalsFor` leitet die Nachkommastellen aus der Schrittweite ab: 1 → 0, 0,5 → 1, 0,25 → 2.
Die Ableitung erfolgt über die Dezimaldarstellung der Zahl, begrenzt auf maximal drei
Nachkommastellen. Werte in Exponentialschreibweise werden auf 3 abgebildet; das Eingabefeld
lässt solche Werte nicht zu.

`formatNumberDe` ist die gemeinsame Zahlendarstellung und verwendet **Dezimalkomma**. Das gilt
ausdrücklich auch für die Bildschirmanzeige, die heute noch `toFixed` mit Dezimalpunkt nutzt:
Die Oberfläche ist durchgängig deutsch, und aus `76.5` wird damit `76,5`. Dadurch stimmen
Anzeige und CSV-Export auch in der Schreibweise überein, nicht nur in der Genauigkeit.

Diese Funktionen sind die einzige Quelle für Zahlendarstellung. Anzeige und CSV-Export rufen
dieselbe Formatierung auf — damit ist die Divergenz aus B4 strukturell ausgeschlossen und nicht
nur an drei Stellen einzeln korrigiert.

Ergänzend wird `index.html:31` von `step="0.1"` auf `step="any"` geändert, damit die
Browser-Validierung die von der Kernlogik unterstützten Schrittweiten nicht ablehnt.

### Rendering

Neu, `src/ui/render.ts`: übernimmt aus `app.ts` das Befüllen von Notenskala-Tabelle,
Schülertabelle, Durchschnittsanzeige und Diagnoseliste.

Alle Textinhalte werden über `textContent` gesetzt, nicht über `innerHTML`. Das beseitigt B8 und
macht die Darstellung unabhängig davon, ob Werte die Bereinigung durchlaufen haben — relevant
für aus `sessionStorage` wiederhergestellte Zustände.

`app.ts` behält Event-Verdrahtung, State, `sessionStorage` und Export-Auslösung. Zielgröße
etwa 180 statt 339 Zeilen.

### Diagnose-Anzeige

Der bestehende `#message`-Container wird um eine Liste ergänzt: Fehler und Warnungen erscheinen
als Aufzählung, jeweils mit Zeilennummer, sofern vorhanden. Der Container erhält
`role="alert"`, damit Screenreader die Meldung erhalten (B7).

Bei `ok = true` mit Warnungen wird zusätzlich die Anzahl der berechneten Schüler genannt, damit
die Lehrkraft die Vollständigkeit gegen ihre Klassenliste prüfen kann.

### Export

`src/export/csvExport.ts`:

- Trennzeichen `;` in allen Exportfunktionen.
- Zahlen über `formatNumberDe`, also mit Dezimalkomma.
- `sanitizeCSVField` verkettet die Schritte statt sie zu verzweigen: Beginnt das Feld mit
  `=`, `+`, `-`, `@`, Tab oder Wagenrücklauf, wird ein Apostroph vorangestellt; **anschließend**
  wird geprüft, ob das Ergebnis Trennzeichen, Anführungszeichen oder Zeilenumbruch enthält, und
  gegebenenfalls gequotet (B5).
- `triggerTextDownload` stellt beim Erzeugen des Blobs das BOM `U+FEFF` voran. Es sitzt bewusst
  dort und nicht in den Exportfunktionen, damit deren Rückgabewerte in Tests unverändert
  prüfbar bleiben.

Der Round-Trip bleibt erhalten: `detectDelimiter` erkennt Semikolon, und die bestehende
Ersetzung von Komma durch Punkt im Punktefeld liest `76,5` korrekt als `76.5`. Der Importer muss
zusätzlich ein führendes BOM in der ersten Zeile verwerfen, damit eine zuvor exportierte Datei
wieder eingelesen werden kann — heute würde es an den ersten Spaltennamen geraten und die
Kopfzeilenerkennung in `csvParser.ts:78` scheitern lassen.

Der Excel-Export (`excelExport.ts`) bleibt unverändert. Er schreibt Punktzahlen und Grenzen
weiterhin als **numerische** Zellen, nicht als formatierten Text: In einer Tabellenkalkulation
sind Zahlen sortier- und weiterrechenbar, und die Darstellung übernimmt dort das Zellformat.
`format.ts` gilt für Bildschirm und CSV, nicht für XLSX.

### Weitere Bereinigungen

- `sanitizeName` wandert nach `src/core/sanitize.ts`; beide Parser importieren von dort (B8).
- `dockerfile` verwendet `node:24-alpine`, passend zu `.nvmrc` und CI.
- `README.md` nennt `xlsx-js-style` statt `xlsx`.
- Meldungstexte erhalten durchgängig korrekte Umlaute.
- ESLint mit TypeScript-Regelsatz, Skript `npm run lint`, ergänzt als Schritt in `ci.yml`.

## Tests

Die 39 bestehenden Tests bleiben inhaltlich gültig, brauchen aber Anpassung an `diagnostics`
statt `errors`. Sie sind Teil der jeweiligen Etappe, nicht ein nachgelagerter Schritt.

Neue Abdeckung:

- **Diagnosen:** je ein Test pro Code aus den beiden Tabellen, der Code und Severity prüft.
- **Abgestuftes Verhalten:** eine CSV mit sowohl kaputten Zeilen als auch einer überhöhten
  Punktzahl ergibt `ok = false`; dieselbe CSV ohne die überhöhte Zeile ergibt `ok = true` mit
  Warnungen und der korrekten Schülerzahl.
- **`row-limit-reached`:** Eingabe oberhalb `LIMITS.maxStudents` erzeugt die Warnung.
- **Formatierung:** `decimalsFor` für 1, 0,5, 0,25, 0,1; `formatRange` für die aus B4 bekannte
  Kombination 37 / 0,25 / 60 liefert `33,25` und nicht `33,3`.
- **Export:** `=Mueller, Hans` ergibt ein korrekt gequotetes Einzelfeld; Semikolon-Trennung;
  Dezimalkomma; BOM im heruntergeladenen Blob.
- **Round-Trip:** Export einer Schülerliste, erneuter Import über `parseCSVContent`, identische
  Namen und Punktzahlen — einmal ohne und einmal mit vorangestelltem BOM, damit eine zuvor
  heruntergeladene Datei nachweislich wieder einlesbar ist.

Nicht Teil dieser Arbeit: jsdom- oder Playwright-Tests für `app.ts`.

## Etappen

Jede Etappe ist für sich lauffähig, mit grünen Tests und Type-Check.

1. **Diagnose-Kanal** — `Diagnostic`-Typ, Umstellung von Parsern, Validierung und Workflow,
   neue Modusprüfung, `validateStudentPoints`. Behebt B1, B2, B3. Anzeige zunächst als
   einfache Textliste.
2. **Formatierung und Rendering** — `format.ts`, `render.ts`, Ausdünnung von `app.ts`,
   `step="any"`, Diagnoseliste mit `role="alert"`. Behebt B4 und B7.
3. **Export** — Semikolon, Dezimalkomma, BOM, korrigierte Feldbereinigung. Behebt B5 und B6.
4. **Bereinigungen** — `sanitize.ts`, Node-Version, README, Umlaute, ESLint. Behebt B8.

Die Reihenfolge ist bindend: Etappe 2 setzt den Diagnose-Typ aus Etappe 1 voraus, Etappe 3 die
Formatierungsfunktionen aus Etappe 2.

## Nicht im Umfang

Aus `docs/improvement-roadmap.md`: Visualisierung der Notenverteilung, Dark Mode, persistente
Einstellungen, Web Worker, Playwright-E2E-Tests. Diese Punkte bleiben unberührt und behalten
ihre dortige Priorisierung.

Ebenfalls außen vor: ein durchgehender `Result<T, E>`-Typ, jsdom-Tests für `app.ts`, sowie
Änderungen am Berechnungsalgorithmus selbst — dieser ist korrekt und bleibt unangetastet.

## Akzeptanzkriterien

1. Eine CSV mit einer Punktzahl über der Maximalpunktzahl führt zum Abbruch mit Nennung der
   betroffenen Zeilen; es erscheint keine Note.
2. Eine CSV mit fehlerhaften Zeilen berechnet die übrigen Schüler, listet jede übersprungene
   Zeile mit Nummer und Grund auf und nennt die Zahl der berechneten Schüler.
3. Daten im nicht gewählten Eingabemodus führen zu einer erklärenden Fehlermeldung statt zu
   einem leeren Ergebnis.
4. Bei Schrittweite 0,25 stimmen angezeigte, ausgedruckte und exportierte Grenzen überein.
5. Ein exportierter CSV-Export öffnet sich in deutschsprachigem Excel per Doppelklick korrekt
   in Spalten, mit lesbaren Umlauten.
6. Ein Name, der mit `=` beginnt und ein Komma enthält, belegt im Export genau ein Feld.
7. Fehler- und Warnmeldungen werden von Screenreadern angesagt.
8. `npm run test`, `npm run type-check` und `npm run lint` laufen fehlerfrei.
