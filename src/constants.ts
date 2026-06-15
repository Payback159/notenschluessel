export const LIMITS = {
    maxPoints: 1000,
    minBreakPointPercent: 1,
    maxBreakPointPercent: 99,
    maxStudents: 10000,
    maxNameLength: 200,
    maxUploadSizeBytes: 10 * 1024 * 1024
} as const;

export const GRADE_COLORS = {
    1: { bg: "#d4edda", border: "#a3d9a3", text: "#155724" },
    2: { bg: "#c3e6cb", border: "#8fd5a3", text: "#155724" },
    3: { bg: "#ffeaa7", border: "#ffda75", text: "#856404" },
    4: { bg: "#ffe5cc", border: "#ffb380", text: "#7a4419" },
    5: { bg: "#f8d7da", border: "#f5b4b4", text: "#721c24" }
} as const;