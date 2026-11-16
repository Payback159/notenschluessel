import { describe, expect, it } from "vitest";
import { calculateGradeBounds, validateGradeBounds } from "../../src/core/calculator";
import { processStudents } from "../../src/core/grading";
import { runCalculationWorkflow, WorkflowInput } from "../../src/ui/workflow";
import { GradeBound } from "../../src/types";

/**
 * Regressionsschutz für die Notenberechnung. Hier geht es um Schülernoten, darum
 * wird nicht stichprobenartig, sondern über ein ganzes Eingaberaster geprüft.
 *
 * - Invarianten: Regeln, die für jeden gültigen Notenschlüssel gelten müssen.
 * - Golden-Snapshots: halten den geprüften Ausgabestand fest. Schlägt einer fehl,
 *   hat sich das Ergebnis für mindestens eine Eingabe verändert. Nur nach
 *   bewusster, fachlich geprüfter Änderung mit `npx vitest run -u` aktualisieren.
 */

const MAX_POINTS = Array.from({ length: 1000 }, (_, i) => i + 1);
const STEPS = [0.01, 0.1, 0.125, 0.25, 0.5, 1, 2, 3, 5];
const BREAKPOINTS = [...Array.from({ length: 99 }, (_, i) => i + 1), 33.3, 37.5, 66.67];

function* allScales(): Generator<{ maxPoints: number; minPoints: number; bp: number; bounds: GradeBound[] }> {
    for (const maxPoints of MAX_POINTS) {
        for (const minPoints of STEPS) {
            if (minPoints > maxPoints) continue;
            for (const bp of BREAKPOINTS) {
                yield { maxPoints, minPoints, bp, bounds: calculateGradeBounds(maxPoints, minPoints, bp) };
            }
        }
    }
}

const gradeOf = (points: number, bounds: GradeBound[]) =>
    processStudents([{ name: "S", points }], bounds)[0]?.grade;

const EPS = 1e-9;

async function sha256(text: string): Promise<string> {
    const digest = await crypto.subtle.digest("SHA-256", new TextEncoder().encode(text));
    return Array.from(new Uint8Array(digest), (b) => b.toString(16).padStart(2, "0")).join("");
}

describe("Regression: Notenberechnung", () => {
    it("jeder gültige Notenschlüssel hält die fachlichen Regeln ein", () => {
        let checked = 0;

        for (const { maxPoints, minPoints, bp, bounds } of allScales()) {
            if (validateGradeBounds(bounds).length > 0) continue;
            checked++;
            const where = `max=${maxPoints} step=${minPoints} bp=${bp}`;
            const fail = (msg: string) => expect.fail(`${msg} (${where}): ${JSON.stringify(bounds)}`);

            if (bounds.map((b) => b.grade).join() !== "1,2,3,4,5") fail("Noten nicht 1–5 in Reihenfolge");
            if (bounds[0]?.upperBound !== maxPoints) fail("Note 1 endet nicht bei der Maximalpunktzahl");
            if (bounds[4]?.lowerBound !== 0) fail("Note 5 beginnt nicht bei 0");

            // Die Untergrenzen liegen höchstens eine halbe Schrittweite neben dem exakten Wert:
            // Knickpunkt für Note 4, darüber vier gleich große Segmente.
            const breakPoint = (maxPoints * bp) / 100;
            const segment = (maxPoints - breakPoint) / 4;
            for (let k = 0; k < 4; k++) {
                const exact = breakPoint + k * segment;
                const lower = bounds[3 - k]?.lowerBound ?? Number.NaN;
                if (Math.abs(lower - exact) > minPoints / 2 + EPS) fail(`Untergrenze Note ${4 - k} falsch gerundet`);
                const steps = lower / minPoints;
                if (Math.abs(steps - Math.round(steps)) > EPS) fail(`Untergrenze Note ${4 - k} kein Vielfaches der Schrittweite`);
            }

            for (let i = 1; i < 5; i++) {
                const upper = bounds[i]?.upperBound ?? Number.NaN;
                const expected = (bounds[i - 1]?.lowerBound ?? Number.NaN) - minPoints;
                if (Math.abs(upper - expected) > EPS) fail(`Lücke/Überlappung zwischen Note ${i} und ${i + 1}`);
            }

            // Grenzpunkte bekommen genau ihre Note, mehr Punkte nie eine schlechtere.
            let previousGrade = 6;
            for (const b of [...bounds].reverse()) {
                for (const p of [b.lowerBound, b.upperBound, b.upperBound + minPoints / 2]) {
                    if (p > maxPoints) continue;
                    const grade = gradeOf(p, bounds) ?? 0;
                    if (grade !== b.grade) fail(`${p} Punkte ergeben Note ${grade} statt ${b.grade}`);
                    if (grade > previousGrade) fail(`${p} Punkte ergeben eine schlechtere Note als weniger Punkte`);
                    previousGrade = grade;
                }
            }
        }

        expect(checked).toBeGreaterThan(500_000);
    });

    it("typische Notenschlüssel sind unverändert (lesbarer Snapshot)", () => {
        const typical = [
            [100, 0.5, 50],
            [100, 1, 50],
            [60, 0.5, 50],
            [48, 0.5, 60],
            [50, 0.1, 50],
            [40, 0.25, 55],
            [24, 1, 50],
            [120, 0.5, 45]
        ] as const;
        const result = Object.fromEntries(
            typical.map(([max, step, bp]) => [
                `max=${max} step=${step} bp=${bp}`,
                calculateGradeBounds(max, step, bp).map((b) => `${b.grade}: ${b.lowerBound} – ${b.upperBound}`)
            ])
        );
        expect(result).toMatchSnapshot();
    });

    it("alle Notenschlüssel im Raster sind unverändert (Prüfsumme)", async () => {
        const lines: string[] = [];
        for (const { maxPoints, minPoints, bp, bounds } of allScales()) {
            lines.push(`${maxPoints}|${minPoints}|${bp}|${JSON.stringify(bounds)}|${validateGradeBounds(bounds).length}`);
        }
        expect({ count: lines.length, sha256: await sha256(lines.join("\n")) }).toMatchSnapshot();
    });

    it("der Gesamtablauf liefert für zufällige Eingaben unveränderte Ergebnisse (Prüfsumme)", async () => {
        // Deterministischer Zufall: dieselben 3000 Eingaben bei jedem Lauf.
        let seed = 42;
        const rand = () => (seed = (seed * 1103515245 + 12345) % 2 ** 31) / 2 ** 31;
        const cells = ["Anna", "Ben", "", " ", "<x>", "=1+1", "12", "12,5", "12.5", "-3", "abc", "999999", "0", '"Mül;ler"'];
        const pick = () => cells[Math.floor(rand() * cells.length)] ?? "";

        const lines: string[] = [];
        for (let i = 0; i < 3000; i++) {
            const csv = i % 2 === 1;
            const sep = rand() < 0.5 ? ";" : ",";
            const rows = Math.floor(rand() * 15);
            const input: WorkflowInput = {
                maxPoints: [10, 50, 100, 120, 1000][i % 5] ?? 100,
                minPoints: STEPS[i % STEPS.length] ?? 1,
                breakPointPercent: BREAKPOINTS[i % BREAKPOINTS.length] ?? 50,
                inputMode: csv ? "csv" : "manual",
                csvContent: csv
                    ? (rand() < 0.5 ? `Name${sep}Punkte\n` : "") +
                      Array.from({ length: rows }, () => `${pick()}${sep}${pick()}`).join(rand() < 0.5 ? "\n" : "\r\n")
                    : "",
                manualEntries: csv ? [] : Array.from({ length: rows }, () => ({ name: pick(), points: pick() }))
            };
            lines.push(JSON.stringify(runCalculationWorkflow(input)));
        }
        expect(await sha256(lines.join("\n"))).toMatchSnapshot();
    });
});
