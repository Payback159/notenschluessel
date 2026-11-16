import { LIMITS } from "../constants";

export function sanitizeName(name: string): string {
    return name
        .replace(/[<>]/g, "")
        .replace(/[\r\n\t]/g, " ")
        .trim()
        .slice(0, LIMITS.maxNameLength);
}
