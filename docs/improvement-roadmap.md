# Roadmap für Verbesserungen (Notenschlüssel)

Dieses Dokument dient als Sammlung und Priorisierung von Verbesserungsvorschlägen für die Notenschlüssel-Applikation. Es soll helfen, geplante Features und technische Optimierungen nach Aufwand und Nutzen zu bewerten.

## 1. User Experience (UX) & Features
*Ziel: Die App intuitiver und nützlicher für den täglichen Einsatz im Unterricht machen.*

| Feature                                | Beschreibung                                                                              | Priorität | Aufwand |
| :------------------------------------- | :---------------------------------------------------------------------------------------- | :-------- | :------ |
| **Visualisierung der Notenverteilung** | Diagramm (z.B. Chart.js) zur Anzeige, wie viele Schüler welche Note erhalten haben.       | Hoch      | Mittel  |
| **Dark Mode**                          | Unterstützung für dunkle Themes zur Schonung der Augen in dunklen Räumen.                 | Mittel    | Gering  |
| **Persistente Einstellungen**          | Speichern von Standardwerten (z.B. `maxPoints`) via `localStorage`.                       | Mittel    | Gering  |
| **Smart CSV-Parser**                   | Automatische Erkennung von Trennzeichen und Spaltennamen zur Fehlerreduktion beim Import. | Hoch      | Mittel  |

## 2. Code & Architektur
*Ziel: Wartbarkeit, Robustheit und Performance optimieren.*

| Feature                            | Beschreibung                                                                                  | Priorität | Aufwand |
| :--------------------------------- | :-------------------------------------------------------------------------------------------- | :-------- | :------ |
| **Detailliertere Fehlermeldungen** | Einführung spezifischer Error-Typen für Validierung und Parsing statt generischer Meldungen.  | Mittel    | Gering  |
| **Dokumentation (JSDoc)**          | Ausführliche mathematische Dokumentation der Berechnungslogik in `src/core/calculator.ts`.    | Hoch      | Gering  |
| **Web Worker**                     | Auslagerung der Berechnung in einen Web Worker bei sehr großen Datensätzen zur UI-Entlastung. | Niedrig   | Mittel  |

## 3. Barrierefreiheit (Accessibility)
*Ziel: Inklusion und Bedienbarkeit für alle Nutzer sicherstellen.*

| Feature                   | Beschreibung                                                                            | Priorität | Aufwand |
| :------------------------ | :-------------------------------------------------------------------------------------- | :-------- | :------ |
| **A11y-Audit & Kontrast** | Sicherstellung ausreichender Farbkontraste (besonders bei Notenfarben) und ARIA-Labels. | Hoch      | Mittel  |

## 4. DevOps & Qualitätssicherung
*Ziel: Stabilität durch automatisierte Prozesse gewährleisten.*

| Feature                    | Beschreibung                                                                            | Priorität | Aufwand |
| :------------------------- | :-------------------------------------------------------------------------------------- | :-------- | :------ |
| **CI/CD Pipeline**         | Automatisierte Tests (`npm run test`) und Typ-Checks bei jedem Push via GitHub Actions. | Hoch      | Gering  |
| **E2E-Tests (Playwright)** | Testen des kritischen Workflows: Import $\rightarrow$ Berechnung $\rightarrow$ Export.  | Mittel    | Mittel  |

---
*Stand: 2026-06-21*
