import js from "@eslint/js";
import globals from "globals";

const sharedRules = {
  "no-unused-vars": [
    "error",
    { argsIgnorePattern: "^_", varsIgnorePattern: "^_", caughtErrorsIgnorePattern: "^_" },
  ],
  "no-undef": "error",
  eqeqeq: ["error", "always", { null: "ignore" }],
  "no-var": "error",
  "prefer-const": "error",
  "no-duplicate-imports": "error",
};

export default [
  { ignores: ["dist/**", "node_modules/**"] },
  js.configs.recommended,
  {
    files: ["src/**/*.{js,jsx}"],
    languageOptions: {
      ecmaVersion: 2024,
      sourceType: "module",
      parserOptions: {
        ecmaFeatures: { jsx: true },
      },
      globals: {
        ...globals.browser,
        ...globals.node,
      },
    },
    rules: sharedRules,
  },
  {
    files: ["scripts/**/*.{js,mjs}", "vite.config.js", "eslint.config.js"],
    languageOptions: {
      ecmaVersion: 2024,
      sourceType: "module",
      globals: globals.node,
    },
    rules: sharedRules,
  },
];
