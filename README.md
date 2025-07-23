# Sistema de Aprobación de Compras Amazon

Sistema de workflow para gestionar solicitudes de compra en Amazon corporativo usando Temporal.io.

## Características

- 🛒 **Validación automática** de productos de Amazon
- 📋 **Flujo de aprobación** configurable por monto
- 🔄 **Workflow robusto** con reintentos y timeouts
- 🌐 **Interface web** simple para empleados y responsables
- 📧 **Notificaciones** automáticas (simuladas)
- 🛡️ **Productos prohibidos** y detección de duplicados

## Arquitectura

### Componentes
- **Temporal Workflow**: Orquesta el proceso completo
- **Activities**: Validación, notificaciones, compra
- **Web Interface**: Frontend para empleados y responsables
- **Docker**: Temporal server local para desarrollo

### Flujo del Proceso
1. Empleado ingresa URLs de productos Amazon
2. Sistema valida productos y precios
3. Se envía a responsables para aprobación
4. Responsables pueden modificar/aprobar/rechazar
5. Si se aprueba, se ejecuta compra automática

## Desarrollo Rápido

### Prerrequisitos
- Go 1.21+
- Docker & Docker Compose
- Make (opcional)

### Inicio Rápido

```bash
# 1. Iniciar Temporal server
make temporal-up

# 2. En otra terminal, iniciar worker
make worker

# 3. En otra terminal, iniciar web server
make web

# Acceder a:
# - App: http://localhost:8081
# - Temporal UI: http://localhost:8080
```

### Comandos de Desarrollo

```bash
make help          # Ver todos los comandos disponibles
make deps          # Instalar dependencias
make temporal-up   # Iniciar Temporal server
make worker        # Ejecutar worker
make web           # Ejecutar servidor web
make test          # Ejecutar tests
make temporal-down # Parar Temporal server
```

## Testing del Sistema

### 1. Crear Solicitud
1. Ir a http://localhost:8081
2. Completar formulario con:
   - Email del empleado
   - URLs de productos Amazon (hay ejemplos precargados)
   - Justificación
   - Oficina de entrega
3. Enviar solicitud

### 2. Monitorear Workflow
- Ver estado en tiempo real: página de estado se auto-refresca
- Temporal UI: http://localhost:8080 para ver detalles técnicos

### 3. Aprobar/Rechazar
1. Buscar en logs del worker la URL de aprobación
2. O ir directamente a: http://localhost:8081/approval/[REQUEST_ID]
3. Tomar decisión de aprobación

### URLs de Ejemplo para Testing
```
https://amazon.es/dp/B08N5WRWNW  # Echo Dot (válido)
https://amazon.es/dp/B07XJ8C8F5  # Fire TV Stick (válido)
https://amazon.es/dp/PROHIBITED1 # Producto prohibido
```

## Estructura del Proyecto

```
.
├── cmd/
│   ├── worker/         # Temporal worker
│   └── web/           # Servidor web
├── internal/
│   ├── workflows/     # Definiciones de workflows
│   ├── activities/    # Activities (Amazon, aprobaciones)
│   └── models/        # Estructuras de datos
├── docs/             # Documentación
├── docker-compose.yml # Temporal server local
└── Makefile          # Comandos de desarrollo
```

## Configuración

### Variables de Entorno
- `TEMPORAL_HOST`: Host de Temporal (default: localhost:7233)
- `WEB_PORT`: Puerto del servidor web (default: 8081)

### Lógica de Aprobación
- **> €500**: Requiere CEO + Manager + Supervisor
- **> €100**: Requiere Manager + Supervisor  
- **Otros**: Solo requiere Supervisor

### Productos Prohibidos
Lista configurable en `activities/amazon.go`:
- Armas, alcohol, tabaco
- Contenido adulto
- IDs específicos en lista negra

## Logs y Monitoreo

### Worker Logs
- Validaciones de productos
- Notificaciones enviadas
- Estados de workflow

### Temporal UI
- http://localhost:8080
- Estado de workflows
- Historia de eventos
- Métricas de performance

## Próximos Pasos

### Funcionalidades Pendientes
- [ ] Base de datos persistente
- [ ] Integración real con Amazon API
- [ ] Sistema de notificaciones por email/Slack
- [ ] Dashboard de administración
- [ ] Tests de integración completos

### Mejoras Técnicas
- [ ] Autenticación y autorización
- [ ] Límites de rate para APIs
- [ ] Métricas y alertas
- [ ] Deployment en producción

## Soporte

Para problemas o preguntas:
1. Revisar logs del worker y web server
2. Verificar estado en Temporal UI
3. Consultar documentación en `/docs`

---

**Desarrollado con Temporal.io para máxima robustez y observabilidad**