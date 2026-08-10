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
