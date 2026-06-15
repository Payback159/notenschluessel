import {
    exportCombinedCSV,
    exportGradeScaleCSV,
    exportStudentResultsCSV,
    triggerTextDownload
} from "./export/csvExport";
import {
    buildCombinedWorkbook,
    buildGradeScaleWorkbook,
    buildStudentsWorkbook,
    triggerWorkbookDownload
} from "./export/excelExport";
import { ManualEntry } from "./types";
import { runCalculationWorkflow } from "./ui/workflow";

interface AppState {
    maxPoints: number;
    minPoints: number;
    breakPointPercent: number;
    gradeBounds: ReturnType<typeof runCalculationWorkflow>["gradeBounds"];
    students: ReturnType<typeof runCalculationWorkflow>["students"];
    averageGrade: number;
}

const state: AppState = {
    maxPoints: 100,
    minPoints: 0.5,
    breakPointPercent: 50,
    gradeBounds: [],
    students: [],
    averageGrade: 0
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

function addManualRow(name = "", points = ""): void {
    const rows = getById<HTMLTableSectionElement>("manualRows");
    const tr = document.createElement("tr");
    tr.innerHTML =
        `<td><input type="text" name="manualName" value="${name.replaceAll('"', '&quot;')}" /></td>` +
        `<td><input type="text" name="manualPoints" value="${points.replaceAll('"', '&quot;')}" /></td>` +
        `<td><button type="button" class="remove-row">Entfernen</button></td>`;
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

function showMessage(type: "error" | "success", text: string): void {
    const msg = getById<HTMLDivElement>("message");
    msg.classList.remove("hidden", "error", "success");
    msg.classList.add(type);
    msg.textContent = text;
}

function hideMessage(): void {
    const msg = getById<HTMLDivElement>("message");
    msg.classList.add("hidden");
    msg.textContent = "";
}

function renderResults(): void {
    const scaleCard = getById<HTMLDivElement>("gradeScaleCard");
    const studentsCard = getById<HTMLDivElement>("studentsCard");
    const scaleBody = getById<HTMLTableSectionElement>("gradeScaleBody");
    const studentsBody = getById<HTMLTableSectionElement>("studentsBody");
    const average = getById<HTMLHeadingElement>("averageGrade");

    scaleBody.innerHTML = "";
    for (const bound of state.gradeBounds) {
        const tr = document.createElement("tr");
        tr.className = `grade-${bound.grade}`;
        tr.innerHTML = `<td>${bound.grade}</td><td>${bound.lowerBound.toFixed(1)} - ${bound.upperBound.toFixed(1)}</td>`;
        scaleBody.appendChild(tr);
    }

    studentsBody.innerHTML = "";
    for (const student of state.students) {
        const tr = document.createElement("tr");
        if (student.grade) {
            tr.className = `grade-${student.grade}`;
        }
        tr.innerHTML = `<td>${student.name}</td><td>${student.points.toFixed(1)}</td><td>${student.grade ?? ""}</td>`;
        studentsBody.appendChild(tr);
    }

    average.textContent = state.students.length > 0
        ? `Notendurchschnitt: ${state.averageGrade.toFixed(2)}`
        : "";

    scaleCard.classList.toggle("hidden", state.gradeBounds.length === 0);
    studentsCard.classList.toggle("hidden", state.students.length === 0);
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

    const maxPoints = Number.parseInt(getById<HTMLInputElement>("maxPoints").value, 10);
    const minPoints = Number.parseFloat(getById<HTMLInputElement>("minPoints").value);
    const breakPointPercent = Number.parseFloat(getById<HTMLInputElement>("breakPointPercent").value);
    const modeInput = document.querySelector<HTMLInputElement>('input[name="inputMode"]:checked');
    const inputMode = (modeInput?.value ?? "csv") as "csv" | "manual";

    let csvContent = "";
    if (inputMode === "csv") {
        const fileInput = getById<HTMLInputElement>("csvFile");
        const selectedFile = fileInput.files?.[0];
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
        showMessage("error", result.errors.join(" "));
        state.gradeBounds = [];
        state.students = [];
        state.averageGrade = 0;
        renderResults();
        return;
    }

    state.maxPoints = maxPoints;
    state.minPoints = minPoints;
    state.breakPointPercent = breakPointPercent;
    state.gradeBounds = result.gradeBounds;
    state.students = result.students;
    state.averageGrade = result.averageGrade;

    sessionStorage.setItem("notenschluessel:lastState", JSON.stringify(state));
    renderResults();
    showMessage("success", "Berechnung abgeschlossen.");
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
        triggerTextDownload(exportGradeScaleCSV(state.gradeBounds), "grade-scale.csv", "text/csv;charset=utf-8;");
    });

    getById<HTMLButtonElement>("downloadStudentsCsvBtn").addEventListener("click", () => {
        triggerTextDownload(exportStudentResultsCSV(state.students), "student-results.csv", "text/csv;charset=utf-8;");
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

    getById<HTMLButtonElement>("downloadScaleXlsxBtn").addEventListener("click", () => {
        triggerWorkbookDownload(buildGradeScaleWorkbook(state.gradeBounds), "grade-scale.xlsx");
    });

    getById<HTMLButtonElement>("downloadStudentsXlsxBtn").addEventListener("click", () => {
        triggerWorkbookDownload(buildStudentsWorkbook(state.students), "student-results.xlsx");
    });

    getById<HTMLButtonElement>("downloadCombinedXlsxBtn").addEventListener("click", () => {
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
        const restored = JSON.parse(raw) as AppState;
        state.maxPoints = restored.maxPoints;
        state.minPoints = restored.minPoints;
        state.breakPointPercent = restored.breakPointPercent;
        state.gradeBounds = restored.gradeBounds;
        state.students = restored.students;
        state.averageGrade = restored.averageGrade ?? 0;

        getById<HTMLInputElement>("maxPoints").value = String(restored.maxPoints);
        getById<HTMLInputElement>("minPoints").value = String(restored.minPoints);
        getById<HTMLInputElement>("breakPointPercent").value = String(restored.breakPointPercent);

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