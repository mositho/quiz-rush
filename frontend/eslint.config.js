import js from "@eslint/js"
import eslintConfigPrettier from "eslint-config-prettier"
import pluginVue from "eslint-plugin-vue"
import tseslint from "typescript-eslint"
import vueParser from "vue-eslint-parser"

export default [
  {
    ignores: ["dist/**", "node_modules/**"]
  },
  js.configs.recommended,
  ...tseslint.configs.recommended,
  ...pluginVue.configs["flat/recommended"],
  {
    files: ["**/*.vue"],
    languageOptions: {
      parser: vueParser,
      parserOptions: {
        parser: tseslint.parser,
        ecmaVersion: "latest",
        sourceType: "module"
      }
    }
  },
  eslintConfigPrettier
]
