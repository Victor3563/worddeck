import js from '@eslint/js'
import pluginVue from 'eslint-plugin-vue'
import globals from 'globals'

export default [
  { ignores: ['dist/**'] },
  js.configs.recommended,
  ...pluginVue.configs['flat/essential'],
  {
    languageOptions: {
      ecmaVersion: 'latest',
      sourceType: 'module',
      globals: { ...globals.browser },
    },
    rules: {
      'vue/multi-word-component-names': 'off', // допустимы имена из одного слова: Modal, Icon
      'vue/no-undef-properties': 'error', // имена из шаблона должны быть объявлены в <script setup>
      'vue/no-unused-properties': 'error',
      'vue/no-unused-refs': 'error',
      'vue/require-explicit-emits': 'error',
      'no-unused-vars': ['error', { argsIgnorePattern: '^_' }],
    },
  },
  {
    files: ['vite.config.js', 'eslint.config.js'],
    languageOptions: { globals: { ...globals.node } },
  },
]
