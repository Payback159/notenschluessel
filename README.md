# Notenschlüssel

Ein simples Tool, für Lehrerinnen und Lehrer entwickelt. Notenschlüssel nimmt dir die Rechnerei ab. Erstelle schnell und unkompliziert Notenskalen für das österreichische Notensystem (1–5), auch wenn sie einen Knick haben.

Kein Login, keine Datenbank, nichts wird gespeichert. Die Berechnung läuft vollständig im Browser.

[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg?style=flat-square)](LICENSE)

## So geht's

1. Maximale Punktzahl, Schrittweite und Knickpunkt eingeben
2. Schülerdaten optional via CSV oder manueller Tabelle erfassen
3. Ergebnisse ansehen und als CSV/Excel exportieren

## Loslegen

```bash
npm install
npm run dev
```

Browser auf http://localhost:5173 – fertig.

### CSV-Format

```csv
Name;Punkte
Max Mustermann;85,5
Anna Schmidt;76,0
Tom Weber;92,5
```

Beim Import werden Semikolon und Komma als Spaltentrenner erkannt, als Dezimaltrennzeichen
sowohl Komma als auch Punkt. Max. 10 MB.

Exportierte Dateien verwenden Semikolon und Dezimalkomma und beginnen mit einem UTF-8-BOM –
so öffnen sie sich in Excel per Doppelklick korrekt, samt Umlauten.

Zeilen, die nicht gelesen werden können, werden übersprungen und einzeln mit Zeilennummer
gemeldet. Eine Punktzahl über der eingegebenen Maximalpunktzahl bricht die Berechnung ab,
da es sich fast immer um einen Tippfehler handelt.

## Unter der Haube

Frontend: Vanilla TypeScript (Vite).
Server: nginx:alpine liefert den fertigen Build aus.

```text
notenschluessel/
├── nginx.conf                 # nginx-Konfiguration (Security-Header, Healthcheck)
├── style.css                  # Zentrales CSS-Theme
├── index.html                 # App-Einstieg
├── src/                       # TypeScript App
│   ├── app.ts                 # DOM-Controller
│   ├── constants.ts           # Limits, Notenfarben
│   ├── core/                  # Berechnung, Validierung
│   ├── parsers/               # CSV- und Manualparser
│   ├── export/                # CSV- und Excel-Export
│   └── ui/                    # Workflow-Orchestrierung
├── tests/                     # Vitest Unit/Integrationstests
├── dockerfile                 # node:24-alpine → nginx:alpine
└── compose.yml
```

Dependencies:
- [xlsx-js-style](https://www.npmjs.com/package/xlsx-js-style) – Excel-Export mit Zellfarben direkt im Browser

## Sicherheit

- Keine serverseitige Verarbeitung von Schülerdaten
- Security-Header via nginx (CSP, HSTS, X-Frame-Options, Permissions-Policy)
- CSV-Injection-Schutz bei Exporten
- Eingaben und Ergebnisse können in der App jederzeit zurückgesetzt werden

## Tests

```bash
npm run test           # Unit + Integrationstests (Vitest)
npm run type-check     # TypeScript-Typen prüfen
```

## Docker

```bash
docker compose up --build
curl http://localhost:8080/healthz  # -> "OK"
```

Multi-Stage Build: node:24-alpine baut das Frontend, nginx:alpine liefert es aus.

## Lizenz

Apache 2.0 – siehe [LICENSE](LICENSE).
