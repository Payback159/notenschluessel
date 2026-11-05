import { LIMITS } from "./constants";
import { fileTooLarge } from "./core/diagnostics";
import { isValidRestoredState, PersistedState } from "./core/validation";
import {
    exportCombinedCSV,
    exportGradeScaleCSV,
    exportStudentResultsCSV,
    triggerTextDownload
} from "./export/csvExport";
import { Diagnostic, InputMode, ManualEntry } from "./types";
import { formatPoints } from "./ui/format";
import { renderAverage, renderDiagnostics, renderGradeScale, renderStudents, summaryLine } from "./ui/render";
import { runCalculationWorkflow } from "./ui/workflow";

const state: PersistedState = {
    maxPoints: 100,
    minPoints: 0.5,
    breakPointPercent: 50,
    gradeBounds: [],
    students: [],
    averageGrade: 0,
    inputMode: "csv"
};

function getById<T extends HTMLElement>(id: string): T {
    const element = document.getElementById(id);
    if (!element) {
        throw new Error(`Missing element with id '${id}'`);
    }
    return element as T;
}

function readFileAsText(file: File): Promise<string> {
    return new Promise((resolve, reject) => {
        const reader = new FileReader();
        reader.onload = () => resolve(String(reader.result ?? ""));
        reader.onerror = () => reject(reader.error ?? new Error("Datei konnte nicht gelesen werden"));
        reader.readAsText(file);
    });
}

function manualInputCell(name: string, value: string): HTMLTableCellElement {
    const td = document.createElement("td");
    const input = document.createElement("input");
    input.type = "text";
    input.name = name;
    input.value = value;
    td.appendChild(input);
    return td;
}

/**
 * Builds the row via createElement and sets input values through the `value`
 * property, not markup — an innerHTML sink here would only be safe today
 * because every call site passes no arguments, and a future caller passing a
 * restored name would otherwise land in an unescaped attribute context.
 */
function addManualRow(name = "", points = ""): void {
    const rows = getById<HTMLTableSectionElement>("manualRows");
    const tr = document.createElement("tr");

    tr.appendChild(manualInputCell("manualName", name));
    tr.appendChild(manualInputCell("manualPoints", points));

    const buttonCell = document.createElement("td");
    const removeButton = document.createElement("button");
    removeButton.type = "button";
    removeButton.className = "remove-row";
    removeButton.textContent = "Entfernen";
    buttonCell.appendChild(removeButton);
    tr.appendChild(buttonCell);

    rows.appendChild(tr);
}

function collectManualEntries(): ManualEntry[] {
    const rows = getById<HTMLTableSectionElement>("manualRows").querySelectorAll("tr");
    const entries: ManualEntry[] = [];

    rows.forEach((row) => {
        const nameInput = row.querySelector<HTMLInputElement>('input[name="manualName"]');
        const pointsInput = row.querySelector<HTMLInputElement>('input[name="manualPoints"]');
        entries.push({
            name: nameInput?.value ?? "",
            points: pointsInput?.value ?? ""
        });
    });

    return entries;
}

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

    renderAverage(
        getById<HTMLHeadingElement>("averageGrade"),
        state.averageGrade,
        state.students.length > 0
    );

    getById<HTMLDivElement>("gradeScaleCard").classList.toggle("hidden", state.gradeBounds.length === 0);
    getById<HTMLDivElement>("studentsCard").classList.toggle("hidden", state.students.length === 0);
}

function resetInputs(): void {
    state.gradeBounds = [];
    state.students = [];
    state.averageGrade = 0;
    sessionStorage.removeItem("notenschluessel:lastState");

    // Auch die eingegebenen Schülerdaten aus den Formularfeldern entfernen,
    // nicht nur den gespeicherten Zustand.
    getById<HTMLInputElement>("csvFile").value = "";
    const rows = getById<HTMLTableSectionElement>("manualRows");
    rows.innerHTML = "";
    addManualRow();

    renderResults();
    showMessage("success", "Eingaben und Ergebnisse wurden zurückgesetzt.");
}

async function handleSubmit(event: SubmitEvent): Promise<void> {
    event.preventDefault();
    hideMessage();

    const maxPoints = Number(getById<HTMLInputElement>("maxPoints").value);
    const minPoints = Number.parseFloat(getById<HTMLInputElement>("minPoints").value);
    const breakPointPercent = Number.parseFloat(getById<HTMLInputElement>("breakPointPercent").value);
    const modeInput = document.querySelector<HTMLInputElement>('input[name="inputMode"]:checked');
    const inputMode = (modeInput?.value ?? "csv") as InputMode;

    let csvContent = "";
    if (inputMode === "csv") {
        const fileInput = getById<HTMLInputElement>("csvFile");
        const selectedFile = fileInput.files?.[0];
        if (selectedFile && selectedFile.size > LIMITS.maxUploadSizeBytes) {
            showMessage("error", "Die Berechnung wurde abgebrochen.", [fileTooLarge()]);
            state.gradeBounds = [];
            state.students = [];
            state.averageGrade = 0;
            sessionStorage.removeItem("notenschluessel:lastState");
            renderResults();
            return;
        }
        csvContent = selectedFile ? await readFileAsText(selectedFile) : "";
    }

    const result = runCalculationWorkflow({
        maxPoints,
        minPoints,
        breakPointPercent,
        inputMode,
        csvContent,
        manualEntries: collectManualEntries()
    });

    if (!result.ok) {
        showMessage("error", "Die Berechnung wurde abgebrochen.", result.diagnostics);
        state.gradeBounds = [];
        state.students = [];
        state.averageGrade = 0;
        sessionStorage.removeItem("notenschluessel:lastState");
        renderResults();
        return;
    }

    state.maxPoints = maxPoints;
    state.minPoints = minPoints;
    state.breakPointPercent = breakPointPercent;
    state.gradeBounds = result.gradeBounds;
    state.students = result.students;
    state.averageGrade = result.averageGrade;
    state.inputMode = inputMode;

    sessionStorage.setItem("notenschluessel:lastState", JSON.stringify(state));
    renderResults();
    showMessage(
        "success",
        summaryLine(result.students.length, result.diagnostics.length > 0),
        result.diagnostics
    );
}

function setupInputModeToggle(): void {
    const csvSection = getById<HTMLDivElement>("csvSection");
    const manualSection = getById<HTMLDivElement>("manualSection");

    document.querySelectorAll<HTMLInputElement>('input[name="inputMode"]').forEach((radio) => {
        radio.addEventListener("change", () => {
            const mode = (document.querySelector<HTMLInputElement>('input[name="inputMode"]:checked')?.value ?? "csv");
            
            if (mode === "csv") {
                csvSection.classList.remove("hidden");
                manualSection.classList.add("hidden");
                // Clear manual entries when switching to CSV
                const rows = getById<HTMLTableSectionElement>("manualRows");
                rows.innerHTML = "";
                addManualRow();
            } else {
                csvSection.classList.add("hidden");
                manualSection.classList.remove("hidden");
                // Clear CSV file input when switching to manual
                getById<HTMLInputElement>("csvFile").value = "";
            }
        });
    });
}

function setupExports(): void {
    getById<HTMLButtonElement>("downloadScaleCsvBtn").addEventListener("click", () => {
        triggerTextDownload(
            exportGradeScaleCSV(state.gradeBounds, state.minPoints),
            "grade-scale.csv",
            "text/csv;charset=utf-8;"
        );
    });

    getById<HTMLButtonElement>("downloadStudentsCsvBtn").addEventListener("click", () => {
        triggerTextDownload(
            exportStudentResultsCSV(state.students, state.minPoints),
            "student-results.csv",
            "text/csv;charset=utf-8;"
        );
    });

    getById<HTMLButtonElement>("downloadCombinedCsvBtn").addEventListener("click", () => {
        triggerTextDownload(
            exportCombinedCSV(state.gradeBounds, state.students, {
                maxPoints: state.maxPoints,
                minPoints: state.minPoints,
                breakPointPercent: state.breakPointPercent
            }),
            "combined.csv",
            "text/csv;charset=utf-8;"
        );
    });

    getById<HTMLButtonElement>("downloadScaleXlsxBtn").addEventListener("click", async () => {
        const { buildGradeScaleWorkbook, triggerWorkbookDownload } = await import("./export/excelExport");
        triggerWorkbookDownload(buildGradeScaleWorkbook(state.gradeBounds), "grade-scale.xlsx");
    });

    getById<HTMLButtonElement>("downloadStudentsXlsxBtn").addEventListener("click", async () => {
        const { buildStudentsWorkbook, triggerWorkbookDownload } = await import("./export/excelExport");
        triggerWorkbookDownload(buildStudentsWorkbook(state.students), "student-results.xlsx");
    });

    getById<HTMLButtonElement>("downloadCombinedXlsxBtn").addEventListener("click", async () => {
        const { buildCombinedWorkbook, triggerWorkbookDownload } = await import("./export/excelExport");
        triggerWorkbookDownload(
            buildCombinedWorkbook(state.gradeBounds, state.students, {
                maxPoints: state.maxPoints,
                minPoints: state.minPoints,
                breakPointPercent: state.breakPointPercent
            }),
            "combined.xlsx"
        );
    });
}

function setupPrivacyModal(): void {
    const modal = document.getElementById("privacyModal") as HTMLDialogElement | null;
    if (!modal) {
        return;
    }

    getById<HTMLButtonElement>("privacyBtn").addEventListener("click", () => {
        modal.showModal();
    });

    getById<HTMLButtonElement>("privacyCloseBtn").addEventListener("click", () => {
        modal.close();
    });

    modal.addEventListener("click", (event) => {
        if (event.target === modal) {
            modal.close();
        }
    });
}

function restoreState(): void {
    const raw = sessionStorage.getItem("notenschluessel:lastState");
    if (!raw) {
        return;
    }

    try {
        const parsed: unknown = JSON.parse(raw);
        if (!isValidRestoredState(parsed)) {
            sessionStorage.removeItem("notenschluessel:lastState");
            return;
        }

        state.maxPoints = parsed.maxPoints;
        state.minPoints = parsed.minPoints;
        state.breakPointPercent = parsed.breakPointPercent;
        state.gradeBounds = parsed.gradeBounds;
        state.students = parsed.students;
        state.averageGrade = parsed.averageGrade;
        state.inputMode = parsed.inputMode ?? "csv";

        getById<HTMLInputElement>("maxPoints").value = String(parsed.maxPoints);
        getById<HTMLInputElement>("minPoints").value = String(parsed.minPoints);
        getById<HTMLInputElement>("breakPointPercent").value = String(parsed.breakPointPercent);

        if (state.inputMode === "manual" && parsed.students.length > 0) {
            const manualRadio = document.querySelector<HTMLInputElement>('input[name="inputMode"][value="manual"]');
            if (manualRadio) {
                manualRadio.checked = true;
            }
            getById<HTMLDivElement>("csvSection").classList.add("hidden");
            getById<HTMLDivElement>("manualSection").classList.remove("hidden");

            const rows = getById<HTMLTableSectionElement>("manualRows");
            rows.replaceChildren();
            for (const student of parsed.students) {
                addManualRow(student.name, formatPoints(student.points, parsed.minPoints));
            }
        }

        renderResults();
    } catch {
        sessionStorage.removeItem("notenschluessel:lastState");
    }
}

function setupApp(): void {
    addManualRow();
    setupInputModeToggle();
    setupExports();
    setupPrivacyModal();
    restoreState();

    getById<HTMLButtonElement>("addRowBtn").addEventListener("click", () => addManualRow());
    getById<HTMLButtonElement>("resetBtn").addEventListener("click", resetInputs);
    getById<HTMLFormElement>("calcForm").addEventListener("submit", (event) => {
        void handleSubmit(event as SubmitEvent);
    });

    getById<HTMLTableSectionElement>("manualRows").addEventListener("click", (event) => {
        const target = event.target as HTMLElement;
        if (!target.classList.contains("remove-row")) {
            return;
        }

        const row = target.closest("tr");
        if (!row) {
            return;
        }

        const rows = getById<HTMLTableSectionElement>("manualRows").querySelectorAll("tr");
        if (rows.length <= 1) {
            const nameInput = row.querySelector<HTMLInputElement>('input[name="manualName"]');
            const pointsInput = row.querySelector<HTMLInputElement>('input[name="manualPoints"]');
            if (nameInput) {
                nameInput.value = "";
            }
            if (pointsInput) {
                pointsInput.value = "";
            }
            return;
        }

        row.remove();
    });
}

setupApp();