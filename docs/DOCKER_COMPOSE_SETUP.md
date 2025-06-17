# Docker Compose Konfiguration

## 🚀 Standard Deployment (ohne Bug-Reports)

```bash
# Einfacher Start ohne GitHub-Integration
docker compose up -d
```

Die Anwendung läuft auf `http://localhost:8080` mit vollem Funktionsumfang für Notenschlüssel-Berechnungen. Der Bug-Report-Button wird nicht angezeigt.

## 🐛 Mit Bug-Report-Funktionalität

### Schritt 1: Environment Datei erstellen
```bash
# Beispiel-Datei kopieren
cp .env.example .env
```

### Schritt 2: GitHub-Konfiguration in .env
```bash
# .env Datei bearbeiten
nano .env
```

Konfiguration:
```bash
# GitHub Integration aktivieren
GITHUB_TOKEN=ghp_your_personal_access_token_here
GITHUB_REPO=your-username/notenschluessel
```

### Schritt 3: Compose-Datei anpassen
```yaml
# In compose.yml die Zeilen uncommentieren:
env_file:
  - .env
```

### Schritt 4: Anwendung starten
```bash
# Mit GitHub-Integration starten
docker compose up -d
```

## 📋 GitHub Personal Access Token erstellen

1. **GitHub.com besuchen** → Settings → Developer settings → Personal access tokens → Tokens (classic)
2. **"Generate new token (classic)"** klicken
3. **Name vergeben:** z.B. "Notenschlüssel Bug Reports"
4. **Expiration:** Nach Bedarf (90 days empfohlen)
5. **Scopes auswählen:**
   - ✅ `repo` (für private Repositories)
   - ✅ `public_repo` (nur für öffentliche Repositories)
6. **Token generieren** und sofort kopieren (wird nur einmal angezeigt!)

## 🔒 Sicherheitshinweise

- ✅ **`.env` Datei nie committen** (steht in `.gitignore`)
- ✅ **Token regelmäßig erneuern** (alle 90 Tage empfohlen)
- ✅ **Minimale Berechtigungen** verwenden (nur `issues:write` notwendig)
- ✅ **Token sicher aufbewahren** (Passwort-Manager)

## 🧪 Testen der Konfiguration

```bash
# Container logs prüfen
docker compose logs notenschluessel

# API-Test
curl -X POST http://localhost:8080/api/bug-report \
  -H "Content-Type: application/json" \
  -d '{"title":"Test","description":"Test"}'

# Erwartete Antworten:
# Ohne GitHub: {"success":false,"message":"Bug-Report-Funktion ist nicht verfügbar"}
# Mit GitHub: {"success":true,"message":"Bug-Report wurde erfolgreich übermittelt. Vielen Dank!"}
```

## 🔧 Troubleshooting

### Problem: Bug-Report-Button nicht sichtbar
**Lösung:**
1. `.env` Datei prüfen
2. `env_file` in `compose.yml` uncommentieren
3. Container neu starten: `docker compose restart`

### Problem: "Bug-Report-Funktion ist nicht verfügbar"
**Ursachen:**
- Environment Variables nicht gesetzt
- Falscher Token oder Repository-Name
- Token ohne ausreichende Berechtigungen

### Problem: "Failed to create GitHub issue"
**Ursachen:**
- Token abgelaufen
- Repository existiert nicht
- Keine `issues:write` Berechtigung

## 📦 Alternative: Direct Environment

Statt `.env` Datei können Environment Variables auch direkt gesetzt werden:

```yaml
# In compose.yml
environment:
  - GITHUB_TOKEN=ghp_your_token_here
  - GITHUB_REPO=your-username/notenschluessel
```

**⚠️ Nicht empfohlen für Production** da sensible Daten in der Compose-Datei stehen.
