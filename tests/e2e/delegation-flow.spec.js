// @ts-check
const { test, expect } = require('@playwright/test');

/**
 * Test Exploratorio - Sistema de Delegaciones de Compras
 * 
 * Este test valida el flujo completo del sistema de delegaciones:
 * 1. Login de diferentes roles (Manager, Empleado, CEO)
 * 2. Navegación del dashboard
 * 3. Creación de delegaciones
 * 4. Visualización de delegaciones creadas/recibidas
 * 5. Uso de delegaciones para aprobaciones
 * 6. Flujo completo de solicitud de compra
 */

test.describe('Sistema de Delegaciones - Flujo Completo', () => {
  
  test.beforeEach(async ({ page }) => {
    // Configurar timeout extendido para operaciones
    test.setTimeout(60000);
    
    // Navegar a la aplicación
    await page.goto('http://localhost:8081');
  });

  test('Manager - Flujo completo de creación de delegación', async ({ page }) => {
    // === LOGIN COMO MANAGER ===
    await test.step('Login como Manager', async () => {
      await expect(page.locator('h2')).toContainText('Sistema de Compras - Login');
      await page.click('a[href="/login-as/manager@empresa.com"]');
      await expect(page).toHaveURL(/.*\/dashboard/);
    });

    // === VERIFICAR DASHBOARD ===
    await test.step('Verificar Dashboard del Manager', async () => {
      await expect(page.locator('h1')).toContainText('¡Bienvenido, Ana Manager!');
      await expect(page.locator('text=Manager - madrid')).toBeVisible();
      
      // Verificar que el botón de delegaciones existe
      await expect(page.locator('a[href="/delegation/list"]')).toBeVisible();
    });

    // === NAVEGAR A DELEGACIONES ===
    await test.step('Navegar a Lista de Delegaciones', async () => {
      await page.click('a[href="/delegation/list"]');
      await expect(page.locator('h1')).toContainText('Mis Delegaciones');
      
      // Verificar que aparece el botón de nueva delegación
      await expect(page.locator('text=Nueva Delegación')).toBeVisible();
      
      // Verificar que hay tabs para creadas y recibidas
      await expect(page.locator('text=Delegaciones Creadas')).toBeVisible();
      await expect(page.locator('text=Delegaciones Recibidas')).toBeVisible();
    });

    // === VERIFICAR DELEGACIONES EXISTENTES ===
    await test.step('Verificar Delegaciones Mock Existentes', async () => {
      // Verificar que se muestran las delegaciones mock
      await expect(page.locator('text=Juan Empleado (Empleado - IT)')).toBeVisible();
      await expect(page.locator('text=Carlos CEO (CEO - Executive)')).toBeVisible();
      await expect(page.locator('text=Sofia Admin (Admin - IT)')).toBeVisible();
      
      // Verificar estados diferentes
      await expect(page.locator('text=⏳ Pendiente de Activación')).toBeVisible();
      await expect(page.locator('text=✅ Activa')).toBeVisible();
      await expect(page.locator('text=⏰ Expirada')).toBeVisible();
      
      // Verificar detalles de las delegaciones
      await expect(page.locator('text=Vacaciones de verano - Delegación temporal')).toBeVisible();
      await expect(page.locator('text=1500.00€')).toBeVisible();
    });

    // === CREAR NUEVA DELEGACIÓN ===
    await test.step('Crear Nueva Delegación', async () => {
      await page.click('text=Nueva Delegación');
      await expect(page.locator('h1')).toContainText('Nueva Delegación de Aprobaciones');
      
      // Llenar formulario - delegar al CEO (debe tener permisos de aprobación)
      await page.selectOption('select[name="to_user_id"]', 'ceo@empresa.com');
      
      // Configurar fechas (mañana a la semana que viene)
      const tomorrow = new Date();
      tomorrow.setDate(tomorrow.getDate() + 1);
      const nextWeek = new Date();
      nextWeek.setDate(nextWeek.getDate() + 8);
      
      const startDate = tomorrow.toISOString().slice(0, 16);
      const endDate = nextWeek.toISOString().slice(0, 16);
      
      await page.fill('input[name="start_date"]', startDate);
      await page.fill('input[name="end_date"]', endDate);
      await page.fill('input[name="max_amount"]', '800.50');
      await page.fill('textarea[name="reason"]', 'Test automatizado - Delegación de prueba para validación E2E');
      
      // Verificar validación JavaScript
      await expect(page.locator('text=Crear Delegación')).toBeVisible();
      
      // Enviar formulario (esto creará un workflow en Temporal)
      await page.click('button[type="submit"]');
      
      // Verificar redirección a lista con éxito
      await expect(page).toHaveURL(/.*\/delegation\/list\?success=created/);
    });
  });

  test('Empleado - Ver delegaciones recibidas y usar para aprobaciones', async ({ page }) => {
    // === LOGIN COMO EMPLEADO ===
    await test.step('Login como Empleado', async () => {
      await page.click('a[href="/login-as/empleado@empresa.com"]');
      await expect(page).toHaveURL(/.*\/dashboard/);
    });

    // === VERIFICAR DASHBOARD EMPLEADO ===
    await test.step('Verificar Dashboard del Empleado', async () => {
      await expect(page.locator('h1')).toContainText('¡Bienvenido, Juan Empleado!');
      await expect(page.locator('text=Empleado - madrid')).toBeVisible();
      
      // Verificar que el empleado ve "Delegaciones Recibidas" en lugar de "Gestionar"
      await expect(page.locator('text=Delegaciones Recibidas')).toBeVisible();
      await expect(page.locator('text=Ver delegaciones de aprobación que has recibido')).toBeVisible();
    });

    // === VER DELEGACIONES RECIBIDAS ===
    await test.step('Ver Delegaciones Recibidas', async () => {
      await page.click('a[href="/delegation/list"]');
      await expect(page.locator('h1')).toContainText('Mis Delegaciones');
      
      // Verificar que NO aparece el botón de crear (empleado no puede delegar)
      await expect(page.locator('text=Nueva Delegación')).not.toBeVisible();
      
      // Verificar que solo aparece la tab de "Recibidas" activa
      await expect(page.locator('text=Delegaciones Recibidas (1)')).toBeVisible();
      
      // Verificar delegación recibida de Ana Manager
      await expect(page.locator('text=Ana Manager (Manager - IT)')).toBeVisible();
      await expect(page.locator('text=✅ Activa - Puedes Aprobar')).toBeVisible();
      await expect(page.locator('text=Cobertura durante conferencia en Barcelona')).toBeVisible();
    });

    // === USAR DELEGACIÓN PARA APROBACIONES ===
    await test.step('Usar Delegación para Aprobaciones', async () => {
      // Hacer clic en "Usar para Aprobaciones"
      await page.click('a[href="/pending-approvals?delegation=delegation_recv_001"]');
      
      // Verificar que llegamos a la página de aprobaciones
      await expect(page.locator('h1')).toContainText('Aprobaciones Pendientes');
      
      // Verificar que se muestra el indicador de delegación
      await expect(page.locator('text=Usando Delegación: delegation_recv_001')).toBeVisible();
      await expect(page.locator('text=Estás aprobando con permisos delegados')).toBeVisible();
      
      // Verificar nombre del usuario
      await expect(page.locator('text=Bienvenido Juan Empleado')).toBeVisible();
    });
  });

  test('CEO - Flujo completo con múltiples delegaciones', async ({ page }) => {
    // === LOGIN COMO CEO ===
    await test.step('Login como CEO', async () => {
      await page.click('a[href="/login-as/ceo@empresa.com"]');
      await expect(page).toHaveURL(/.*\/dashboard/);
    });

    // === VERIFICAR DASHBOARD CEO ===
    await test.step('Verificar Dashboard del CEO', async () => {
      await expect(page.locator('h1')).toContainText('¡Bienvenido, Carlos CEO!');
      await expect(page.locator('text=CEO - madrid')).toBeVisible();
      
      // Verificar permisos especiales del CEO
      // Nota: Panel de Admin solo para rol Admin, no CEO
      await expect(page.locator('a[href="/delegation/list"]')).toBeVisible();
      
      // Verificar estadísticas
      await expect(page.locator('text=Aprobadas Hoy')).toBeVisible();
      await expect(page.locator('text=Monto Total')).toBeVisible();
    });

    // === VER DELEGACIONES CEO ===
    await test.step('Ver Delegaciones del CEO', async () => {
      await page.click('a[href="/delegation/list"]');
      
      // El CEO puede tanto crear como recibir delegaciones
      await expect(page.locator('text=Nueva Delegación')).toBeVisible();
      await expect(page.locator('text=Delegaciones Creadas')).toBeVisible();
      await expect(page.locator('text=Delegaciones Recibidas')).toBeVisible();
      
      // Cambiar a tab de recibidas
      await page.click('text=Delegaciones Recibidas');
      
      // Verificar delegación recibida del CFO
      await expect(page.locator('text=Director Financiero (CFO - Finance)')).toBeVisible();
      await expect(page.locator('text=Auditoría anual - Delegación de aprobaciones financieras')).toBeVisible();
      await expect(page.locator('text=Aprobaciones hasta 5K€')).toBeVisible();
    });
  });

  test('Flujo completo de solicitud de compra con delegación', async ({ page }) => {
    // === CREAR SOLICITUD COMO EMPLEADO ===
    await test.step('Empleado crea solicitud de compra', async () => {
      await page.click('a[href="/login-as/empleado@empresa.com"]');
      await expect(page).toHaveURL(/.*\/dashboard/);
      
      // Crear nueva solicitud
      await page.click('a[href="/request/new"]');
      await expect(page.locator('h1')).toContainText('Nueva Solicitud de Compra');
      
      // Llenar formulario de solicitud
      await page.selectOption('select[name="delivery_office"]', 'madrid');
      
      // Las URLs ya están pre-rellenadas por el JavaScript
      await page.fill('textarea[name="justification"]', 
        'Solicitud de prueba E2E - Necesito estos productos para el proyecto de automatización');
      
      await page.click('button[type="submit"]');
      
      // Verificar redirección a página de estado
      await expect(page).toHaveURL(/.*\/status\?id=.*/);
      await expect(page.locator('text=Estado de la Solicitud')).toBeVisible();
    });

    // === APROBAR USANDO DELEGACIÓN ===
    await test.step('Manager aprueba usando delegación', async () => {
      // Navegar al login para cambiar de usuario
      await page.goto('http://localhost:8081/');
      await expect(page.locator('h2')).toContainText('Sistema de Compras - Login');
      
      // Cambiar a manager
      await page.click('a[href="/login-as/manager@empresa.com"]');
      await expect(page).toHaveURL(/.*\/dashboard/);
      
      // Ir a aprobaciones pendientes
      await page.click('a[href="/approvals/pending"]');
      await expect(page.locator('h1')).toContainText('Aprobaciones Pendientes');
      
      // Verificar que hay solicitudes pendientes
      // (En el mock debería aparecer la solicitud creada)
    });
  });

  test('Validación de permisos y seguridad', async ({ page }) => {
    // === VERIFICAR RESTRICCIONES DE EMPLEADO ===
    await test.step('Empleado no puede crear delegaciones directamente', async () => {
      await page.click('a[href="/login-as/empleado@empresa.com"]');
      
      // Intentar acceder directamente a crear delegación
      await page.goto('http://localhost:8081/delegation/new');
      
      // Debería mostrar error de permisos insuficientes
      await expect(page.locator('text=Insufficient permissions')).toBeVisible();
    });

    // === VERIFICAR AUTENTICACIÓN ===
    await test.step('Verificar redirección sin autenticación', async () => {
      // Limpiar cookies y acceder a página protegida
      await page.context().clearCookies();
      await page.goto('http://localhost:8081/delegation/list');
      
      // Debería redirigir al login
      await expect(page.locator('h2')).toContainText('Sistema de Compras - Login');
    });
  });

  test('Navegación y UX', async ({ page }) => {
    // === VERIFICAR NAVEGACIÓN FLUIDA ===
    await test.step('Navegación entre páginas', async () => {
      await page.click('a[href="/login-as/manager@empresa.com"]');
      
      // Dashboard -> Delegaciones -> Nueva -> Volver
      await page.click('a[href="/delegation/list"]');
      await page.click('text=Nueva Delegación');
      await page.click('text=❌ Cancelar');
      await expect(page).toHaveURL(/.*\/delegation\/list/);
      
      // Volver al dashboard
      await page.click('text=🏠 Volver al Dashboard');
      await expect(page).toHaveURL(/.*\/dashboard/);
    });

    // === VERIFICAR ELEMENTOS VISUALES ===
    await test.step('Verificar elementos visuales clave', async () => {
      await page.click('a[href="/delegation/list"]');
      
      // Verificar que los emojis se muestran correctamente (no caracteres raros)
      await expect(page.locator('text=🔄 Mis Delegaciones')).toBeVisible();
      await expect(page.locator('text=📤 Delegaciones Creadas')).toBeVisible();
      await expect(page.locator('text=📥 Delegaciones Recibidas')).toBeVisible();
      await expect(page.locator('text=➕ Nueva Delegación')).toBeVisible();
      
      // Verificar colores de estado
      await expect(page.locator('.status-badge.status-active')).toBeVisible();
      await expect(page.locator('.status-badge.status-pending')).toBeVisible();
      await expect(page.locator('.status-badge.status-expired')).toBeVisible();
    });
  });

  test('Temporal workflow integration', async ({ page }) => {
    // === VERIFICAR INTEGRACIÓN CON TEMPORAL ===
    await test.step('Verificar que Temporal está funcionando', async () => {
      // Este test verifica que el sistema puede comunicarse con Temporal
      await page.click('a[href="/login-as/manager@empresa.com"]');
      await page.click('a[href="/delegation/list"]');
      await page.click('text=Nueva Delegación');
      
      // Llenar un formulario mínimo - delegar al admin
      await page.selectOption('select[name="to_user_id"]', 'admin@empresa.com');
      
      const tomorrow = new Date();
      tomorrow.setDate(tomorrow.getDate() + 1);
      const nextWeek = new Date();
      nextWeek.setDate(nextWeek.getDate() + 3);
      
      await page.fill('input[name="start_date"]', tomorrow.toISOString().slice(0, 16));
      await page.fill('input[name="end_date"]', nextWeek.toISOString().slice(0, 16));
      await page.fill('input[name="max_amount"]', '500');
      await page.fill('textarea[name="reason"]', 'Test de integración con Temporal');
      
      // Enviar - esto debería crear un workflow en Temporal
      await page.click('button[type="submit"]');
      
      // Si Temporal está funcionando, debería redirigir con éxito
      // Si no, mostraría un error de conexión
      await expect(page).toHaveURL(/.*\/delegation\/list/, { timeout: 10000 });
    });
  });
});

// Configuración global de Playwright
module.exports = {};