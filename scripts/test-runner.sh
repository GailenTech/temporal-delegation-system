#!/bin/bash

# Test Runner para Sistema de Delegaciones
# Ejecuta validación completa del sistema

set -e

echo "🚀 Iniciando Test Runner del Sistema de Delegaciones"
echo "=================================================="

# Colores para output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Función para logging
log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Función para verificar si un servicio está corriendo
check_service() {
    local service_name=$1
    local port=$2
    local max_attempts=${3:-30}
    local attempt=1

    log_info "Verificando $service_name en puerto $port..."
    
    while [ $attempt -le $max_attempts ]; do
        # Para puerto 7233 (gRPC), verificar solo si está escuchando
        if [ "$port" = "7233" ]; then
            if nc -z localhost $port 2>/dev/null; then
                log_info "$service_name está funcionando ✅"
                return 0
            fi
        else
            # Para puertos HTTP, verificar con curl
            if curl -s -o /dev/null -w "%{http_code}" http://localhost:$port | grep -q "200\|302"; then
                log_info "$service_name está funcionando ✅"
                return 0
            fi
        fi
        
        echo -n "."
        sleep 1
        ((attempt++))
    done
    
    log_error "$service_name no está respondiendo en puerto $port ❌"
    return 1
}

# Función para verificar prerequisitos
check_prerequisites() {
    log_info "Verificando prerequisitos..."
    
    # Verificar Go
    if ! command -v go &> /dev/null; then
        log_error "Go no está instalado"
        exit 1
    fi
    log_info "Go: $(go version)"
    
    # Verificar Docker
    if ! command -v docker &> /dev/null; then
        log_error "Docker no está instalado"
        exit 1
    fi
    log_info "Docker: $(docker --version)"
    
    # Verificar Node.js para Playwright
    if ! command -v node &> /dev/null; then
        log_error "Node.js no está instalado"
        exit 1
    fi
    log_info "Node.js: $(node --version)"
    
    # Verificar que los binarios están compilados
    if [ ! -f "webserver" ]; then
        log_info "Compilando webserver..."
        go build -o webserver ./cmd/web
        if [ $? -ne 0 ]; then
            log_error "Error compilando webserver"
            exit 1
        fi
    fi
    
    if [ ! -f "worker" ]; then
        log_info "Compilando worker..."
        go build -o worker ./cmd/worker
        if [ $? -ne 0 ]; then
            log_error "Error compilando worker"
            exit 1
        fi
    fi
    
    log_info "Prerequisitos verificados ✅"
}

# Función para iniciar servicios
start_services() {
    log_info "Iniciando servicios necesarios..."
    
    # Iniciar Temporal con Docker
    log_info "Iniciando Temporal Server..."
    docker compose up -d
    if [ $? -ne 0 ]; then
        log_error "Error iniciando Temporal con Docker"
        exit 1
    fi
    
    # Esperar a que Temporal esté listo
    check_service "Temporal Server" 7233 60
    check_service "Temporal UI" 8082 30
    
    # Iniciar Worker
    log_info "Iniciando Temporal Worker..."
    ./worker > worker-test.log 2>&1 &
    WORKER_PID=$!
    echo $WORKER_PID > worker.pid
    sleep 3
    
    # Verificar que el worker está corriendo
    if ! kill -0 $WORKER_PID 2>/dev/null; then
        log_error "Worker no se inició correctamente"
        cat worker-test.log
        exit 1
    fi
    
    # Iniciar Web Server
    log_info "Iniciando Web Server..."
    ./webserver > web-test.log 2>&1 &
    WEB_PID=$!
    echo $WEB_PID > web.pid
    
    # Verificar que el web server está respondiendo
    check_service "Web Server" 8081 30
    
    log_info "Todos los servicios iniciados ✅"
}

# Función para instalar dependencias de Playwright
setup_playwright() {
    log_info "Configurando Playwright..."
    
    if [ ! -d "node_modules" ]; then
        log_info "Instalando dependencias de Node.js..."
        npm install
    fi
    
    log_info "Instalando navegadores de Playwright..."
    npx playwright install
    npx playwright install-deps
    
    log_info "Playwright configurado ✅"
}

# Función para ejecutar tests
run_tests() {
    log_info "Ejecutando tests E2E con Playwright..."
    
    # Crear directorio de resultados
    mkdir -p test-results
    
    # Ejecutar tests con diferentes configuraciones
    local test_exit_code=0
    
    # Test básico en Chrome
    log_info "Ejecutando tests en Chrome..."
    npx playwright test --project=chromium --reporter=list
    if [ $? -ne 0 ]; then
        test_exit_code=1
        log_warn "Algunos tests fallaron en Chrome"
    fi
    
    # Si tenemos tiempo, ejecutar en otros navegadores
    if [ "${QUICK_TEST:-false}" != "true" ]; then
        log_info "Ejecutando tests en Firefox..."
        npx playwright test --project=firefox --reporter=list
        if [ $? -ne 0 ]; then
            test_exit_code=1
            log_warn "Algunos tests fallaron en Firefox"
        fi
    fi
    
    # Generar reporte
    log_info "Generando reporte HTML..."
    npx playwright show-report --host=127.0.0.1 > /dev/null 2>&1 &
    
    if [ $test_exit_code -eq 0 ]; then
        log_info "Todos los tests pasaron ✅"
    else
        log_warn "Algunos tests fallaron ⚠️"
    fi
    
    return $test_exit_code
}

# Función para limpiar recursos
cleanup() {
    log_info "Limpiando recursos..."
    
    # Terminar procesos
    if [ -f "web.pid" ]; then
        WEB_PID=$(cat web.pid)
        if kill -0 $WEB_PID 2>/dev/null; then
            kill $WEB_PID
            log_info "Web server terminado"
        fi
        rm -f web.pid
    fi
    
    if [ -f "worker.pid" ]; then
        WORKER_PID=$(cat worker.pid)
        if kill -0 $WORKER_PID 2>/dev/null; then
            kill $WORKER_PID
            log_info "Worker terminado"
        fi
        rm -f worker.pid
    fi
    
    # Limpiar logs de test
    rm -f web-test.log worker-test.log
    
    # Opcional: detener Docker (comentado para no interferir con desarrollo)
    # docker compose down
    
    log_info "Limpieza completada"
}

# Función principal
main() {
    local skip_setup=${1:-false}
    
    # Configurar trap para limpieza
    trap cleanup EXIT
    
    if [ "$skip_setup" != "true" ]; then
        check_prerequisites
        start_services
        setup_playwright
    else
        log_info "Saltando setup - usando servicios existentes"
        check_service "Web Server" 8081 5
        check_service "Temporal Server" 7233 5
    fi
    
    # Ejecutar tests
    run_tests
    local test_result=$?
    
    # Mostrar resumen
    echo ""
    echo "=================================================="
    if [ $test_result -eq 0 ]; then
        log_info "🎉 TODOS LOS TESTS PASARON EXITOSAMENTE"
        log_info "Reporte disponible en: playwright-report/index.html"
        log_info "Temporal UI disponible en: http://localhost:8082"
    else
        log_warn "⚠️  ALGUNOS TESTS FALLARON"
        log_info "Revisa los logs y el reporte HTML para más detalles"
    fi
    echo "=================================================="
    
    return $test_result
}

# Verificar argumentos
if [ "$1" = "--help" ] || [ "$1" = "-h" ]; then
    echo "Test Runner para Sistema de Delegaciones"
    echo ""
    echo "Uso:"
    echo "  $0                    # Ejecutar setup completo y tests"
    echo "  $0 --skip-setup       # Usar servicios existentes"
    echo "  $0 --help            # Mostrar esta ayuda"
    echo ""
    echo "Variables de entorno:"
    echo "  QUICK_TEST=true       # Solo ejecutar tests en Chrome"
    echo ""
    exit 0
fi

# Ejecutar
if [ "$1" = "--skip-setup" ]; then
    main true
else
    main false
fi