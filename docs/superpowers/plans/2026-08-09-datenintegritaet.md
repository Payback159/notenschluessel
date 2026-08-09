# Datenintegrität, Anzeige und Export — Implementierungsplan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Die App meldet jedes verworfene oder verdächtige Datum sichtbar, statt still ein falsches Ergebnis zu liefern, und zeigt Punktzahlen überall identisch an.

**Architecture:** Ein strukturierter Diagnose-Kanal (`Diagnostic` mit `severity`, `code`, `row`) ersetzt die heutigen `errors: string[]` und `skippedRows` in Parsern, Validierung und Workflow. Warnungen lassen die Berechnung durchlaufen, Fehler brechen sie ab. Zahlenformatierung wandert in ein reines Modul, das Bildschirm und CSV-Export gemeinsam nutzen.

**Tech Stack:** TypeScript 6 (strict, `noUncheckedIndexedAccess`), Vite 8, Vitest 4, `xlsx-js-style`. Keine Frameworks, kein Backend.

## Global Constraints

- **Sprache der Oberfläche:** Deutsch, mit korrekten Umlauten. Neue Meldungstexte nie mit „ue"/„ae"/„oe" umschreiben.
- **Client-only:** Keine Schülerdaten an einen Server. Kein `fetch` mit Nutzdaten, in keiner Task.
- **TypeScript strict:** `noUncheckedIndexedAccess` ist aktiv — jeder Index-Zugriff liefert `T | undefined` und muss geprüft werden.
- **Diagnose-Codes:** Ausschließlich die zwölf im Spec definierten Codes. Kein Code wird erfunden, keiner umbenannt.
- **`row` ist 1-basiert** und bezeichnet die Zeilennummer der Quelldatei **einschließlich Kopfzeile** — also die Zeile, die die Lehrkraft im Editor sieht.
- **Zahlendarstellung:** Bildschirm und CSV verwenden Dezimalkomma. XLSX behält numerische Zellen.
- **Tests:** Jede Task endet mit grünem `npm run test` **und** `npm run type-check`.
- **Commits:** Pro Task ein Commit. Branch: `docs/datenintegritaet-spec` oder ein davon abgezweigter Feature-Branch.

**Spec:** `docs/superpowers/specs/2026-08-09-datenintegritaet-design.md`

---

## File Structure

**Neu:**

| Datei | Verantwortung |
| :--- | :--- |
| `src/core/diagnostics.ts` | `Diagnostic`-Factories, je eine pro Code, samt deutschem Meldungstext. Einzige Quelle für Diagnose-Texte. |
| `src/core/sanitize.ts` | `sanitizeName`, heute in beiden Parsern dupliziert. |
| `src/core/studentValidation.ts` | `validateStudentPoints` — Prüfung gegen die eingegebene Maximalpunktzahl. |
| `src/ui/format.ts` | Reine Zahlendarstellung, kein DOM. |
| `src/ui/render.ts` | DOM-Ausgabe von Tabellen, Durchschnitt und Diagnoseliste. |

**Geändert:**

| Datei | Änderung |
| :--- | :--- |
| `src/types.ts` | `Diagnostic`, `Severity`, `DiagnosticCode`; `Student.sourceRow`; Umbau der vier Result-Typen. |
| `src/parsers/csvParser.ts` | Diagnosen statt `errors`/`skippedRows`; Zeilenlimit als Warnung. |
| `src/parsers/manualParser.ts` | Diagnosen; Zeilenlimit vereinheitlicht. |
| `src/core/validation.ts` | Diagnosen; `validateInputMode` ersetzt `validateInputModeExclusivity`. |
| `src/core/calculator.ts` | `validateGradeBounds` liefert Diagnosen. |
| `src/ui/workflow.ts` | Aggregation, `ok` aus Severity, neue Stufenreihenfolge. |
| `src/export/csvExport.ts` | Semikolon, Dezimalkomma, korrigierte Feldbereinigung, BOM. |
| `src/app.ts` | Ausgedünnt auf Events, State, Export-Verdrahtung. |
| `index.html` | `step="any"`, `role="alert"`, Diagnose-Container. |
| `style.css` | Klassen für die Diagnoseliste. |

**Warum `Student.sourceRow`:** Akzeptanzkriterium 1 verlangt, dass `points-exceed-max` die betroffenen Zeilen nennt. Diese Prüfung läuft nach dem Parsen — ohne mitgeführte Zeilennummer wäre die Zeile dort nicht mehr bekannt. Das Feld ist optional, damit Testdaten und `processStudents` es weglassen können.

---

## Task 1: Diagnose-Typ und Factories

**Files:**
- Modify: `src/types.ts`
- Create: `src/core/diagnostics.ts`
- Test: `tests/unit/diagnostics.test.ts`

**Interfaces:**
- Consumes: nichts
- Produces: `Severity`, `DiagnosticCode`, `Diagnostic` (aus `types.ts`); aus `diagnostics.ts` die Factories `invalidCoreInput`, `degenerateScale`, `inputModeConflict`, `inputModeMismatch`, `pointsExceedMax`, `noValidStudents`, `rowTooFewColumns`, `rowMissingName`, `rowMissingPoints`, `rowUnparsablePoints`, `rowNegativePoints`, `rowLimitReached` sowie `hasErrors(diagnostics: Diagnostic[]): boolean`.

- [ ] **Step 1: Write the failing test**

Create `tests/unit/diagnostics.test.ts`:

```ts
import { describe, expect, it } from "vitest";
import {
    hasErrors,
    inputModeMismatch,
    pointsExceedMax,
    rowLimitReached,
    rowUnparsablePoints
} from "../../src/core/diagnostics";

describe("diagnostics", () => {
    it("marks row problems as warnings and carries the row number", () => {
        const d = rowUnparsablePoints(7, "achtzig");
        expect(d.severity).toBe("warning");
        expect(d.code).toBe("row-unparsable-points");
        expect(d.row).toBe(7);
        expect(d.message).toContain("Zeile 7");
        expect(d.message).toContain("achtzig");
    });

    it("marks points above the maximum as an error", () => {
        const d = pointsExceedMax(4, 855, 100);
        expect(d.severity).toBe("error");
        expect(d.code).toBe("points-exceed-max");
        expect(d.row).toBe(4);
        expect(d.message).toContain("855");
        expect(d.message).toContain("100");
    });

    it("reports a mode mismatch without a row number", () => {
        const d = inputModeMismatch("csv");
        expect(d.severity).toBe("error");
        expect(d.code).toBe("input-mode-mismatch");
        expect(d.row).toBeUndefined();
    });

    it("reports the row limit as a warning", () => {
        const d = rowLimitReached(10000);
        expect(d.severity).toBe("warning");
        expect(d.code).toBe("row-limit-reached");
        expect(d.message).toContain("10000");
    });

    it("detects whether a list contains errors", () => {
        expect(hasErrors([rowUnparsablePoints(1, "x")])).toBe(false);
        expect(hasErrors([rowUnparsablePoints(1, "x"), pointsExceedMax(2, 200, 100)])).toBe(true);
        expect(hasErrors([])).toBe(false);
    });

    it("uses correct German umlauts in messages", () => {
        expect(rowMissingNameMessageHasUmlaut()).toBe(true);
    });
});

function rowMissingNameMessageHasUmlaut(): boolean {
    // Guards against the "Ungueltige" style that exists in the current codebase.
    return !/ue[a-z]|ae[a-z]|oe[a-z]/.test(pointsExceedMax(1, 2, 1).message);
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `npx vitest run tests/unit/diagnostics.test.ts`
Expected: FAIL — `Failed to resolve import "../../src/core/diagnostics"`

- [ ] **Step 3: Add the types**

In `src/types.ts`, add above the existing interfaces:

```ts
export type Severity = "error" | "warning";

export type DiagnosticCode =
    | "invalid-core-input"
    | "degenerate-scale"
    | "input-mode-conflict"
    | "input-mode-mismatch"
    | "points-exceed-max"
    | "no-valid-students"
    | "row-too-few-columns"
    | "row-missing-name"
    | "row-missing-points"
    | "row-unparsable-points"
    | "row-negative-points"
    | "row-limit-reached";

export interface Diagnostic {
    severity: Severity;
    code: DiagnosticCode;
    /** 1-based line number of the source file, including the header row. */
    row?: number;
    message: string;
}
```

Also extend `Student`:

```ts
export interface Student {
    name: string;
    points: number;
    grade?: number;
    /** 1-based source line, set by the parsers. Absent for hand-built test data. */
    sourceRow?: number;
}
```

- [ ] **Step 4: Write the factories**

Create `src/core/diagnostics.ts`:

```ts
import { Diagnostic, InputMode } from "../types";

export function hasErrors(diagnostics: Diagnostic[]): boolean {
    return diagnostics.some((d) => d.severity === "error");
}

export function invalidCoreInput(message: string): Diagnostic {
    return { severity: "error", code: "invalid-core-input", message };
}

export function degenerateScale(): Diagnostic {
    return {
        severity: "error",
        code: "degenerate-scale",
        message:
            "Diese Kombination aus maximaler Punktzahl, Schrittweite und Knickpunkt ergibt keine gültige Notenskala."
    };
}

export function inputModeConflict(): Diagnostic {
    return {
        severity: "error",
        code: "input-mode-conflict",
        message:
            "Bitte entweder CSV-Import oder manuelle Eingabe verwenden. Eine Kombination ist nicht erlaubt."
    };
}

export function inputModeMismatch(selected: InputMode): Diagnostic {
    const other = selected === "csv" ? "der manuellen Tabelle" : "der CSV-Datei";
    const chosen = selected === "csv" ? "CSV-Import" : "manuelle Eingabe";
    return {
        severity: "error",
        code: "input-mode-mismatch",
        message: `Es ist ${chosen} ausgewählt, die Daten stehen aber in ${other}. Bitte den Eingabemodus wechseln.`
    };
}

export function pointsExceedMax(row: number, points: number, maxPoints: number): Diagnostic {
    return {
        severity: "error",
        code: "points-exceed-max",
        row,
        message: `Zeile ${row}: ${points} Punkte liegen über der Maximalpunktzahl von ${maxPoints}.`
    };
}

export function noValidStudents(): Diagnostic {
    return {
        severity: "error",
        code: "no-valid-students",
        message: "Keine gültigen Schülerdaten gefunden."
    };
}

export function rowTooFewColumns(row: number): Diagnostic {
    return {
        severity: "warning",
        code: "row-too-few-columns",
        row,
        message: `Zeile ${row}: übersprungen, weniger als zwei Spalten.`
    };
}

export function rowMissingName(row: number): Diagnostic {
    return {
        severity: "warning",
        code: "row-missing-name",
        row,
        message: `Zeile ${row}: übersprungen, Name fehlt.`
    };
}

export function rowMissingPoints(row: number): Diagnostic {
    return {
        severity: "warning",
        code: "row-missing-points",
        row,
        message: `Zeile ${row}: übersprungen, Punkte fehlen.`
    };
}

export function rowUnparsablePoints(row: number, raw: string): Diagnostic {
    return {
        severity: "warning",
        code: "row-unparsable-points",
        row,
        message: `Zeile ${row}: übersprungen, „${raw}" ist keine Punktzahl.`
    };
}

export function rowNegativePoints(row: number, points: number): Diagnostic {
    return {
        severity: "warning",
        code: "row-negative-points",
        row,
        message: `Zeile ${row}: übersprungen, ${points} Punkte sind negativ.`
    };
}

export function rowLimitReached(limit: number): Diagnostic {
    return {
        severity: "warning",
        code: "row-limit-reached",
        message: `Nur die ersten ${limit} Zeilen wurden gelesen, weitere wurden ignoriert.`
    };
}
```

- [ ] **Step 5: Run tests and type-check**

Run: `npx vitest run tests/unit/diagnostics.test.ts && npm run type-check`
Expected: PASS, keine Typfehler. Die übrigen Tests sind noch unberührt.

- [ ] **Step 6: Commit**

```bash
git add src/types.ts src/core/diagnostics.ts tests/unit/diagnostics.test.ts
git commit -m "feat: Diagnose-Typ mit Severity und Meldungs-Factories"
```

---

## Task 2: CSV-Parser auf Diagnosen umstellen

**Files:**
- Modify: `src/parsers/csvParser.ts`
- Modify: `src/types.ts` (`CSVParseResult`)
- Test: `tests/unit/csvParser.test.ts`

**Interfaces:**
- Consumes: Factories aus Task 1.
- Produces: `parseCSVContent(content: string): CSVParseResult` mit `{ students: Student[]; diagnostics: Diagnostic[] }`. Jeder gelieferte `Student` trägt `sourceRow`.

Behebt B2 (verworfene Zeilen werden gemeldet) und vereinheitlicht das Zeilenlimit.

- [ ] **Step 1: Rewrite the test file**

Replace `tests/unit/csvParser.test.ts` completely:

```ts
import { describe, expect, it } from "vitest";
import { detectDelimiter, parseCSVContent } from "../../src/parsers/csvParser";
import { LIMITS } from "../../src/constants";

describe("csv parser", () => {
    it("detects comma delimiter", () => {
        expect(detectDelimiter("Name,Punkte\nAlice,12")).toBe(",");
    });

    it("detects semicolon delimiter", () => {
        expect(detectDelimiter("Name;Punkte\nAlice;12")).toBe(";");
    });

    it("parses valid csv and records the source row", () => {
        const result = parseCSVContent("Name,Punkte\nAlice,95\nBob,42.5");
        expect(result.diagnostics).toHaveLength(0);
        expect(result.students).toHaveLength(2);
        expect(result.students[0]).toEqual({ name: "Alice", points: 95, sourceRow: 2 });
        expect(result.students[1]?.sourceRow).toBe(3);
    });

    it("warns about every skipped row instead of staying silent", () => {
        const result = parseCSVContent("Name,Punkte\nAlice,95\nKaputt\nBob,abc\nCharlie,20");
        expect(result.students).toHaveLength(2);

        const codes = result.diagnostics.map((d) => d.code);
        expect(codes).toContain("row-too-few-columns");
        expect(codes).toContain("row-unparsable-points");
        expect(result.diagnostics.every((d) => d.severity === "warning")).toBe(true);

        const rows = result.diagnostics.map((d) => d.row);
        expect(rows).toContain(3);
        expect(rows).toContain(4);
    });

    it("warns about a missing name", () => {
        const result = parseCSVContent("Name,Punkte\n,50");
        expect(result.diagnostics.some((d) => d.code === "row-missing-name" && d.row === 2)).toBe(true);
    });

    it("warns about negative points", () => {
        const result = parseCSVContent("Name,Punkte\nAlice,-5\nBob,10");
        expect(result.diagnostics.some((d) => d.code === "row-negative-points" && d.row === 2)).toBe(true);
        expect(result.students).toHaveLength(1);
    });

    it("returns an error diagnostic if no valid rows exist", () => {
        const result = parseCSVContent("Name,Punkte\nAlice,abc");
        expect(result.students).toHaveLength(0);
        expect(result.diagnostics.some((d) => d.code === "no-valid-students")).toBe(true);
    });

    it("returns an error diagnostic for empty input", () => {
        const result = parseCSVContent("   ");
        expect(result.students).toHaveLength(0);
        expect(result.diagnostics.some((d) => d.code === "no-valid-students")).toBe(true);
    });

    it("warns once when the row limit is reached, keeping the rows read so far", () => {
        const rows = Array.from({ length: LIMITS.maxStudents + 5 }, (_, i) => `S${i},10`).join("\n");
        const result = parseCSVContent(`Name,Punkte\n${rows}`);

        expect(result.students).toHaveLength(LIMITS.maxStudents);
        expect(result.diagnostics.filter((d) => d.code === "row-limit-reached")).toHaveLength(1);
        expect(result.diagnostics.some((d) => d.code === "no-valid-students")).toBe(false);
    });

    it("sanitizes names", () => {
        const result = parseCSVContent("Name,Punkte\n<bad>,10");
        expect(result.students[0]?.name).toBe("bad");
    });
});
```

- [ ] **Step 2: Run test to verify it fails**

Run: `npx vitest run tests/unit/csvParser.test.ts`
Expected: FAIL — `result.diagnostics` ist `undefined`.

- [ ] **Step 3: Update the result type**

In `src/types.ts`, replace `CSVParseResult`:

```ts
export interface CSVParseResult {
    students: Student[];
    diagnostics: Diagnostic[];
}
```

- [ ] **Step 4: Rewrite the parser body**

In `src/parsers/csvParser.ts`, replace the imports and `parseCSVContent`. `sanitizeName` and `parseCSVLine` stay unchanged for now (Task 12 moves `sanitizeName`).

```ts
import { LIMITS } from "../constants";
import {
    noValidStudents,
    rowLimitReached,
    rowMissingName,
    rowNegativePoints,
    rowTooFewColumns,
    rowUnparsablePoints
} from "../core/diagnostics";
import { CSVParseResult, Diagnostic, Student } from "../types";

export function parseCSVContent(content: string): CSVParseResult {
    const students: Student[] = [];
    const diagnostics: Diagnostic[] = [];

    if (content.trim() === "") {
        return { students: [], diagnostics: [noValidStudents()] };
    }

    const delimiter = detectDelimiter(content);
    const lines = content.split(/\r?\n/);
    let limitReached = false;

    for (let index = 0; index < lines.length; index++) {
        const line = lines[index];
        if (line === undefined || line.trim() === "") {
            continue;
        }

        // 1-based line number as the teacher sees it in an editor.
        const row = index + 1;
        const record = parseCSVLine(line, delimiter);
        const col0 = record[0];
        const col1 = record[1];

        if (index === 0 && col0 !== undefined && col0.trim().toLowerCase() === "name") {
            continue;
        }

        if (record.length < 2 || col0 === undefined || col1 === undefined) {
            diagnostics.push(rowTooFewColumns(row));
            continue;
        }

        const name = col0.trim();
        const rawPoints = col1.trim();

        if (name === "" && rawPoints === "") {
            continue;
        }

        if (name === "") {
            diagnostics.push(rowMissingName(row));
            continue;
        }

        const points = Number.parseFloat(rawPoints.replaceAll(",", "."));

        if (!Number.isFinite(points)) {
            diagnostics.push(rowUnparsablePoints(row, rawPoints));
            continue;
        }

        if (points < 0) {
            diagnostics.push(rowNegativePoints(row, points));
            continue;
        }

        if (points > LIMITS.maxPoints) {
            diagnostics.push(rowUnparsablePoints(row, rawPoints));
            continue;
        }

        if (students.length >= LIMITS.maxStudents) {
            limitReached = true;
            break;
        }

        students.push({ name: sanitizeName(name), points, sourceRow: row });
    }

    if (limitReached) {
        diagnostics.push(rowLimitReached(LIMITS.maxStudents));
    }

    if (students.length === 0 && !limitReached) {
        diagnostics.push(noValidStudents());
    }

    return { students, diagnostics };
}
```

Note the ordering change: the limit is now checked **before** pushing, so exactly `LIMITS.maxStudents` students are kept and the warning fires once.

- [ ] **Step 5: Run tests and type-check**

Run: `npx vitest run tests/unit/csvParser.test.ts && npm run type-check`
Expected: PASS. `npm run test` insgesamt schlägt noch fehl — `workflow.ts` liest `result.errors`. Das ist erwartet und wird in Task 7 behoben.

- [ ] **Step 6: Commit**

```bash
git add src/types.ts src/parsers/csvParser.ts tests/unit/csvParser.test.ts
git commit -m "feat: CSV-Parser meldet verworfene Zeilen als Diagnosen"
```

---

## Task 3: Manueller Parser auf Diagnosen umstellen

**Files:**
- Modify: `src/parsers/manualParser.ts`
- Modify: `src/types.ts` (`ManualParseResult`)
- Test: `tests/unit/manualParser.test.ts`

**Interfaces:**
- Consumes: Factories aus Task 1.
- Produces: `parseManualEntries(entries: ManualEntry[]): ManualParseResult` mit `{ students, diagnostics }`; `hasNonEmptyManualEntries` bleibt unverändert.

Vereinheitlicht das Zeilenlimit: bisher verwarf dieser Parser bei Überschreitung **alle** Schüler.

- [ ] **Step 1: Rewrite the test file**

Replace `tests/unit/manualParser.test.ts` completely:

```ts
import { describe, expect, it } from "vitest";
import { hasNonEmptyManualEntries, parseManualEntries } from "../../src/parsers/manualParser";
import { LIMITS } from "../../src/constants";

describe("manual parser", () => {
    it("detects whether manual entries contain data", () => {
        expect(hasNonEmptyManualEntries([{ name: "", points: "" }])).toBe(false);
        expect(hasNonEmptyManualEntries([{ name: "Alice", points: "" }])).toBe(true);
    });

    it("parses valid rows and records the source row", () => {
        const result = parseManualEntries([
            { name: "Alice", points: "80" },
            { name: "Bob", points: "45,5" }
        ]);

        expect(result.diagnostics).toHaveLength(0);
        expect(result.students).toEqual([
            { name: "Alice", points: 80, sourceRow: 1 },
            { name: "Bob", points: 45.5, sourceRow: 2 }
        ]);
    });

    it("skips completely empty rows without a diagnostic", () => {
        const result = parseManualEntries([
            { name: "", points: "" },
            { name: "Alice", points: "50" }
        ]);

        expect(result.students).toHaveLength(1);
        expect(result.diagnostics).toHaveLength(0);
    });

    it("warns per problematic row and keeps the valid ones", () => {
        const result = parseManualEntries([
            { name: "", points: "50" },
            { name: "Alice", points: "" },
            { name: "Bob", points: "abc" },
            { name: "Carol", points: "70" }
        ]);

        expect(result.students).toHaveLength(1);
        expect(result.students[0]?.name).toBe("Carol");

        const codes = result.diagnostics.map((d) => d.code);
        expect(codes).toContain("row-missing-name");
        expect(codes).toContain("row-missing-points");
        expect(codes).toContain("row-unparsable-points");
        expect(result.diagnostics.every((d) => d.severity === "warning")).toBe(true);
    });

    it("keeps the rows read so far when the limit is reached", () => {
        const entries = Array.from({ length: LIMITS.maxStudents + 3 }, (_, i) => ({
            name: `S${i}`,
            points: "10"
        }));
        const result = parseManualEntries(entries);

        expect(result.students).toHaveLength(LIMITS.maxStudents);
        expect(result.diagnostics.filter((d) => d.code === "row-limit-reached")).toHaveLength(1);
    });

    it("sanitizes names", () => {
        const result = parseManualEntries([{ name: "<Alice>", points: "10" }]);
        expect(result.students[0]?.name).toBe("Alice");
    });
});
```

- [ ] **Step 2: Run test to verify it fails**

Run: `npx vitest run tests/unit/manualParser.test.ts`
Expected: FAIL — `result.diagnostics` ist `undefined`.

- [ ] **Step 3: Update the result type**

In `src/types.ts`:

```ts
export interface ManualParseResult {
    students: Student[];
    diagnostics: Diagnostic[];
}
```

- [ ] **Step 4: Rewrite the parser body**

In `src/parsers/manualParser.ts`, replace imports and `parseManualEntries`:

```ts
import { LIMITS } from "../constants";
import {
    rowLimitReached,
    rowMissingName,
    rowMissingPoints,
    rowNegativePoints,
    rowUnparsablePoints
} from "../core/diagnostics";
import { Diagnostic, ManualEntry, ManualParseResult, Student } from "../types";

export function parseManualEntries(entries: ManualEntry[]): ManualParseResult {
    const students: Student[] = [];
    const diagnostics: Diagnostic[] = [];
    let limitReached = false;

    for (let i = 0; i < entries.length; i++) {
        const entry = entries[i];
        if (!entry) {
            continue;
        }

        const row = i + 1;
        const name = entry.name.trim();
        const pointsRaw = entry.points.trim();

        if (name === "" && pointsRaw === "") {
            continue;
        }

        if (name === "") {
            diagnostics.push(rowMissingName(row));
            continue;
        }

        if (pointsRaw === "") {
            diagnostics.push(rowMissingPoints(row));
            continue;
        }

        const points = Number.parseFloat(pointsRaw.replaceAll(",", "."));

        if (!Number.isFinite(points)) {
            diagnostics.push(rowUnparsablePoints(row, pointsRaw));
            continue;
        }

        if (points < 0) {
            diagnostics.push(rowNegativePoints(row, points));
            continue;
        }

        if (points > LIMITS.maxPoints) {
            diagnostics.push(rowUnparsablePoints(row, pointsRaw));
            continue;
        }

        if (students.length >= LIMITS.maxStudents) {
            limitReached = true;
            break;
        }

        students.push({ name: sanitizeName(name), points, sourceRow: row });
    }

    if (limitReached) {
        diagnostics.push(rowLimitReached(LIMITS.maxStudents));
    }

    return { students, diagnostics };
}
```

`parseManualEntries` erzeugt bewusst **kein** `no-valid-students` — anders als der CSV-Parser. Der Workflow entscheidet das in Task 7 zentral, weil dort bekannt ist, ob überhaupt eine Eingabe vorlag.

- [ ] **Step 5: Run tests and type-check**

Run: `npx vitest run tests/unit/manualParser.test.ts && npm run type-check`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add src/types.ts src/parsers/manualParser.ts tests/unit/manualParser.test.ts
git commit -m "feat: manueller Parser meldet Diagnosen statt abzubrechen"
```

---

## Task 4: Kernvalidierung und Skalenprüfung auf Diagnosen

**Files:**
- Modify: `src/core/validation.ts`
- Modify: `src/core/calculator.ts` (`validateGradeBounds`)
- Modify: `src/types.ts` (entfernt `ValidationResult`, `GradeBoundsValidationResult`)
- Test: `tests/unit/validation.test.ts`, `tests/unit/calculator.test.ts`

**Interfaces:**
- Consumes: Factories aus Task 1.
- Produces: `validateCoreInputs(maxPoints, minPoints, breakPointPercent): Diagnostic[]`, `validateInputMode(inputMode, csvProvided, manualProvided): Diagnostic[]`, `validateGradeBounds(gradeBounds: GradeBound[]): Diagnostic[]`. Leeres Array bedeutet „gültig".

`validateInputModeExclusivity` entfällt und wird durch `validateInputMode` ersetzt — dieses erkennt zusätzlich B3.

- [ ] **Step 1: Rewrite the validation test**

Replace `tests/unit/validation.test.ts` completely:

```ts
import { describe, expect, it } from "vitest";
import { validateCoreInputs, validateInputMode } from "../../src/core/validation";

describe("core validation", () => {
    it("accepts valid core input values", () => {
        expect(validateCoreInputs(100, 0.5, 50)).toHaveLength(0);
    });

    it("rejects invalid max points", () => {
        const d = validateCoreInputs(0, 0.5, 50);
        expect(d.some((x) => x.code === "invalid-core-input" && x.message.includes("maximale Punktzahl"))).toBe(true);
    });

    it("rejects invalid min points", () => {
        const d = validateCoreInputs(100, -1, 50);
        expect(d.some((x) => x.message.includes("Punkteschrittweite"))).toBe(true);
    });

    it("rejects invalid break point", () => {
        const d = validateCoreInputs(100, 0.5, 100);
        expect(d.some((x) => x.message.includes("Knickpunkt"))).toBe(true);
    });

    it("reports every core problem at once", () => {
        expect(validateCoreInputs(0, -1, 100)).toHaveLength(3);
    });
});

describe("input mode validation", () => {
    it("accepts csv-only in csv mode", () => {
        expect(validateInputMode("csv", true, false)).toHaveLength(0);
    });

    it("accepts manual-only in manual mode", () => {
        expect(validateInputMode("manual", false, true)).toHaveLength(0);
    });

    it("accepts an empty input in either mode", () => {
        expect(validateInputMode("csv", false, false)).toHaveLength(0);
        expect(validateInputMode("manual", false, false)).toHaveLength(0);
    });

    it("rejects combined inputs", () => {
        const d = validateInputMode("manual", true, true);
        expect(d.some((x) => x.code === "input-mode-conflict")).toBe(true);
    });

    it("rejects data sitting in the mode that is not selected", () => {
        const csv = validateInputMode("csv", false, true);
        expect(csv.some((x) => x.code === "input-mode-mismatch")).toBe(true);

        const manual = validateInputMode("manual", true, false);
        expect(manual.some((x) => x.code === "input-mode-mismatch")).toBe(true);
    });
});
```

- [ ] **Step 2: Adjust the calculator test**

In `tests/unit/calculator.test.ts`, replace the `describe("validateGradeBounds", …)` block:

```ts
describe("validateGradeBounds", () => {
    it("accepts valid bounds", () => {
        expect(validateGradeBounds(calculateGradeBounds(100, 0.5, 50))).toHaveLength(0);
    });

    it("rejects insufficient bounds", () => {
        const d = validateGradeBounds([]);
        expect(d.some((x) => x.code === "degenerate-scale")).toBe(true);
    });

    it("rejects inverted ranges", () => {
        const d = validateGradeBounds([
            { grade: 1, lowerBound: 50, upperBound: 40 },
            { grade: 2, lowerBound: 30, upperBound: 39 },
            { grade: 3, lowerBound: 20, upperBound: 29 },
            { grade: 4, lowerBound: 10, upperBound: 19 },
            { grade: 5, lowerBound: 0, upperBound: 9 }
        ]);
        expect(d).toHaveLength(1);
        expect(d[0]?.code).toBe("degenerate-scale");
    });
});
```

Der `describe("calculateGradeBounds", …)`-Block bleibt unverändert — der Algorithmus wird nicht angefasst.

- [ ] **Step 3: Run tests to verify they fail**

Run: `npx vitest run tests/unit/validation.test.ts tests/unit/calculator.test.ts`
Expected: FAIL — `validateInputMode` existiert nicht, `validateGradeBounds` liefert ein Objekt statt eines Arrays.

- [ ] **Step 4: Rewrite validation.ts**

Replace the contents of `src/core/validation.ts`:

```ts
import { LIMITS } from "../constants";
import { inputModeConflict, inputModeMismatch, invalidCoreInput } from "./diagnostics";
import { Diagnostic, InputMode } from "../types";

export function validateCoreInputs(
    maxPoints: number,
    minPoints: number,
    breakPointPercent: number
): Diagnostic[] {
    const diagnostics: Diagnostic[] = [];

    if (!Number.isInteger(maxPoints) || maxPoints <= 0 || maxPoints > LIMITS.maxPoints) {
        diagnostics.push(invalidCoreInput("Ungültige maximale Punktzahl (1-1000 erlaubt)"));
    }

    if (!Number.isFinite(minPoints) || minPoints <= 0 || minPoints > maxPoints) {
        diagnostics.push(invalidCoreInput("Ungültige Punkteschrittweite"));
    }

    if (
        !Number.isFinite(breakPointPercent) ||
        breakPointPercent < LIMITS.minBreakPointPercent ||
        breakPointPercent > LIMITS.maxBreakPointPercent
    ) {
        diagnostics.push(invalidCoreInput("Ungültiger Knickpunkt (1-99% erlaubt)"));
    }

    return diagnostics;
}

export function validateInputMode(
    inputMode: InputMode,
    csvProvided: boolean,
    manualProvided: boolean
): Diagnostic[] {
    if (csvProvided && manualProvided) {
        return [inputModeConflict()];
    }

    const activeProvided = inputMode === "csv" ? csvProvided : manualProvided;
    const otherProvided = inputMode === "csv" ? manualProvided : csvProvided;

    if (!activeProvided && otherProvided) {
        return [inputModeMismatch(inputMode)];
    }

    return [];
}
```

Note the umlaut fix in all three messages (`Ungültige` instead of `Ungueltige`).

- [ ] **Step 5: Rewrite validateGradeBounds**

In `src/core/calculator.ts`, replace the import line and the function. The JSDoc above `calculateGradeBounds` stays untouched.

```ts
import { degenerateScale } from "./diagnostics";
import { Diagnostic, GradeBound } from "../types";
```

```ts
/**
 * Validates the integrity of an array of grade boundaries.
 *
 * Checks that all five grades are present and that no range is inverted or
 * overlaps its neighbour.
 *
 * @param gradeBounds - The array of `GradeBound` objects to validate.
 * @returns An empty array if the scale is usable, otherwise a single error diagnostic.
 */
export function validateGradeBounds(gradeBounds: GradeBound[]): Diagnostic[] {
    if (gradeBounds.length !== 5) {
        return [degenerateScale()];
    }

    for (const bound of gradeBounds) {
        if (bound.upperBound < bound.lowerBound) {
            return [degenerateScale()];
        }
    }

    for (let i = 1; i < gradeBounds.length; i++) {
        const current = gradeBounds[i];
        const previous = gradeBounds[i - 1];
        if (!current || !previous || current.upperBound >= previous.lowerBound) {
            return [degenerateScale()];
        }
    }

    return [];
}
```

- [ ] **Step 6: Remove the obsolete types**

In `src/types.ts`, delete `ValidationResult` and `GradeBoundsValidationResult`.

- [ ] **Step 7: Run tests and type-check**

Run: `npx vitest run tests/unit/validation.test.ts tests/unit/calculator.test.ts && npm run type-check`
Expected: Tests PASS. `npm run type-check` meldet weiterhin Fehler in `workflow.ts` — erwartet, Task 7 behebt sie.

- [ ] **Step 8: Commit**

```bash
git add src/core/validation.ts src/core/calculator.ts src/types.ts tests/unit/validation.test.ts tests/unit/calculator.test.ts
git commit -m "feat: Validierung liefert Diagnosen und erkennt Modus-Verwechslung"
```

---

## Task 5: Prüfung gegen die Maximalpunktzahl

**Files:**
- Create: `src/core/studentValidation.ts`
- Test: `tests/unit/studentValidation.test.ts`

**Interfaces:**
- Consumes: `pointsExceedMax` aus Task 1, `Student.sourceRow` aus Task 2.
- Produces: `validateStudentPoints(students: Student[], maxPoints: number): Diagnostic[]`.

Behebt B1 — der Kern des Reviews.

- [ ] **Step 1: Write the failing test**

Create `tests/unit/studentValidation.test.ts`:

```ts
import { describe, expect, it } from "vitest";
import { validateStudentPoints } from "../../src/core/studentValidation";

describe("validateStudentPoints", () => {
    it("accepts points up to and including the maximum", () => {
        const d = validateStudentPoints(
            [
                { name: "Alice", points: 100, sourceRow: 2 },
                { name: "Bob", points: 0, sourceRow: 3 }
            ],
            100
        );
        expect(d).toHaveLength(0);
    });

    it("rejects a typo above the maximum and names the row", () => {
        const d = validateStudentPoints([{ name: "Tippfehler", points: 855, sourceRow: 4 }], 100);
        expect(d).toHaveLength(1);
        expect(d[0]?.severity).toBe("error");
        expect(d[0]?.code).toBe("points-exceed-max");
        expect(d[0]?.row).toBe(4);
        expect(d[0]?.message).toContain("855");
    });

    it("collects every offending row so they can be fixed in one pass", () => {
        const d = validateStudentPoints(
            [
                { name: "A", points: 855, sourceRow: 2 },
                { name: "B", points: 50, sourceRow: 3 },
                { name: "C", points: 101, sourceRow: 4 }
            ],
            100
        );
        expect(d).toHaveLength(2);
        expect(d.map((x) => x.row)).toEqual([2, 4]);
    });

    it("falls back to row 0 when the source row is unknown", () => {
        const d = validateStudentPoints([{ name: "A", points: 200 }], 100);
        expect(d).toHaveLength(1);
        expect(d[0]?.row).toBe(0);
    });
});
```

- [ ] **Step 2: Run test to verify it fails**

Run: `npx vitest run tests/unit/studentValidation.test.ts`
Expected: FAIL — Modul nicht auflösbar.

- [ ] **Step 3: Write the implementation**

Create `src/core/studentValidation.ts`:

```ts
import { pointsExceedMax } from "./diagnostics";
import { Diagnostic, Student } from "../types";

/**
 * Checks parsed students against the maximum score the teacher entered.
 *
 * This cannot live in the parsers: they do not know the maximum and only guard
 * against the absolute ceiling in `LIMITS`. A score above the maximum is almost
 * always a typo that would otherwise silently produce a top grade.
 */
export function validateStudentPoints(students: Student[], maxPoints: number): Diagnostic[] {
    const diagnostics: Diagnostic[] = [];

    for (const student of students) {
        if (student.points > maxPoints) {
            diagnostics.push(pointsExceedMax(student.sourceRow ?? 0, student.points, maxPoints));
        }
    }

    return diagnostics;
}
```

- [ ] **Step 4: Run tests and type-check**

Run: `npx vitest run tests/unit/studentValidation.test.ts && npm run type-check`
Expected: Tests PASS.

- [ ] **Step 5: Commit**

```bash
git add src/core/studentValidation.ts tests/unit/studentValidation.test.ts
git commit -m "feat: Punktzahlen oberhalb der Maximalpunktzahl brechen ab"
```

---

## Task 6: Workflow zusammenführen

**Files:**
- Modify: `src/ui/workflow.ts`
- Modify: `src/index.ts` (Export des neuen Moduls)
- Test: `tests/integration/fullWorkflow.test.ts`, `tests/integration/inputModes.test.ts`

**Interfaces:**
- Consumes: alles aus Tasks 1–5.
- Produces: `runCalculationWorkflow(input: WorkflowInput): WorkflowResult` mit `{ ok, diagnostics, gradeBounds, students, averageGrade }`.

Diese Task schließt Etappe 1 ab: Danach läuft `npm run test` wieder vollständig grün.

- [ ] **Step 1: Update the workflow result type**

In `src/ui/workflow.ts`, replace the `WorkflowResult` interface:

```ts
export interface WorkflowResult {
    ok: boolean;
    diagnostics: Diagnostic[];
    gradeBounds: GradeBound[];
    students: Student[];
    averageGrade: number;
}
```

- [ ] **Step 2: Write the failing integration tests**

Replace `tests/integration/inputModes.test.ts` completely:

```ts
import { describe, expect, it } from "vitest";
import { runCalculationWorkflow } from "../../src/ui/workflow";

describe("input mode behavior", () => {
    it("allows empty student input and still calculates the grade scale", () => {
        const result = runCalculationWorkflow({
            maxPoints: 100,
            minPoints: 1,
            breakPointPercent: 50,
            inputMode: "csv",
            csvContent: "",
            manualEntries: []
        });

        expect(result.ok).toBe(true);
        expect(result.diagnostics).toHaveLength(0);
        expect(result.gradeBounds).toHaveLength(5);
        expect(result.students).toHaveLength(0);
    });

    it("returns an error for invalid core inputs", () => {
        const result = runCalculationWorkflow({
            maxPoints: 0,
            minPoints: 1,
            breakPointPercent: 50,
            inputMode: "csv",
            csvContent: "",
            manualEntries: []
        });

        expect(result.ok).toBe(false);
        expect(result.diagnostics.some((d) => d.code === "invalid-core-input")).toBe(true);
    });

    it("reports data sitting in the mode that is not selected", () => {
        const result = runCalculationWorkflow({
            maxPoints: 100,
            minPoints: 0.5,
            breakPointPercent: 50,
            inputMode: "csv",
            csvContent: "",
            manualEntries: [{ name: "Anna", points: "90" }]
        });

        expect(result.ok).toBe(false);
        expect(result.diagnostics.some((d) => d.code === "input-mode-mismatch")).toBe(true);
    });
});
```

Append to `tests/integration/fullWorkflow.test.ts` — inside the existing `describe("full workflow", …)` block, and change `result.errors.join(" ")` in the last existing test to `result.diagnostics.map((d) => d.message).join(" ")`:

```ts
    it("calculates the remaining students and warns about every skipped row", () => {
        const result = runCalculationWorkflow({
            maxPoints: 100,
            minPoints: 0.5,
            breakPointPercent: 50,
            inputMode: "csv",
            csvContent: "Name,Punkte\nAlice,95\nKaputt\nBob,abc\nCharlie,20",
            manualEntries: []
        });

        expect(result.ok).toBe(true);
        expect(result.students).toHaveLength(2);
        expect(result.diagnostics).toHaveLength(2);
        expect(result.diagnostics.every((d) => d.severity === "warning")).toBe(true);
    });

    it("aborts when a score exceeds the maximum, naming the row", () => {
        const result = runCalculationWorkflow({
            maxPoints: 100,
            minPoints: 0.5,
            breakPointPercent: 50,
            inputMode: "csv",
            csvContent: "Name,Punkte\nTippfehler,855\nNormal,85.5",
            manualEntries: []
        });

        expect(result.ok).toBe(false);
        expect(result.students).toHaveLength(0);
        expect(result.gradeBounds).toHaveLength(0);

        const exceed = result.diagnostics.find((d) => d.code === "points-exceed-max");
        expect(exceed?.row).toBe(2);
        expect(exceed?.message).toContain("855");
    });

    it("reports an unusable scale", () => {
        const result = runCalculationWorkflow({
            maxPoints: 1,
            minPoints: 1,
            breakPointPercent: 99,
            inputMode: "csv",
            csvContent: "",
            manualEntries: []
        });

        expect(result.ok).toBe(false);
        expect(result.diagnostics.some((d) => d.code === "degenerate-scale")).toBe(true);
    });
```

- [ ] **Step 3: Run tests to verify they fail**

Run: `npx vitest run tests/integration`
Expected: FAIL — `result.diagnostics` ist `undefined`.

- [ ] **Step 4: Rewrite the workflow**

Replace the body of `src/ui/workflow.ts` below the `WorkflowInput` interface:

```ts
import { calculateGradeBounds, validateGradeBounds } from "../core/calculator";
import { hasErrors, noValidStudents } from "../core/diagnostics";
import { calculateAverageGrade, processStudents } from "../core/grading";
import { validateStudentPoints } from "../core/studentValidation";
import { validateCoreInputs, validateInputMode } from "../core/validation";
import { parseCSVContent } from "../parsers/csvParser";
import { hasNonEmptyManualEntries, parseManualEntries } from "../parsers/manualParser";
import { Diagnostic, GradeBound, InputMode, ManualEntry, Student } from "../types";

export function runCalculationWorkflow(input: WorkflowInput): WorkflowResult {
    const coreDiagnostics = validateCoreInputs(
        input.maxPoints,
        input.minPoints,
        input.breakPointPercent
    );
    if (hasErrors(coreDiagnostics)) {
        return failed(coreDiagnostics);
    }

    const gradeBounds = calculateGradeBounds(
        input.maxPoints,
        input.minPoints,
        input.breakPointPercent
    );

    const boundsDiagnostics = validateGradeBounds(gradeBounds);
    if (hasErrors(boundsDiagnostics)) {
        return failed(boundsDiagnostics);
    }

    const csvProvided = (input.csvContent ?? "").trim() !== "";
    const manualProvided = hasNonEmptyManualEntries(input.manualEntries);

    const modeDiagnostics = validateInputMode(input.inputMode, csvProvided, manualProvided);
    if (hasErrors(modeDiagnostics)) {
        return failed(modeDiagnostics);
    }

    const diagnostics: Diagnostic[] = [];
    let students: Student[] = [];

    if (input.inputMode === "csv" && csvProvided) {
        const parsed = parseCSVContent(input.csvContent ?? "");
        students = parsed.students;
        diagnostics.push(...parsed.diagnostics);
    } else if (input.inputMode === "manual" && manualProvided) {
        const parsed = parseManualEntries(input.manualEntries);
        students = parsed.students;
        diagnostics.push(...parsed.diagnostics);

        // The manual parser stays silent here; the workflow knows that input existed.
        if (students.length === 0) {
            diagnostics.push(noValidStudents());
        }
    }

    if (hasErrors(diagnostics)) {
        return failed(diagnostics);
    }

    diagnostics.push(...validateStudentPoints(students, input.maxPoints));
    if (hasErrors(diagnostics)) {
        return failed(diagnostics);
    }

    const gradedStudents = students.length > 0 ? processStudents(students, gradeBounds) : [];
    const averageGrade = gradedStudents.length > 0 ? calculateAverageGrade(gradedStudents) : 0;

    return {
        ok: true,
        diagnostics,
        gradeBounds,
        students: gradedStudents,
        averageGrade
    };
}

function failed(diagnostics: Diagnostic[]): WorkflowResult {
    return {
        ok: false,
        diagnostics,
        gradeBounds: [],
        students: [],
        averageGrade: 0
    };
}
```

Note: `failed()` deliberately keeps the warnings collected so far alongside the error — the teacher sees the full picture in one go.

- [ ] **Step 5: Export the new modules**

In `src/index.ts`, add after the existing core exports:

```ts
export * from "./core/diagnostics";
export * from "./core/studentValidation";
```

- [ ] **Step 6: Run the whole suite**

Run: `npm run test && npm run type-check`
Expected: Beides grün. `app.ts` referenziert noch `result.errors` — falls `type-check` das meldet, in `app.ts:167` vorläufig auf `result.diagnostics.map((d) => d.message).join(" ")` ändern; Task 8 ersetzt die Stelle vollständig.

- [ ] **Step 7: Commit**

```bash
git add src/ui/workflow.ts src/index.ts src/app.ts tests/integration
git commit -m "feat: Workflow aggregiert Diagnosen und trennt Warnung von Abbruch"
```

---

## Task 7: Formatierung

**Files:**
- Create: `src/ui/format.ts`
- Test: `tests/unit/format.test.ts`

**Interfaces:**
- Consumes: `GradeBound` aus `types.ts`.
- Produces: `decimalsFor(minPoints: number): number`, `formatNumberDe(value: number, decimals: number): string`, `formatPoints(value: number, minPoints: number): string`, `formatRange(bound: GradeBound, minPoints: number): string`.

Behebt B4. Reines Modul, kein DOM-Zugriff.

- [ ] **Step 1: Write the failing test**

Create `tests/unit/format.test.ts`:

```ts
import { describe, expect, it } from "vitest";
import { decimalsFor, formatNumberDe, formatPoints, formatRange } from "../../src/ui/format";
import { calculateGradeBounds } from "../../src/core/calculator";

describe("decimalsFor", () => {
    it("derives the decimals from the step size", () => {
        expect(decimalsFor(1)).toBe(0);
        expect(decimalsFor(0.5)).toBe(1);
        expect(decimalsFor(0.25)).toBe(2);
        expect(decimalsFor(0.1)).toBe(1);
    });

    it("caps at three decimals", () => {
        expect(decimalsFor(0.00001)).toBe(3);
    });
});

describe("formatNumberDe", () => {
    it("uses a decimal comma", () => {
        expect(formatNumberDe(76.5, 1)).toBe("76,5");
        expect(formatNumberDe(100, 0)).toBe("100");
        expect(formatNumberDe(33.25, 2)).toBe("33,25");
    });
});

describe("formatPoints", () => {
    it("keeps quarter points intact instead of rounding them away", () => {
        // The bug from the review: toFixed(1) turned 33.25 into "33.3".
        expect(formatPoints(33.25, 0.25)).toBe("33,25");
        expect(formatPoints(39.5, 0.5)).toBe("39,5");
        expect(formatPoints(45, 1)).toBe("45");
    });
});

describe("formatRange", () => {
    it("formats a grade boundary with the step size precision", () => {
        const bounds = calculateGradeBounds(37, 0.25, 60);
        const grade1 = bounds[0];
        expect(grade1).toBeDefined();
        expect(grade1?.lowerBound).toBe(33.25);
        expect(formatRange(grade1!, 0.25)).toBe("33,25 - 37");
    });

    it("matches the displayed scale for half points", () => {
        const bounds = calculateGradeBounds(45, 0.5, 50);
        expect(formatRange(bounds[0]!, 0.5)).toBe("39,5 - 45");
        expect(formatRange(bounds[4]!, 0.5)).toBe("0 - 22");
    });
});
```

- [ ] **Step 2: Run test to verify it fails**

Run: `npx vitest run tests/unit/format.test.ts`
Expected: FAIL — Modul nicht auflösbar.

- [ ] **Step 3: Write the implementation**

Create `src/ui/format.ts`:

```ts
import { GradeBound } from "../types";

const MAX_DECIMALS = 3;

/**
 * Derives how many decimals a value needs from the configured step size.
 *
 * A step of 0.25 produces boundaries with two decimals; formatting those with a
 * fixed single decimal (as the old `toFixed(1)` did) showed 33.3 where 33.25 was
 * meant, so the printed scale disagreed with the calculation.
 */
export function decimalsFor(minPoints: number): number {
    if (!Number.isFinite(minPoints) || minPoints <= 0) {
        return 0;
    }

    const text = String(minPoints);
    if (text.includes("e") || text.includes("E")) {
        return MAX_DECIMALS;
    }

    const fraction = text.split(".")[1];
    if (fraction === undefined) {
        return 0;
    }

    return Math.min(fraction.length, MAX_DECIMALS);
}

/** Formats a number German-style, with a decimal comma and no trailing zeros. */
export function formatNumberDe(value: number, decimals: number): string {
    const fixed = value.toFixed(decimals);
    const trimmed = decimals > 0 ? fixed.replace(/\.?0+$/, "") : fixed;
    return trimmed.replace(".", ",");
}

export function formatPoints(value: number, minPoints: number): string {
    return formatNumberDe(value, decimalsFor(minPoints));
}

export function formatRange(bound: GradeBound, minPoints: number): string {
    return `${formatPoints(bound.lowerBound, minPoints)} - ${formatPoints(bound.upperBound, minPoints)}`;
}
```

- [ ] **Step 4: Run tests and type-check**

Run: `npx vitest run tests/unit/format.test.ts && npm run type-check`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add src/ui/format.ts tests/unit/format.test.ts
git commit -m "feat: Zahlendarstellung folgt der Schrittweite"
```

---

## Task 8: Rendering herauslösen und Diagnosen anzeigen

**Files:**
- Create: `src/ui/render.ts`
- Modify: `src/app.ts` (entfernt `renderResults`, `showMessage`, `hideMessage`)
- Modify: `index.html` (Diagnose-Container, `step="any"`, `role="alert"`)
- Modify: `style.css`
- Test: `tests/unit/render.test.ts`

**Interfaces:**
- Consumes: `formatPoints`, `formatRange` aus Task 7; `Diagnostic` aus Task 1.
- Produces: `buildDiagnosticLines(diagnostics: Diagnostic[]): string[]`, `summaryLine(studentCount: number, hasWarnings: boolean): string`, `renderGradeScale(tbody: HTMLTableSectionElement, bounds: GradeBound[], minPoints: number): void`, `renderStudents(tbody: HTMLTableSectionElement, students: Student[], minPoints: number): void`, `renderDiagnostics(list: HTMLUListElement, diagnostics: Diagnostic[]): void`.

Behebt B7 und den `innerHTML`-Teil von B8. Die reinen Funktionen sind testbar, die DOM-Funktionen bleiben schlank.

- [ ] **Step 1: Write the failing test**

Create `tests/unit/render.test.ts`:

```ts
import { describe, expect, it } from "vitest";
import { buildDiagnosticLines, summaryLine } from "../../src/ui/render";
import { pointsExceedMax, rowUnparsablePoints } from "../../src/core/diagnostics";

describe("buildDiagnosticLines", () => {
    it("returns one line per diagnostic, errors first", () => {
        const lines = buildDiagnosticLines([
            rowUnparsablePoints(3, "abc"),
            pointsExceedMax(2, 855, 100)
        ]);

        expect(lines).toHaveLength(2);
        expect(lines[0]).toContain("855");
        expect(lines[1]).toContain("abc");
    });

    it("returns an empty list for no diagnostics", () => {
        expect(buildDiagnosticLines([])).toHaveLength(0);
    });
});

describe("summaryLine", () => {
    it("names the student count when warnings occurred", () => {
        expect(summaryLine(25, true)).toContain("25");
    });

    it("stays short when everything went through", () => {
        expect(summaryLine(25, false)).toBe("Berechnung abgeschlossen.");
    });
});
```

- [ ] **Step 2: Run test to verify it fails**

Run: `npx vitest run tests/unit/render.test.ts`
Expected: FAIL — Modul nicht auflösbar.

- [ ] **Step 3: Write render.ts**

Create `src/ui/render.ts`:

```ts
import { formatPoints, formatRange } from "./format";
import { Diagnostic, GradeBound, Student } from "../types";

/** Errors first, then warnings; within a group the original order is kept. */
export function buildDiagnosticLines(diagnostics: Diagnostic[]): string[] {
    const errors = diagnostics.filter((d) => d.severity === "error");
    const warnings = diagnostics.filter((d) => d.severity === "warning");
    return [...errors, ...warnings].map((d) => d.message);
}

export function summaryLine(studentCount: number, hasWarnings: boolean): string {
    if (!hasWarnings) {
        return "Berechnung abgeschlossen.";
    }
    return `Berechnung abgeschlossen: ${studentCount} Schülerinnen und Schüler gewertet. Bitte die Hinweise prüfen.`;
}

export function renderGradeScale(
    tbody: HTMLTableSectionElement,
    bounds: GradeBound[],
    minPoints: number
): void {
    tbody.replaceChildren();

    for (const bound of bounds) {
        const tr = document.createElement("tr");
        tr.className = `grade-${bound.grade}`;
        tr.appendChild(cell(String(bound.grade)));
        tr.appendChild(cell(formatRange(bound, minPoints)));
        tbody.appendChild(tr);
    }
}

export function renderStudents(
    tbody: HTMLTableSectionElement,
    students: Student[],
    minPoints: number
): void {
    tbody.replaceChildren();

    for (const student of students) {
        const tr = document.createElement("tr");
        if (student.grade) {
            tr.className = `grade-${student.grade}`;
        }
        tr.appendChild(cell(student.name));
        tr.appendChild(cell(formatPoints(student.points, minPoints)));
        tr.appendChild(cell(student.grade === undefined ? "" : String(student.grade)));
        tbody.appendChild(tr);
    }
}

export function renderDiagnostics(list: HTMLUListElement, diagnostics: Diagnostic[]): void {
    list.replaceChildren();

    for (const line of buildDiagnosticLines(diagnostics)) {
        const li = document.createElement("li");
        li.textContent = line;
        list.appendChild(li);
    }
}

/** Uses textContent throughout — names never reach the DOM as markup. */
function cell(text: string): HTMLTableCellElement {
    const td = document.createElement("td");
    td.textContent = text;
    return td;
}
```

- [ ] **Step 4: Update index.html**

Change line 31 so quarter-point steps are accepted by the browser:

```html
<input id="minPoints" type="number" value="0.5" min="0.1" step="any" required />
```

Replace the message container (line 21) with a container that screen readers announce:

```html
<div id="message" class="message hidden" role="alert">
    <p id="messageText"></p>
    <ul id="diagnosticList" class="diagnostics"></ul>
</div>
```

- [ ] **Step 5: Add the styles**

Append to `style.css`:

```css
.diagnostics {
    margin: 0.5rem 0 0;
    padding-left: 1.2rem;
    list-style: disc;
}

.diagnostics:empty {
    display: none;
}

.diagnostics li {
    margin-top: 0.2rem;
}
```

- [ ] **Step 6: Rewire app.ts**

In `src/app.ts`: delete `renderResults`, `showMessage` and `hideMessage`, add the imports

```ts
import { renderDiagnostics, renderGradeScale, renderStudents, summaryLine } from "./ui/render";
import { Diagnostic } from "./types";
```

and insert these replacements:

```ts
function showMessage(type: "error" | "success", text: string, diagnostics: Diagnostic[] = []): void {
    const msg = getById<HTMLDivElement>("message");
    msg.classList.remove("hidden", "error", "success");
    msg.classList.add(type);
    getById<HTMLParagraphElement>("messageText").textContent = text;
    renderDiagnostics(getById<HTMLUListElement>("diagnosticList"), diagnostics);
}

function hideMessage(): void {
    const msg = getById<HTMLDivElement>("message");
    msg.classList.add("hidden");
    getById<HTMLParagraphElement>("messageText").textContent = "";
    renderDiagnostics(getById<HTMLUListElement>("diagnosticList"), []);
}

function renderResults(): void {
    renderGradeScale(
        getById<HTMLTableSectionElement>("gradeScaleBody"),
        state.gradeBounds,
        state.minPoints
    );
    renderStudents(
        getById<HTMLTableSectionElement>("studentsBody"),
        state.students,
        state.minPoints
    );

    const average = getById<HTMLHeadingElement>("averageGrade");
    average.textContent =
        state.students.length > 0 ? `Notendurchschnitt: ${state.averageGrade.toFixed(2)}` : "";

    getById<HTMLDivElement>("gradeScaleCard").classList.toggle("hidden", state.gradeBounds.length === 0);
    getById<HTMLDivElement>("studentsCard").classList.toggle("hidden", state.students.length === 0);
}
```

In `handleSubmit`, replace the error branch and the success call:

```ts
    if (!result.ok) {
        showMessage("error", "Die Berechnung wurde abgebrochen.", result.diagnostics);
        state.gradeBounds = [];
        state.students = [];
        state.averageGrade = 0;
        renderResults();
        return;
    }
```

```ts
    sessionStorage.setItem("notenschluessel:lastState", JSON.stringify(state));
    renderResults();
    showMessage(
        "success",
        summaryLine(result.students.length, result.diagnostics.length > 0),
        result.diagnostics
    );
```

Important: the four `state.*` assignments must run **before** `renderResults()`, so `state.minPoints` already holds the new step size.

- [ ] **Step 7: Run tests, type-check and the app**

Run: `npm run test && npm run type-check`
Expected: Beides grün.

Run: `npm run dev`, dann im Browser `http://localhost:5173`:
- Maximale Punktzahl 37, Schrittweite 0,25, Knickpunkt 60 → Note 1 muss **33,25 - 37** anzeigen, nicht 33,3.
- CSV mit einer kaputten Zeile hochladen → Ergebnis erscheint, darunter eine Hinweisliste mit Zeilennummer.

- [ ] **Step 8: Commit**

```bash
git add src/ui/render.ts src/app.ts index.html style.css tests/unit/render.test.ts
git commit -m "feat: Rendering ausgelagert, Diagnosen sichtbar und vorlesbar"
```

---

## Task 9: CSV-Export korrigieren

**Files:**
- Modify: `src/export/csvExport.ts`
- Test: `tests/unit/export.test.ts`

**Interfaces:**
- Consumes: `formatPoints`, `formatNumberDe` aus Task 7.
- Produces: `exportGradeScaleCSV(bounds, minPoints)`, `exportStudentResultsCSV(students, minPoints)`, `exportCombinedCSV(bounds, students, metadata)` — alle mit Semikolon getrennt. Signaturen ändern sich: `minPoints` kommt hinzu.

Behebt B5 und den Trennzeichen-Teil von B6.

- [ ] **Step 1: Write the failing tests**

Replace the CSV portion of `tests/unit/export.test.ts` (the Excel tests stay untouched):

```ts
    it("keeps a dangerous name in a single field even when it contains the delimiter", () => {
        // The old code returned early after prefixing, so the delimiter split the row.
        const csv = exportStudentResultsCSV([{ name: "=Mueller; Hans", points: 90, grade: 1 }], 0.5);
        const dataLine = csv.split("\n")[1];
        expect(dataLine).toBeDefined();
        expect(dataLine).toBe(`"'=Mueller; Hans";90;1`);
    });

    it("still prefixes formula characters", () => {
        const csv = exportStudentResultsCSV([{ name: "=cmd", points: 45, grade: 1 }], 0.5);
        expect(csv).toContain("'=cmd");
    });

    it("uses semicolons and a decimal comma", () => {
        const csv = exportStudentResultsCSV([{ name: "Alice", points: 76.5, grade: 2 }], 0.5);
        expect(csv.split("\n")[0]).toBe("Name;Punkte;Note");
        expect(csv).toContain("Alice;76,5;2");
    });

    it("exports the grade scale with semicolons", () => {
        const csv = exportGradeScaleCSV(bounds, 0.5);
        expect(csv.split("\n")[0]).toBe("Note;Untergrenze;Obergrenze");
        expect(csv).toContain("1;39,5;45");
    });

    it("exports combined csv", () => {
        const csv = exportCombinedCSV(bounds, students, {
            maxPoints: 45,
            minPoints: 0.5,
            breakPointPercent: 50
        });
        expect(csv).toContain("## Notenskala");
        expect(csv).toContain("## Schüler");
    });
```

The shared `students` fixture at the top of the file stays as it is: `sourceRow` is optional and the exporters do not read it.

- [ ] **Step 2: Run tests to verify they fail**

Run: `npx vitest run tests/unit/export.test.ts`
Expected: FAIL — Kommas statt Semikolons, und das gefährliche Feld ist nicht gequotet.

- [ ] **Step 3: Rewrite csvExport.ts**

Replace the top half of `src/export/csvExport.ts`:

```ts
import { formatNumberDe, formatPoints } from "../ui/format";
import { GradeBound, Student } from "../types";

const DELIMITER = ";";
const FORMULA_PREFIXES = "=+-@\t\r";

/**
 * Guards a field against spreadsheet formula injection **and** against breaking
 * the column layout. Both steps run: prefixing alone left a value containing the
 * delimiter split across cells.
 */
function sanitizeCSVField(field: string): string {
    if (!field) {
        return "";
    }

    const firstChar = field[0] ?? "";
    const guarded = FORMULA_PREFIXES.includes(firstChar) ? `'${field}` : field;

    if (
        guarded.includes(DELIMITER) ||
        guarded.includes('"') ||
        guarded.includes("\n") ||
        guarded !== field
    ) {
        return `"${guarded.replaceAll('"', '""')}"`;
    }

    return guarded;
}

export function exportGradeScaleCSV(bounds: GradeBound[], minPoints: number): string {
    const lines = [["Note", "Untergrenze", "Obergrenze"].join(DELIMITER)];
    for (const b of bounds) {
        lines.push(
            [
                String(b.grade),
                formatPoints(b.lowerBound, minPoints),
                formatPoints(b.upperBound, minPoints)
            ].join(DELIMITER)
        );
    }
    return lines.join("\n");
}

export function exportStudentResultsCSV(students: Student[], minPoints: number): string {
    const lines = [["Name", "Punkte", "Note"].join(DELIMITER)];
    for (const s of students) {
        lines.push(
            [
                sanitizeCSVField(s.name),
                formatPoints(s.points, minPoints),
                s.grade === undefined ? "" : String(s.grade)
            ].join(DELIMITER)
        );
    }
    return lines.join("\n");
}

export function exportCombinedCSV(
    bounds: GradeBound[],
    students: Student[],
    metadata: { maxPoints: number; minPoints: number; breakPointPercent: number }
): string {
    const lines: string[] = [];
    lines.push("# Notenschlüssel Export");
    lines.push(`# MaxPoints: ${metadata.maxPoints}`);
    lines.push(`# MinPoints: ${formatNumberDe(metadata.minPoints, 2)}`);
    lines.push(`# BreakPointPercent: ${metadata.breakPointPercent}`);
    lines.push("");
    lines.push("## Notenskala");
    lines.push(exportGradeScaleCSV(bounds, metadata.minPoints));
    lines.push("");
    lines.push("## Schüler");
    lines.push(exportStudentResultsCSV(students, metadata.minPoints));
    return lines.join("\n");
}
```

`triggerTextDownload` bleibt in dieser Task unverändert — Task 10 ergänzt das BOM.

- [ ] **Step 4: Update the call sites**

In `src/app.ts`, `setupExports()`: pass the step size to the two changed functions.

```ts
        triggerTextDownload(
            exportGradeScaleCSV(state.gradeBounds, state.minPoints),
            "grade-scale.csv",
            "text/csv;charset=utf-8;"
        );
```

```ts
        triggerTextDownload(
            exportStudentResultsCSV(state.students, state.minPoints),
            "student-results.csv",
            "text/csv;charset=utf-8;"
        );
```

- [ ] **Step 5: Run tests and type-check**

Run: `npm run test && npm run type-check`
Expected: Beides grün.

- [ ] **Step 6: Commit**

```bash
git add src/export/csvExport.ts src/app.ts tests/unit/export.test.ts
git commit -m "fix: CSV-Export quotet gefaehrliche Felder und nutzt Semikolon"
```

---

## Task 10: BOM schreiben und beim Import wieder entfernen

**Files:**
- Modify: `src/export/csvExport.ts` (`triggerTextDownload`)
- Modify: `src/parsers/csvParser.ts` (BOM-Toleranz)
- Test: `tests/unit/csvParser.test.ts`, `tests/integration/roundTrip.test.ts`

**Interfaces:**
- Consumes: Exportfunktionen aus Task 9, `parseCSVContent` aus Task 2.
- Produces: unveränderte Signaturen; `triggerTextDownload` stellt das BOM `\uFEFF` voran.

Behebt den Umlaut-Teil von B6 und sichert den Round-Trip.

- [ ] **Step 1: Write the failing round-trip test**

Create `tests/integration/roundTrip.test.ts`:

```ts
import { describe, expect, it } from "vitest";
import { exportStudentResultsCSV } from "../../src/export/csvExport";
import { parseCSVContent } from "../../src/parsers/csvParser";

describe("csv round trip", () => {
    const students = [
        { name: "Müller", points: 76.5, grade: 2 },
        { name: "Öz", points: 100, grade: 1 }
    ];

    it("reads its own export back unchanged", () => {
        const csv = exportStudentResultsCSV(students, 0.5);
        const parsed = parseCSVContent(csv);

        expect(parsed.diagnostics).toHaveLength(0);
        expect(parsed.students.map((s) => s.name)).toEqual(["Müller", "Öz"]);
        expect(parsed.students.map((s) => s.points)).toEqual([76.5, 100]);
    });

    it("reads its own export back even with a leading BOM", () => {
        const csv = `\uFEFF${exportStudentResultsCSV(students, 0.5)}`;
        const parsed = parseCSVContent(csv);

        expect(parsed.diagnostics).toHaveLength(0);
        expect(parsed.students.map((s) => s.name)).toEqual(["Müller", "Öz"]);
    });
});
```

- [ ] **Step 2: Add a BOM test to the parser suite**

Append to `tests/unit/csvParser.test.ts`:

```ts
    it("ignores a leading BOM when detecting the header", () => {
        const result = parseCSVContent("\uFEFFName,Punkte\nAlice,95");
        expect(result.students).toHaveLength(1);
        expect(result.students[0]?.name).toBe("Alice");
    });
```

- [ ] **Step 3: Run tests to verify they fail**

Run: `npx vitest run tests/integration/roundTrip.test.ts tests/unit/csvParser.test.ts`
Expected: FAIL — der BOM-Test schlägt fehl, weil die Kopfzeile als `\uFEFFname` nicht erkannt wird und `Name` als Schülername mit unlesbarer Punktzahl endet.

- [ ] **Step 4: Strip the BOM on import**

In `src/parsers/csvParser.ts`, at the start of `parseCSVContent`, before the empty check:

```ts
export function parseCSVContent(rawContent: string): CSVParseResult {
    // A file exported by this app starts with a BOM so Excel reads it as UTF-8.
    const content = rawContent.startsWith("\uFEFF") ? rawContent.slice(1) : rawContent;

    const students: Student[] = [];
    const diagnostics: Diagnostic[] = [];
    // … rest unchanged
```

- [ ] **Step 5: Write the BOM on export**

In `src/export/csvExport.ts`, replace `triggerTextDownload`:

```ts
/**
 * Prepends a UTF-8 BOM so Excel on Windows reads umlauts correctly. It sits here
 * rather than in the export functions so their return values stay plain strings
 * that tests can assert on directly.
 */
export function triggerTextDownload(content: string, filename: string, mimeType: string): void {
    const blob = new Blob([`\uFEFF${content}`], { type: mimeType });
    const url = URL.createObjectURL(blob);
    const link = document.createElement("a");
    link.href = url;
    link.download = filename;
    link.click();
    URL.revokeObjectURL(url);
}
```

- [ ] **Step 6: Run tests and type-check**

Run: `npm run test && npm run type-check`
Expected: Beides grün.

- [ ] **Step 7: Commit**

```bash
git add src/export/csvExport.ts src/parsers/csvParser.ts tests/unit/csvParser.test.ts tests/integration/roundTrip.test.ts
git commit -m "fix: BOM im Export, BOM-Toleranz im Import"
```

---

## Task 11: `sanitizeName` entdoppeln

**Files:**
- Create: `src/core/sanitize.ts`
- Modify: `src/parsers/csvParser.ts`, `src/parsers/manualParser.ts`
- Modify: `src/index.ts`
- Test: `tests/unit/sanitize.test.ts`

**Interfaces:**
- Produces: `sanitizeName(name: string): string`.

- [ ] **Step 1: Write the failing test**

Create `tests/unit/sanitize.test.ts`:

```ts
import { describe, expect, it } from "vitest";
import { sanitizeName } from "../../src/core/sanitize";
import { LIMITS } from "../../src/constants";

describe("sanitizeName", () => {
    it("removes angle brackets", () => {
        expect(sanitizeName("<Alice>")).toBe("Alice");
    });

    it("collapses line breaks and tabs into spaces", () => {
        expect(sanitizeName("A\nB\tC")).toBe("A B C");
    });

    it("trims surrounding whitespace", () => {
        expect(sanitizeName("  Alice  ")).toBe("Alice");
    });

    it("keeps umlauts intact", () => {
        expect(sanitizeName("Müller-Öztürk")).toBe("Müller-Öztürk");
    });

    it("truncates overly long names", () => {
        const long = "x".repeat(LIMITS.maxNameLength + 50);
        expect(sanitizeName(long)).toHaveLength(LIMITS.maxNameLength);
    });
});
```

- [ ] **Step 2: Run test to verify it fails**

Run: `npx vitest run tests/unit/sanitize.test.ts`
Expected: FAIL — Modul nicht auflösbar.

- [ ] **Step 3: Create the module**

Create `src/core/sanitize.ts` with the function that is currently duplicated in both parsers:

```ts
import { LIMITS } from "../constants";

export function sanitizeName(name: string): string {
    let result = name.replaceAll("<", "").replaceAll(">", "");
    result = result.replaceAll("\n", " ").replaceAll("\r", " ").replaceAll("\t", " ");
    result = result.trim();

    if (result.length > LIMITS.maxNameLength) {
        return result.slice(0, LIMITS.maxNameLength);
    }

    return result;
}
```

- [ ] **Step 4: Remove both copies**

Delete the local `sanitizeName` from `src/parsers/csvParser.ts` and `src/parsers/manualParser.ts`, adding to each:

```ts
import { sanitizeName } from "../core/sanitize";
```

In `src/index.ts`, add:

```ts
export * from "./core/sanitize";
```

- [ ] **Step 5: Run tests and type-check**

Run: `npm run test && npm run type-check`
Expected: Beides grün — die bestehenden „sanitizes names"-Tests beider Parser decken die Verdrahtung ab.

- [ ] **Step 6: Commit**

```bash
git add src/core/sanitize.ts src/parsers src/index.ts tests/unit/sanitize.test.ts
git commit -m "refactor: sanitizeName in ein gemeinsames Modul"
```

---

## Task 12: Linter einrichten

**Files:**
- Create: `eslint.config.js`
- Modify: `package.json`, `.github/workflows/ci.yml`

- [ ] **Step 1: Install the dependencies**

```bash
npm install --save-dev eslint@^9 typescript-eslint@^8 @eslint/js@^9
```

- [ ] **Step 2: Create the config**

Create `eslint.config.js`:

```js
import tseslint from "typescript-eslint";

// Only the TypeScript ruleset: eslint's own `recommended` enables `no-undef`,
// which reports false positives on TS type syntax.
export default tseslint.config(
    ...tseslint.configs.recommended,
    {
        ignores: ["dist/**", "node_modules/**", "eslint.config.js"]
    },
    {
        files: ["src/**/*.ts", "tests/**/*.ts"],
        rules: {
            "@typescript-eslint/no-unused-vars": ["error", { argsIgnorePattern: "^_" }],
            "no-console": "warn",
            eqeqeq: ["error", "always"]
        }
    }
);
```

The `argsIgnorePattern` keeps `_lowerBound5` in `src/core/grading.ts` legal.

- [ ] **Step 3: Add the script**

In `package.json`, inside `scripts`:

```json
        "lint": "eslint src tests"
```

- [ ] **Step 4: Run it and fix what it reports**

Run: `npm run lint`
Expected: Entweder sauber oder eine kurze Liste. Gemeldete Verstöße hier beheben — aber ausschließlich mechanische (ungenutzte Importe, `==` statt `===`). Keine inhaltlichen Umbauten in dieser Task.

- [ ] **Step 5: Add it to CI**

In `.github/workflows/ci.yml`, after the „Run Type Check" step:

```yaml
      - name: Run Lint
        run: npm run lint
```

- [ ] **Step 6: Verify the whole suite**

Run: `npm run lint && npm run type-check && npm run test`
Expected: Alle drei grün.

- [ ] **Step 7: Commit**

```bash
git add eslint.config.js package.json package-lock.json .github/workflows/ci.yml
git commit -m "chore: ESLint einrichten und in CI einbinden"
```

---

## Task 13: Dokumentation und Node-Version angleichen

**Files:**
- Modify: `dockerfile`, `README.md`, `.github/copilot-instructions.md`, `docs/improvement-roadmap.md`

- [ ] **Step 1: Align the Node version**

In `dockerfile` line 2, change `node:26-alpine` to `node:24-alpine`, matching `.nvmrc` and CI.

- [ ] **Step 2: Fix the dependency name in README.md**

Replace the dependency bullet:

```markdown
- [xlsx-js-style](https://www.npmjs.com/package/xlsx-js-style) – Excel-Export mit Zellfarben direkt im Browser
```

- [ ] **Step 3: Document the CSV format change in README.md**

Replace the „CSV-Format" section:

````markdown
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
````

- [ ] **Step 4: Update copilot-instructions.md**

In the „Eingabevalidierung" section, replace the bullet list with:

```markdown
- `maxPoints`: Pflicht, positive Ganzzahl, max 1000
- `minPoints`: Pflicht, positive Zahl, ≤ maxPoints
- `breakPointPercent`: 1–99
- Punktzahlen über `maxPoints` → `points-exceed-max` (Abbruch), geprüft in `src/core/studentValidation.ts`
- CSV und Manuell sind gegenseitig ausschließend; Daten im nicht gewählten Modus → `input-mode-mismatch`
- Alle Meldungen entstehen als `Diagnostic` in `src/core/diagnostics.ts`, nie als roher String
- `severity: "warning"` rechnet weiter, `severity: "error"` bricht ab
```

Add to „Häufige Fehler vermeiden":

```markdown
8. ❌ Zahlen mit `toFixed()` formatieren – immer `formatPoints()` aus `src/ui/format.ts`, sonst weicht die Anzeige bei Schrittweite 0,25 von der Berechnung ab
9. ❌ Diagnose-Texte an der Fundstelle zusammenbauen – Factories in `src/core/diagnostics.ts` verwenden
10. ❌ Werte per `innerHTML` in Tabellen schreiben – `src/ui/render.ts` setzt ausschließlich `textContent`
```

- [ ] **Step 5: Mark the roadmap entries that are now done**

In `docs/improvement-roadmap.md`, section 2, replace the „Detailliertere Fehlermeldungen" row and add a note under section 4:

```markdown
| ~~**Detailliertere Fehlermeldungen**~~ | Erledigt: strukturierte `Diagnostic`-Typen mit Severity und Zeilennummer. | – | – |
```

```markdown
*CI/CD Pipeline ist umgesetzt (`.github/workflows/ci.yml`), inklusive Lint-Schritt.*
```

- [ ] **Step 6: Verify the build still works**

```bash
npm run build
docker build -t notenschluessel:verify .
```

Note: use `docker build`, not `docker compose build` — `compose.yml` only references
a prebuilt `image:` from ghcr.io and defines no build context, so the compose variant fails.

Expected: Beide Kommandos erfolgreich; das Image meldet am Ende `naming to ...notenschluessel:verify`.

- [ ] **Step 7: Commit**

```bash
git add dockerfile README.md .github/copilot-instructions.md docs/improvement-roadmap.md
git commit -m "docs: Node-Version, CSV-Format und Diagnose-Konventionen dokumentieren"
```

---

## Task 14: Abnahme gegen die Akzeptanzkriterien

**Files:** keine Änderungen, außer es fällt etwas auf.

- [ ] **Step 1: Run everything**

```bash
npm run lint && npm run type-check && npm run test && npm run build
```

Expected: Alle vier grün.

- [ ] **Step 2: Walk through the acceptance criteria in the browser**

Run `npm run dev`, then check each one:

| # | Prüfung | Erwartung |
| :--- | :--- | :--- |
| 1 | CSV mit `Tippfehler,855` bei Maximum 100 | Abbruch, Zeilennummer und 855 genannt, keine Notentabelle |
| 2 | CSV mit 4 Zeilen, davon 2 kaputt | 2 Schüler berechnet, beide übersprungenen Zeilen einzeln gelistet, Anzahl genannt |
| 3 | Modus „CSV", Daten in der manuellen Tabelle | Erklärende Fehlermeldung, kein leeres Ergebnis |
| 4 | 37 / 0,25 / 60 | Anzeige „33,25 - 37"; CSV-Export enthält `33,25` |
| 5 | Export öffnen | Trennzeichen `;`, BOM vorhanden, Umlaute lesbar |
| 6 | Schüler `=Test; Fall` exportieren | Genau ein Feld: `"'=Test; Fall"` |
| 7 | Fehlermeldung auslösen | Container trägt `role="alert"` |
| 8 | siehe Step 1 | grün |

Kriterium 5 lässt sich ohne Excel prüfen mit:

```bash
xxd ~/Downloads/student-results.csv | head -1   # muss mit "efbb bf" beginnen
```

Adjust the path if the browser saves elsewhere.

- [ ] **Step 3: Confirm the original review findings are gone**

```bash
git log --oneline docs/superpowers/specs/2026-08-09-datenintegritaet-design.md
```

Für jeden Befund B1–B8 im Spec prüfen, dass ein Test existiert, der ihn abdeckt. Fehlt einer, hier nachtragen statt ihn zu übergehen.

- [ ] **Step 4: Commit any fixes and push the branch**

```bash
git push -u origin "$(git rev-parse --abbrev-ref HEAD)"
```

Never force-push, and never push to `main`.

---

## Deckungsübersicht

| Befund | Task | Test |
| :--- | :--- | :--- |
| B1 Punkte über Maximum | 5, 6 | `studentValidation.test.ts`, `fullWorkflow.test.ts` |
| B2 Verworfene Zeilen stumm | 2, 6 | `csvParser.test.ts`, `fullWorkflow.test.ts` |
| B3 Daten im inaktiven Modus | 4, 6 | `validation.test.ts`, `inputModes.test.ts` |
| B4 Anzeige-Rundung | 7, 8 | `format.test.ts` |
| B5 CSV-Injection bricht Spalten | 9 | `export.test.ts` |
| B6 BOM und Trennzeichen | 9, 10 | `export.test.ts`, `roundTrip.test.ts` |
| B7 Meldungen ohne `role="alert"` | 8 | manuell, Task 14 |
| B8 Duplikat, `innerHTML`, Node, README, Umlaute, Lint | 8, 11, 12, 13 | `sanitize.test.ts`, `render.test.ts`, `npm run lint` |
