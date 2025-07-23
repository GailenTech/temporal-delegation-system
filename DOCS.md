# 📚 Documentación del Proyecto

## 🎯 Documento Principal

- **[MANUAL_TEMPORAL.md](docs/MANUAL_TEMPORAL.md)** - 📖 **Manual Completo y Tutorial de Temporal.io** 
  - Guía comprehensiva que explica todos los conceptos de Temporal.io
  - Incluye código detallado de workflows y activities
  - Screenshots del Temporal UI para monitoreo
  - Casos de uso avanzados y mejores prácticas
  - Tutorial paso a paso usando nuestro sistema como ejemplo

## 📋 Documentos Complementarios

- [PLAN.md](docs/PLAN.md) - Plan de desarrollo completo y arquitectura del sistema
- [DIARY.md](docs/DIARY.md) - Diario de desarrollo con progreso diario y decisiones técnicas
- [AUTHENTICATION_DESIGN.md](docs/AUTHENTICATION_DESIGN.md) - Diseño del sistema de autenticación y autorización

## 🤖 Arquitecturas Avanzadas de Temporal.io

- **[ADVANCED_WORKFLOW_VERSIONING.md](docs/ADVANCED_WORKFLOW_VERSIONING.md)** - 📈 **Estrategias Avanzadas de Versionado de Workflows**
  - Caso de estudio completo: Sistema de Revisión Automatizada
  - Implementación con GetVersion() API para compatibility
  - Worker Versioning con Build IDs para zero-downtime deployments
  - Feature flags dinámicas y control runtime
  - Testing multi-versión y replay compatibility
  - Monitoreo y observabilidad avanzada
  - Scripts de deployment selectivo y rollback automático

- **[DYNAMIC_WORKFLOW_DEPLOYMENT.md](docs/DYNAMIC_WORKFLOW_DEPLOYMENT.md)** - 🚀 **Despliegue Dinámico y Generación Runtime**
  - Deployment sin reinicio del sistema con control granular de usuarios
  - Routing selectivo por departamento/usuario/porcentaje
  - API REST para control dinámico en tiempo real
  - Generación dinámica de workflows via agentes IA
  - Template-based generation y compilación runtime
  - LLM integration para chatbot de creación de workflows
  - Análisis de viabilidad técnica y limitaciones de producción

- **[AI_AGENT_PIPELINE.md](docs/AI_AGENT_PIPELINE.md)** - 🧠 **Pipeline de Agentes IA para Desarrollo Autónomo**
  - Sistema multi-agente para desarrollo completamente autónomo
  - Agente de Especificación: Convierte lenguaje natural en specs técnicas
  - Agente de Código: Implementa especificaciones en código Go compilable
  - Agente de Testing: Validación automática con tests unitarios y smoke tests
  - Agente de Deployment: Blue/green deployment con monitoreo automático
  - Agente de QA: Validación funcional y análisis de calidad con IA
  - Pipeline orchestrator y API REST completa

## 🔬 Investigación Empresarial

- **[ENTERPRISE_AUTHORIZATION_RESEARCH.md](docs/ENTERPRISE_AUTHORIZATION_RESEARCH.md)** - 📊 **Investigación Completa de Sistemas de Autorización Empresarial**
  - Análisis comparativo de modelos de autorización (RBAC, ABAC, PBAC)
  - Evaluación de soluciones empresariales (Auth0, Okta, AWS Cognito, Keycloak, OPA)
  - Estándares modernos (OAuth 2.1, SCIM 2.0, JWT mejores prácticas)
  - Arquitectura recomendada para escalar el sistema actual
  - Estrategia de migración detallada con análisis costo-beneficio
  - Ejemplos de código e integración con Temporal.io

- **[MICROFRONTEND_ENTERPRISE_PORTAL_ARCHITECTURE.md](docs/MICROFRONTEND_ENTERPRISE_PORTAL_ARCHITECTURE.md)** - 🏗️ **Arquitectura de Portales Empresariales y Microfrontends**
  - Evolución de arquitecturas de portal modernas (Backstage, Module Federation)
  - Análisis de tecnologías microfrontend (Vite, Single-SPA, Qiankun)
  - Integración con motores de workflow como Temporal.io
  - Roadmap de evolución desde HTML simple a microfrontends
  - Casos de uso empresariales y métricas de rendimiento
  - Estimaciones de costo y esfuerzo de implementación

## ☁️ Deployment y Infraestructura

- **[DEPLOYMENT_ARCHITECTURE.md](docs/DEPLOYMENT_ARCHITECTURE.md)** - 🚀 **Arquitectura de Deployment en Google Cloud**
  - Arquitectura híbrida Cloud Run + GKE para demos y producción
  - Análisis comparativo de opciones de deployment (GKE vs Cloud Run vs híbrido)
  - Scripts de automatización y herramientas de deployment
  - Estimaciones de costo detalladas por entorno (demo/staging/producción)
  - Terraform modules y Helm charts listos para usar
  - CI/CD pipeline y estrategias de seguridad empresarial

## 🗂️ Estructura de Documentación

La documentación se mantiene en el directorio `/docs` y está organizada de la siguiente manera:

### 📖 Documentos Principales
- **MANUAL_TEMPORAL.md**: Manual completo con tutorial de Temporal.io ⭐ **DOCUMENTO PRINCIPAL**
- **PLAN.md**: Contiene la arquitectura, plan de implementación por fases y consideraciones técnicas
- **DIARY.md**: Registro cronológico de avances, decisiones y obstáculos encontrados

### 🤖 Arquitecturas Avanzadas
- **ADVANCED_WORKFLOW_VERSIONING.md**: Estrategias de versionado y deployment selectivo
- **DYNAMIC_WORKFLOW_DEPLOYMENT.md**: Deployment dinámico y generación runtime con IA
- **AI_AGENT_PIPELINE.md**: Pipeline de agentes IA para desarrollo autónomo

### 🔬 Investigación Empresarial
- **ENTERPRISE_AUTHORIZATION_RESEARCH.md**: Análisis completo de sistemas de autorización escalables
- **MICROFRONTEND_ENTERPRISE_PORTAL_ARCHITECTURE.md**: Arquitecturas modernas de portales empresariales

### ☁️ Infraestructura y Deployment
- **DEPLOYMENT_ARCHITECTURE.md**: Arquitectura completa para Google Cloud con scripts y automation

### 🛠️ Herramientas y Scripts
- **scripts/deploy-demo.sh**: Script de deployment automático para demos
- **scripts/cost-calculator.py**: Calculadora interactiva de costos GCP
- **terraform/**: Modules de infraestructura como código
- **helm/**: Charts personalizados para Kubernetes

## Actualizaciones

La documentación se actualiza constantemente durante el desarrollo siguiendo las normas establecidas en CLAUDE.md.