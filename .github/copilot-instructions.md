# Copilot Instructions for Notenschlüssel

## Project Overview

**Notenschlüssel** ist ein österreichischer Notenschlüssel-Rechner als **client-only TypeScript/Vite App** (DSGVO-konform). Die gesamte Berechnung, CSV-Verarbeitung und Export-Logik läuft im Browser. Der Server liefert nur statische Assets aus.

### Architecture

- **Frontend**: Vanilla TypeScript, gebaut mit Vite, keine Frameworks
- **Server**: `nginx:alpine` liefert statische Assets aus `dist/`
- **Keine serverseitige Verarbeitung von Schülerdaten**
- **Data flow**: DOM-Formular → TS-Core-Logik → sessionStorage → Client-Export

## Technology Stack

### Frontend (in `src/`)
- **TypeScript** (strict mode) mit Vite als Build-Tool
- **Vitest** für Unit- und Integrationstests
- **xlsx-js-style** für Excel-Export direkt im Browser (Fork von SheetJS mit Zell-Styling – das normale `xlsx`-Community-Paket schreibt keine Styles)
- Kein CSS-Framework – zentrales Stylesheet in `style.css`

### Server
- **nginx:alpine** – liefert `dist/`, setzt Security-Header, stellt `/healthz` bereit
- Konfiguration in `nginx.conf`
- Kein Backend-Code, keine Routen für Nutzdaten

## Critical Patterns & Conventions

### Client-only Datenverarbeitung

Schülerdaten verlassen nie den Browser:

```ts
// ✅ DO: Alles läuft im Browser
const result = runCalculationWorkflow({ maxPoints, minPoints, breakPointPercent, inputMode, csvContent, manualEntries });

// ❌ DON'T: Kein POST mit Schülerdaten an Server
fetch("/calculate", { method: "POST", body: JSON.stringify(students) });
```

### Session-Speicherung (Client-side)

Zustand wird nur im Browser gespeichert:

```ts
// Speichern
sessionStorage.setItem("notenschluessel:lastState", JSON.stringify(state));

// Laden
const raw = sessionStorage.getItem("notenschluessel:lastState");

// Löschen (DSGVO: Nutzer kann selbst löschen)
sessionStorage.removeItem("notenschluessel:lastState");
```

### Security-Header (nginx.conf)

Alle Header via `add_header` in `nginx.conf`:

- `X-Frame-Options: DENY`
- `X-Content-Type-Options: nosniff`
- `Referrer-Policy: strict-origin-when-cross-origin`
- `Permissions-Policy: camera=(), microphone=(), geolocation=(), payment=()`
- `Content-Security-Policy: default-src 'self'; script-src 'self'; style-src 'self'`
- `Strict-Transport-Security: max-age=31536000; includeSubDomains`

Wichtig: In nested `location`-Blöcken müssen `add_header`-Direktiven wiederholt werden (nginx erbt sie nicht).

### Input-Modus (CSV vs. Manuell)

Beim Umschalten zwischen Modi werden die Daten des anderen Modus geleert:

```ts
// In setupInputModeToggle():
if (mode === "csv") {
    // Manuelle Zeilen leeren, dann addManualRow()
} else {
    // CSV FileInput leeren: getById<HTMLInputElement>("csvFile").value = "";
}
```

### Eingabevalidierung (Client-side, `src/core/validation.ts`)

- `maxPoints`: Pflicht, positive Ganzzahl, max 1000
- `minPoints`: Pflicht, positive Zahl, ≤ maxPoints
- `breakPointPercent`: 1–99
- CSV und Manuell sind gegenseitig ausschließend
- Degenerate Skalen werden via `validateGradeBounds()` abgefangen
- Validierungsfehler: `showMessage("error", ...)` in der UI

### Notenfarben (Single Source of Truth)

Alle Notenfarben kommen aus `src/constants.ts`:

```ts
export const GRADE_COLORS = {
    1: { bg: "#d4edda", border: "#a3d9a3", text: "#155724" },
    // ...
} as const;
```

- UI: CSS-Klassen `.grade-1` bis `.grade-5` in `style.css`
- Excel-Export: `getGradeStyle()` in `src/export/excelExport.ts`
  - RGB-Format ohne Prefix (z.B. `D4EDDA`) – `xlsx-js-style` erwartet 6-stelliges Hex, kein ARGB
  - `patternType: "solid"` ist Pflicht, sonst ignoriert LibreOffice die Farbe
  - Die ganze Tabellenzeile (Spalten A–C) wird via `styleGradeRow()` eingefärbt, nicht nur die Notenzelle

### CSV-Injection-Schutz

Im CSV-Export werden gefährliche Zeichen am Zeilenanfang escaped:

```ts
// src/export/csvExport.ts
function sanitizeCSVValue(value: string): string { ... }
```

## Project Structure

```
notenschluessel/
├── nginx.conf                     # nginx Server-Konfiguration
├── style.css                      # Zentrales CSS-Theme
├── index.html                     # Vite App-Einstieg
├── src/
│   ├── app.ts                     # DOM-Controller, Form-Handling, Exports
│   ├── constants.ts               # LIMITS, GRADE_COLORS (Single Source of Truth)
│   ├── types.ts                   # Shared TypeScript-Interfaces
│   ├── core/
│   │   ├── calculator.ts          # Notengrenz-Berechnung
│   │   ├── grading.ts             # Schüler-Benotung und Durchschnitt
│   │   └── validation.ts          # Eingabe- und Modus-Validierung
│   ├── parsers/
│   │   ├── csvParser.ts           # CSV-Parsing mit Delimiter-Erkennung
│   │   └── manualParser.ts        # Manuelle Zeilen-Eingabe
│   ├── export/
│   │   ├── csvExport.ts           # CSV-Export mit Injection-Schutz
│   │   └── excelExport.ts         # XLSX-Export mit Notenfarben
│   └── ui/
│       └── workflow.ts            # Orchestrierung: Validierung → Parsing → Berechnung
├── tests/
│   ├── unit/                      # calculator, grading, validation, csvParser, manualParser, export
│   └── integration/               # fullWorkflow, inputModes
├── dockerfile                     # Multi-Stage: node:24-alpine Build → nginx:alpine Runtime
└── compose.yml                    # Docker Compose für lokale Entwicklung
```

## Development Workflows

### Lokal entwickeln (schnellste Schleife)

```bash
npm install
npm run dev       # Vite Dev-Server auf http://localhost:5173
```

### Produktions-Build lokal testen

```bash
npm run build
docker compose up --build
curl http://localhost:8080/healthz  # → "OK"
```

### Tests

```bash
npm run test           # Vitest: Unit + Integration
npm run type-check     # TypeScript-Typen prüfen
```

## Notenberechnungs-Algorithmus

`src/core/calculator.ts` implementiert die österreichische 1–5-Skala.

Der **Knickpunkt** ist die Bestehensschwelle (Untergrenze Note 4). Darunter → Note 5. Der Bereich vom Knickpunkt bis Maximalpunktzahl wird in 4 gleiche Segmente geteilt.

Mit `breakAbs = maxPoints * breakPointPercent/100` und `segment = (maxPoints - breakAbs) / 4`:

- Note 1: `breakAbs + 3*segment` bis maxPoints
- Note 2: `breakAbs + 2*segment` bis Note-1-Untergrenze
- Note 3: `breakAbs + 1*segment` bis Note-2-Untergrenze
- Note 4: `breakAbs` bis Note-3-Untergrenze
- Note 5: 0 bis unter Knickpunkt

**Rundung**: Alle Grenzen auf nächstes `minPoints`-Inkrement gerundet.

## Häufige Fehler vermeiden

1. ❌ Schülerdaten an Server senden – Berechnung läuft komplett im Browser
2. ❌ CSS-Klassen für Noten in `style.css` und Farben in `constants.ts` separat pflegen – immer aus `GRADE_COLORS` ableiten
3. ❌ XLSX-Zellfarben ohne `patternType: "solid"` setzen – LibreOffice ignoriert sie sonst
4. ❌ Excel-Styles mit `xlsx` (Community) statt `xlsx-js-style` schreiben – Styles werden dann verworfen; Farben als 6-stelliges RGB (`D4EDDA`), nicht ARGB
5. ❌ CSV und Manuell gleichzeitig übergeben – gegenseitig ausschließend
6. ❌ Beim Mode-Umschalten alte Eingabedaten nicht leeren – führt zu falschem Validierungsfehler
7. ❌ Security-Header nur im Server-Block definieren – in nginx nested `location`-Blöcken müssen `add_header` wiederholt werden

## Wichtige Dateien

- **Algorithmus**: `src/core/calculator.ts`
- **Validierung**: `src/core/validation.ts`
- **Notenfarben (Basis)**: `src/constants.ts` → `GRADE_COLORS`
- **Excel-Export**: `src/export/excelExport.ts`
- **DOM-Controller**: `src/app.ts`
- **CSS-Theme**: `style.css`
- **nginx-Konfiguration**: `nginx.conf`
