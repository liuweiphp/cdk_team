import js from '@eslint/js'
import vue from 'eslint-plugin-vue'
import vueTsConfig from '@vue/eslint-config-typescript'
import prettier from '@vue/eslint-config-prettier'

export default [
  js.configs.recommended,
  ...vue.configs['flat/recommended'],
  ...vueTsConfig(),
  prettier,
  {
    rules: {
      'vue/multi-word-component-names': 'off',
      'vue/require-default-prop': 'off',
      '@typescript-eslint/no-unused-vars': ['warn', { argsIgnorePattern: '^_' }],
      '@typescript-eslint/no-explicit-any': 'warn'
    }
  },
  {
    ignores: ['dist/**', 'node_modules/**', '*.d.ts']
  }
]
