import js from '@eslint/js'
import globals from 'globals'
import reactHooks from 'eslint-plugin-react-hooks'
import reactRefresh from 'eslint-plugin-react-refresh'
import tseslint from 'typescript-eslint'
import { defineConfig, globalIgnores } from 'eslint/config'

export default defineConfig([
  globalIgnores(['dist']),
  {
    files: ['**/*.{ts,tsx}'],
    extends: [
      js.configs.recommended,
      tseslint.configs.recommended,
      reactHooks.configs.flat.recommended,
      reactRefresh.configs.vite,
    ],
    languageOptions: {
      ecmaVersion: 2020,
      globals: globals.browser,
    },
    rules: {
      // The project fetches data inside effects and calls setState in the
      // resulting (async) callback — a benign, intentional pattern. This
      // React 19 rule is overly strict for that and is disabled project-wide
      // rather than refactoring every fetch effect.
      "react-hooks/set-state-in-effect": "off",
      // Fast-refresh only matters for HMR ergonomics; exporting small helpers
      ///constants alongside components is intentional here.
      "react-refresh/only-export-components": "off",
    },
  },
])
