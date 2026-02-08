# Notenschlüssel

Ein simples Tool, für Lehrerinnen und Lehrer entwickelt. Notenschlüssel nimmt dir die Rechnerei ab. Erstelle schnell und unkompliziert Notenskalen für das österreichische Notensystem (1–5), auch wenn sie einen Knick haben.

Kein Login, keine Datenbank, nichts wird gespeichert. Einfach Punkte eingeben, fertig.

[![Go Version](https://img.shields.io/badge/Go-1.25-00ADD8?style=flat-square&logo=go)](https://golang.org)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg?style=flat-square)](LICENSE)

## So geht's

1. Maximale Punktzahl, Schrittweite und Knickpunkt eingeben
2. Optional eine CSV-Datei mit Schülernamen und Punkten hochladen
3. Ergebnisse ansehen, als CSV oder Excel runterladen

Das war's schon.

## Loslegen

```bash
go run main.go
```

Browser auf `http://localhost:8080` – fertig. Oder mit Docker:

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

Geschrieben in Go 1.25, keine Frameworks, fast nur Stdlib. Ein einzelner HTTP-Service, der alles macht.

```
notenschluessel/
├── main.go                    # Server, Middleware, Routen
├── pkg/
│   ├── calculator/            # Notenberechnung, CSV-Parsing
│   ├── downloads/             # CSV/Excel-Export
│   ├── handlers/              # HTTP-Handler
│   ├── logging/               # Strukturiertes Logging (slog/JSON)
│   ├── models/                # Datentypen
│   ├── security/              # Rate Limiting, IP-Extraktion
│   └── session/               # In-Memory Session Store
├── templates/
├── dockerfile
└── compose.yml
```

Zwei externe Dependencies:
- [excelize](https://github.com/xuri/excelize) für Excel-Export
- [golang.org/x/time/rate](https://pkg.go.dev/golang.org/x/time/rate) für Rate Limiting

## Sicherheit

Auch wenn es nur ein Notenschlüssel ist – wenn man das Ding ins Netz stellt, sollte es ordentlich abgesichert sein:

- CSRF-Schutz über Go 1.25 nativ (`http.NewCrossOriginProtection()`)
- Rate Limiting pro IP (10 req/min, Burst 20)
- Security Headers (CSP, HSTS, X-Frame-Options, Permissions-Policy)
- Session-Cookies: HttpOnly, SameSite=Strict
- CSV-Injection-Schutz bei Exporten
- Reverse-Proxy-Support (X-Forwarded-For, X-Real-IP)

Keine Datenbank, keine Persistenz. Sessions leben im Memory und laufen nach 24h ab. Schülerdaten werden nur für die Berechnung verwendet und nie gespeichert. DSGVO-konform.

## Konfiguration

| Variable   | Werte                        | Beschreibung                        |
| ---------- | ---------------------------- | ----------------------------------- |
| `ENV`      | `production` / `development` | Steuert HSTS, CSRF-Origins, Logging |
| `HOSTNAME` | z.B. `noten.example.com`     | Trusted Origin für CSRF             |

Im Development geht `localhost:8080`, in Production nur der konfigurierte Hostname über HTTPS.

## Tests

```bash
go test ./... -v        # alle Tests
go test ./... -race     # mit Race-Detector
go test ./... -cover    # mit Coverage
```

## Docker

Multi-Stage Build nach `scratch`, ~15 MB Image, non-root User. Health Check über `/healthz`:

```bash
./notenschluessel --health-check
```

Braucht kein curl im Container – der Binary prüft sich selbst.

## Für Entwickler

- Middleware (Security Headers, CSRF, Rate Limiting) liegt einmal auf dem gesamten Router – neue Endpunkte sind automatisch geschützt
- Logging über das `logging`-Package, nicht `fmt.Println`
- Keine CSRF-Tokens in Formulare – Go 1.25 macht das über Browser-Header

## Lizenz

Apache 2.0 – siehe [LICENSE](LICENSE).
