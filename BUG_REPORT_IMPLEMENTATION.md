# Bug-Report-System Implementierung

## ✅ Was wurde implementiert

### 1. **Backend (Go)**
- **Neue Strukturen:**
  - `BugReport` - Sammelt alle Bug-Report-Daten
  - `BugReportResponse` - API-Response-Format
  - `GitHubClient` - Für GitHub API Integration

- **Neuer API-Endpoint:** `/api/bug-report`
  - Akzeptiert POST-Requests mit JSON
  - Validiert Eingabedaten
  - Erstellt automatisch GitHub Issues (wenn konfiguriert)
  - Fallback zu Console-Logging

### 2. **Frontend (HTML/CSS/JavaScript)**
- **Modal-Dialog** mit professionellem Design
- **Strukturiertes Formular** mit allen notwendigen Feldern
- **Auto-Detection** von Browser und Betriebssystem
- **Loading-States** und Fehlerbehandlung
- **Responsive Design** für mobile Geräte

## 🔧 Konfiguration

### Environment Variables setzen:
```bash
export GITHUB_TOKEN="ghp_your_personal_access_token_here"
export GITHUB_REPO="your-username/notenschluessel"
```

### GitHub Personal Access Token erstellen:
1. GitHub.com → Settings → Developer settings → Personal access tokens
2. "Generate new token (classic)"
3. Scopes: `repo` (oder nur `public_repo` für öffentliche Repos)
4. Token kopieren und als `GITHUB_TOKEN` setzen

## 🚀 Verwendung

### Für Benutzer:
1. Klick auf den roten "🐛 Fehler melden" Button (rechts unten)
2. Modal öffnet sich mit strukturiertem Formular
3. Ausfüllen der Felder (Titel und Beschreibung sind Pflicht)
4. Browser/OS werden automatisch erkannt
5. "Bug-Report senden" klicken
6. Erfolgs-/Fehlermeldung erscheint

### Für Entwickler:
- Bug-Reports erscheinen automatisch als GitHub Issues
- Issues sind gelabelt mit `bug` und `user-report`
- Strukturierte Formatierung für einfache Bearbeitung

## 📋 Formular-Felder

**Pflichtfelder:**
- Titel
- Fehlerbeschreibung

**Optionale Felder:**
- Schritte zur Reproduktion
- Erwartetes Verhalten
- Browser (auto-erkannt)
- Betriebssystem (auto-erkannt)
- Eingabedaten (Max. Punkte, Schrittweite, Knickpunkt, CSV-Verwendung)
- Zusätzliche Informationen

## 🛠️ Technische Details

### API-Endpoint: `/api/bug-report`
```json
POST /api/bug-report
Content-Type: application/json

{
  "title": "Beispiel Bug",
  "description": "Detaillierte Beschreibung...",
  "steps": "1. Schritt 1\n2. Schritt 2",
  "expected": "Erwartetes Verhalten",
  "browser": "Chrome",
  "os": "Windows 10",
  "maxPoints": "50",
  "minPoints": "0.5",
  "breakPoint": "50",
  "csvUsed": "Ja",
  "additionalInfo": "Weitere Details..."
}
```

### Response:
```json
{
  "success": true,
  "message": "Bug-Report wurde erfolgreich übermittelt. Vielen Dank!"
}
```

## 🔒 Sicherheit

- **Keine sensiblen Daten** im Frontend
- **GitHub Token** nur im Backend als Environment Variable
- **Input Validation** auf Server-Seite
- **CORS Headers** für Frontend-Integration
- **Error Handling** ohne Informations-Leakage

## 🎯 Fallback-Verhalten

Wenn GitHub nicht konfiguriert ist:

- Bug-Reports werden in Console geloggt
- Benutzer erhält trotzdem Erfolgs-Bestätigung
- Entwickler kann Logs manuell auswerten

## 🧪 Testen

1. Server starten: `go run main.go`
2. Browser öffnen: `http://localhost:8080`
3. Bug-Report-Button klicken
4. Formular ausfüllen und testen

## 📦 Docker-Integration

Die Environment Variables können in `docker-compose.yml` oder beim Docker-Run gesetzt werden:

```yaml
# docker-compose.yml
environment:
  - GITHUB_TOKEN=ghp_your_token_here
  - GITHUB_REPO=your-username/notenschluessel
```

```bash
# Docker run
docker run -e GITHUB_TOKEN=ghp_xxx -e GITHUB_REPO=user/repo notenschluessel
```
