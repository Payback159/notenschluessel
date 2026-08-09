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
