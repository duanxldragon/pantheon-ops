import { defineConfig } from 'vitest/config'
import react from '@vitejs/plugin-react'
import path from 'node:path'

export default defineConfig({
  plugins: [react()],
  // Keeps the `@/` alias from tsconfig.app.json / vite.config.ts working in tests.
  resolve: {
    alias: {
      '@': path.resolve(__dirname, 'src'),
    },
  },
  test: {
    environment: 'jsdom',
    globals: true,
    include: ['tests/unit/**/*.{test,spec}.{ts,tsx}'],
    coverage: {
      provider: 'v8',
      reporter: ['text', 'json-summary', 'lcov'],
      reportsDirectory: './coverage',
      include: [
        'src/core/arco/**/*.{ts,tsx}',
        'src/core/format/**/*.{ts,tsx}',
        'src/core/permission/**/*.{ts,tsx}',
        'src/core/refresh/**/*.{ts,tsx}',
        'src/core/runtime/**/*.{ts,tsx}',
        'src/core/setting/**/*.{ts,tsx}',
        'src/core/theme/**/*.{ts,tsx}',
        'src/hooks/**/*.{ts,tsx}',
      ],
      exclude: [
        'src/**/*.d.ts',
        'src/i18n/**',
        'src/modules/generated/**',
        // CSS-only side-effect imports — no testable exports
        'src/core/arco/style.ts',
        // HTTP API wrappers — tested via integration, not unit
        'src/core/refresh/api.ts',
        // React hooks/components with heavy external deps — covered by e2e
        'src/core/theme/ThemeSwitcher.tsx',
        'src/core/theme/theme.ts',
        'src/core/setting/publicSettings.ts',
        // Hooks with HTTP/Zustand deps — integration-level, not unit-testable
        'src/core/refresh/refreshBus.ts',
        // Re-export barrels — no logic to test
        'src/hooks/index.ts',
        // Hooks that depend on Zustand store internals
        'src/hooks/usePermission.ts',
        'src/hooks/useTheme.ts',
      ],
      thresholds: {
        perFile: true,
        lines: 80,
        functions: 80,
        branches: 70,
        statements: 80,
      },
    },
  },
})
