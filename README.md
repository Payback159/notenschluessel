# Notenschlüssel

Ein simples Tool, für Lehrerinnen und Lehrer entwickelt. Notenschlüssel nimmt dir die Rechnerei ab. Erstelle schnell und unkompliziert Notenskalen für das österreichische Notensystem (1–5), auch wenn sie einen Knick haben.

Kein Login, keine Datenbank, nichts wird gespeichert. Die Berechnung läuft client-only im Browser.

[![Go Version](https://img.shields.io/badge/Go-1.25-00ADD8?style=flat-square&logo=go)](https://golang.org)
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

Browser auf `http://localhost:5173` – fertig. Produktionsbuild:

```bash
npm run build
go run main.go
```

Der Go-Dienst liefert dann nur statische Assets aus `dist` aus. Oder mit Docker:

```bash
docker compose up
```

### CSV-Format

```csv
Name,Punkte
Max Mustermann,85.5
Anna Schmidt,76.0
Tom Weber,92.5
```

Punkt als Dezimaltrennzeichen. Semikolon als Spaltentrenner geht auch. Max. 10 MB.

## Unter der Haube

Frontend: Vanilla TypeScript (Vite).  
Backend: Minimaler Go-Static-Server (Health + Asset-Auslieferung).

```
notenschluessel/
├── main.go                    # Minimaler Static-Server
├── src/                       # Client-only TypeScript App
├── tests/                     # Vitest Unit/Integrationstests
├── index.html                 # App Einstieg
├── privacy.html               # Datenschutzseite
├── pkg/
│   ├── calculator/            # Legacy Referenzlogik (Go)
│   ├── downloads/             # Legacy Exportlogik (Go)
│   ├── handlers/              # Legacy Handler (Go)
│   ├── logging/               # Strukturiertes Logging (slog/JSON)
│   ├── models/                # Datentypen
│   ├── security/              # Rate Limiting, IP-Extraktion
│   └── session/               # In-Memory Session Store
├── templates/                 # Legacy Templates (Go)
├── dockerfile
└── compose.yml
```

Dependencies:
- [xlsx](https://www.npmjs.com/package/xlsx) für Excel-Export im Browser
- [excelize](https://github.com/xuri/excelize) für Excel-Export
- [golang.org/x/time/rate](https://pkg.go.dev/golang.org/x/time/rate) für Rate Limiting

## Sicherheit

Client-only Datenverarbeitung reduziert Datenschutzrisiko deutlich:

- Keine serverseitige Verarbeitung von Schülerdaten
- Keine Nutzdaten-POST-Endpunkte
- Security Headers (CSP, HSTS, X-Frame-Options, Permissions-Policy)
- CSV-Injection-Schutz bei Exporten
- Lokale Daten können in der App direkt gelöscht werden

## Konfiguration

| Variable   | Werte                        | Beschreibung                        |
| ---------- | ---------------------------- | ----------------------------------- |
| `ENV`      | `production` / `development` | Steuert HSTS Header                 |
| `STATIC_DIR` | z.B. `dist` / `/app/dist`  | Verzeichnis für statische Assets    |

Im Development läuft die TS-App via Vite (`localhost:5173`), in Production statisch via Go (`localhost:8080`).

## Tests

```bash
npm run test            # TS Unit + Integration
go test ./... -v        # Legacy Go Tests
```

## Docker

Multi-Stage Build: Node für Frontend-Build, Go für Static-Binary, Runtime in `scratch`. Health Check über `/healthz`:

```bash
./notenschluessel --health-check
```

Braucht kein curl im Container – der Binary prüft sich selbst.

## Für Entwickler

- TS-Fachlogik in `src/` ist die aktive App-Implementierung
- Go-Server dient nur dem statischen Ausliefern von `dist`
- Legacy-Go-Code bleibt vorerst als Referenz erhalten

## Lizenz

Apache 2.0 – siehe [LICENSE](LICENSE).
