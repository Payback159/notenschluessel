import { calculateGradeBounds, validateGradeBounds } from "../core/calculator";
import { hasErrors, noValidStudents } from "../core/diagnostics";
import { calculateAverageGrade, processStudents } from "../core/grading";
import { validateStudentPoints } from "../core/studentValidation";
import { validateCoreInputs, validateInputMode } from "../core/validation";
import { parseCSVContent } from "../parsers/csvParser";
import { hasNonEmptyManualEntries, parseManualEntries } from "../parsers/manualParser";
import { Diagnostic, GradeBound, InputMode, ManualEntry, Student } from "../types";

export interface WorkflowInput {
    maxPoints: number;
    minPoints: number;
    breakPointPercent: number;
    inputMode: InputMode;
    csvContent?: string;
    manualEntries: ManualEntry[];
}

export interface WorkflowResult {
    ok: boolean;
    diagnostics: Diagnostic[];
    gradeBounds: GradeBound[];
    students: Student[];
    averageGrade: number;
}

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