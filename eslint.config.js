import { defineConfig } from "eslint/config";
import globals from "globals";
import pluginVue from "eslint-plugin-vue";

export default defineConfig([
  ...pluginVue.configs["flat/recommended"],
  {
    files: ["src/**/*.vue"],
    rules: {
      "vue/no-v-html": "off",
      "vue/multi-word-component-names": "off",
      "vue/max-attributes-per-line": "off",
      "vue/html-self-closing": "off",
      "vue/html-indent": "off",
      "vue/singleline-html-element-content-newline": "off",
      "vue/no-deprecated-v-on-native-modifier": "warn",
      "vue/no-deprecated-router-link-tag-prop": "warn",
      "vue/no-deprecated-v-bind-sync": "warn",
      "vue/attribute-hyphenation": "off",
    },
    languageOptions: {
      sourceType: "module",
      globals: {
        ...globals["shared-node-browser"],
      },
    },
  },
]);
