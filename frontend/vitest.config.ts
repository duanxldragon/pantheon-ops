import { defineConfig } from 'vitest/config'
import react from '@vitejs/plugin-react'

export default defineConfig({
  plugins: [react()],
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
        'src/core/permissions/**/*.{ts,tsx}',
        'src/core/refresh/**/*.{ts,tsx}',
        'src/core/runtime/**/*.{ts,tsx}',
        'src/core/settings/**/*.{ts,tsx}',
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
        'src/core/settings/publicSettings.ts',
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
