// @ts-check
const { defineConfig, devices } = require('@playwright/test');

/**
 * Configuración de Playwright para Testing E2E
 * Sistema de Delegaciones de Compras
 */
module.exports = defineConfig({
  testDir: './tests/e2e',
  
  /* Timeout para tests individuales */
  timeout: 60 * 1000,
  
  /* Expect timeout */
  expect: {
    timeout: 10 * 1000,
  },
  
  /* Ejecutar tests en paralelo */
  fullyParallel: false, // Deshabilitado para evitar conflictos con cookies/sesiones
  
  /* Fallar el build si hay tests que fallan */
  forbidOnly: !!process.env.CI,
  
  /* Retry en CI */
  retries: process.env.CI ? 2 : 0,
  
  /* Número de workers */
  workers: process.env.CI ? 1 : 1, // Un solo worker para evitar conflictos
  
  /* Reporter */
  reporter: [
    ['html', { outputFolder: 'playwright-report' }],
    ['json', { outputFile: 'test-results/results.json' }],
    ['list']
  ],
  
  /* Configuración compartida para todos los proyectos */
  use: {
    /* URL base de la aplicación */
    baseURL: 'http://localhost:8081',
    
    /* Recolectar traces on retry */
    trace: 'on-first-retry',
    
    /* Screenshots en fallos */
    screenshot: 'only-on-failure',
    
    /* Video en fallos */
    video: 'retain-on-failure',
    
    /* Navegador en modo headless por defecto */
    headless: true,
    
    /* Viewport */
    viewport: { width: 1280, height: 720 },
  },

  /* Configurar proyectos para diferentes navegadores */
  projects: [
    {
      name: 'chromium',
      use: { ...devices['Desktop Chrome'] },
    },

    {
      name: 'firefox',
      use: { ...devices['Desktop Firefox'] },
    },

    {
      name: 'webkit',
      use: { ...devices['Desktop Safari'] },
    },

    /* Tests en dispositivos móviles */
    {
      name: 'Mobile Chrome',
      use: { ...devices['Pixel 5'] },
    },
    {
      name: 'Mobile Safari',
      use: { ...devices['iPhone 12'] },
    },
  ],

  /* Configuración del servidor web local */
  webServer: {
    command: 'echo "Servidor debe estar ejecutándose en localhost:8081"',
    port: 8081,
    reuseExistingServer: !process.env.CI,
    timeout: 120 * 1000,
  },
});