# AGENTS.md

Gemeinsame Anweisungen für alle Coding-Agents (Claude Code, Copilot, Codex, Gemini CLI …) in diesem Repository. `CLAUDE.md` importiert nur diese Datei.

Österreichischer Notenschlüssel-Rechner (Noten 1–5 mit Knickpunkt). Reine Client-App in Vanilla TypeScript mit Vite, ohne Framework und ohne Backend. Schülerdaten verlassen nie den Browser. Projektsprache (UI, Commits, Doku) ist Deutsch, Commits im Conventional-Commits-Stil (`fix:`, `chore:`, `build:`, `deps:` …).

## Befehle

Node ≥ 24 (`.nvmrc`).

```bash
npm run dev                              # Vite Dev-Server, http://localhost:5173
npm run test                             # Vitest (tests/**/*.test.ts, Umgebung: node)
npx vitest run tests/unit/calculator.test.ts   # einzelne Testdatei
npx vitest run -t "Testname"             # einzelner Test nach Name
npm run type-check                       # tsc --noEmit
npm run lint                             # eslint src tests
npm run build                            # Produktions-Build nach dist/
```

Die CI (`.github/workflows/ci.yml`) führt in dieser Reihenfolge aus: type-check → lint → test → build.

## Architektur

Datenfluss: `src/app.ts` (DOM-Controller) → `runCalculationWorkflow()` in `src/ui/workflow.ts` → Rendering über `src/ui/render.ts` und Persistenz in `sessionStorage` (Schlüssel `notenschluessel:lastState`, wird beim Laden mit `isValidRestoredState` geprüft und lässt sich vom Nutzer löschen). Kein `fetch` mit Schülerdaten. Die Exporte (`src/export/`) laufen ebenfalls im Client.

`runCalculationWorkflow` ist reine Logik ohne DOM und der zentrale Testeinstieg (`tests/integration/`). Ablauf mit frühem Abbruch:
validateCoreInputs → calculateGradeBounds → validateGradeBounds → validateInputMode → Parser (CSV oder manuell) → validateStudentPoints → processStudents/Durchschnitt.

- **Diagnostics statt Exceptions/Strings**: Alle Meldungen sind `Diagnostic`-Objekte (`src/types.ts`) und werden über die Factories in `src/core/diagnostics.ts` erzeugt. `severity: "error"` bricht ab, `"warning"` rechnet weiter. Ein neuer Fehlerfall braucht einen `DiagnosticCode` in `types.ts` und eine Factory.
- **Parser** (`src/parsers/`) geben `{ students, diagnostics }` zurück. Fehlerhafte Zeilen werden mit 1-basierter Zeilennummer (inkl. Header) gemeldet und übersprungen.
- **CSV- und Manuell-Modus schließen sich aus.** Daten im jeweils anderen Modus ergeben `input-mode-mismatch`. Beim Moduswechsel leert `app.ts` die Daten des anderen Modus.
- **Algorithmus** (`src/core/calculator.ts`): Der Knickpunkt ist die Untergrenze von Note 4. Der Bereich darüber wird in 4 gleiche Segmente geteilt, und alle Grenzen werden auf Vielfache von `minPoints` (Schrittweite) gerundet. `cleanFloat` beseitigt Float-Artefakte.
- **Eingabegrenzen** (`src/core/validation.ts`): `maxPoints` ist eine positive Ganzzahl ≤ 1000, `minPoints` ist positiv und ≤ `maxPoints`, `breakPointPercent` liegt zwischen 1 und 99. Punktzahlen über `maxPoints` ergeben `points-exceed-max` (Abbruch, prüft `src/core/studentValidation.ts`). Limits stehen in `LIMITS` in `src/constants.ts`.
- `src/index.ts` ist ein Barrel-Export der öffentlichen API (ohne `app.ts`).

## Fallstricke

- Zahlen nie mit `toFixed()` formatieren, sondern immer `formatPoints()` aus `src/ui/format.ts` verwenden (Dezimalkomma, Nachkommastellen aus `minPoints`).
- Kein `innerHTML` mit Daten. `render.ts` und `app.ts` arbeiten ausschließlich mit `textContent`/`createElement`/`.value`.
- Notenfarben kommen nur aus `GRADE_COLORS` in `src/constants.ts`. Die CSS-Klassen `.grade-1`…`.grade-5` in `style.css` und `getGradeStyle()` im Excel-Export müssen dazu passen.
- Excel-Export: Paket `xlsx-js-style` (nicht `xlsx`). Es wird dynamisch nachgeladen und von `vite.config.ts` in einen eigenen Chunk gelegt. Farben als 6-stelliges RGB ohne `#`/ARGB, `patternType: "solid"` ist Pflicht (wegen LibreOffice), und die ganze Zeile A–C wird eingefärbt.
- CSV-Export: Semikolon, Dezimalkomma, UTF-8-BOM. Der Injection-Schutz in `sanitizeCSVField()` maskiert zuerst mit `'` und quotet danach. Beide Schritte laufen nacheinander.
- Security-Header (CSP ohne `unsafe-inline`, HSTS usw.) stehen in `public/_headers`. Inline-Skripte und -Styles brechen deshalb in Produktion.

## Deployment

Cloudflare (statische Assets aus `dist/`, `wrangler.jsonc`). `npm run build` schreibt bei gesetzter Umgebungsvariable `CUSTOM_DOMAIN` eine Custom-Domain-Route in `wrangler.jsonc`. Diese Änderung nach lokalen Builds nicht versehentlich committen.
