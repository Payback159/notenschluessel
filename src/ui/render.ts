import { formatFixedNumberDe, formatPoints, formatRange } from "./format";
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

/**
 * Renders the average grade with exactly two decimals, German-style. Uses
 * `formatFixedNumberDe` rather than `formatNumberDe`: the latter trims trailing
 * zeros, which would render an average of exactly 2 as "2" instead of "2,00".
 */
export function renderAverage(
    heading: HTMLHeadingElement,
    averageGrade: number,
    hasStudents: boolean
): void {
    heading.textContent = hasStudents
        ? `Notendurchschnitt: ${formatFixedNumberDe(averageGrade, 2)}`
        : "";
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
