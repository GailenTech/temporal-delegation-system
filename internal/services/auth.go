package services

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"temporal-workflow/internal/models"
)

// AuthService maneja la autenticación y sesiones
type AuthService struct {
	// En producción aquí iría redis/database para sesiones
	sessions map[string]*models.UserSession
}

// NewAuthService crea una nueva instancia del servicio de auth
func NewAuthService() *AuthService {
	return &AuthService{
		sessions: make(map[string]*models.UserSession),
	}
}

// Simulate login - en producción sería OAuth/SAML
func (s *AuthService) SimulateLogin(userID string) (*models.UserSession, error) {
	user, exists := models.GetUser(userID)
	if !exists {
		return nil, fmt.Errorf("user not found: %s", userID)
	}

	session := &models.UserSession{
		User:        *user,
		Permissions: user.GetPermissions(),
		LoginTime:   time.Now(),
		LastActive:  time.Now(),
	}

	// En producción sería un JWT o session token seguro
	sessionToken := fmt.Sprintf("session-%s-%d", userID, time.Now().Unix())
	s.sessions[sessionToken] = session

	return session, nil
}

// GetSession obtiene la sesión del usuario desde el token
func (s *AuthService) GetSession(sessionToken string) (*models.UserSession, bool) {
	session, exists := s.sessions[sessionToken]
	if !exists {
		return nil, false
	}

	// Actualizar última actividad
	session.LastActive = time.Now()
	return session, true
}

// GetUserFromRequest extrae el usuario de la request HTTP
func (s *AuthService) GetUserFromRequest(r *http.Request) (*models.User, error) {
	// Simular autenticación - en producción sería JWT/Cookie/Session
	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		// Fallback: cookie de sesión simulada
		cookie, err := r.Cookie("user_session")
		if err != nil {
			return nil, fmt.Errorf("no authentication found")
		}
		
		session, exists := s.GetSession(cookie.Value)
		if !exists {
			return nil, fmt.Errorf("invalid session")
		}
		return &session.User, nil
	}

	user, exists := models.GetUser(userID)
	if !exists {
		return nil, fmt.Errorf("user not found: %s", userID)
	}

	return user, nil
}

// RequireAuth middleware que requiere autenticación
func (s *AuthService) RequireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, err := s.GetUserFromRequest(r)
		if err != nil {
			// Redirect to login page
			s.redirectToLogin(w, r)
			return
		}

		// Añadir usuario al contexto
		ctx := context.WithValue(r.Context(), "user", user)
		next.ServeHTTP(w, r.WithContext(ctx))
	}
}

// RequireRole middleware que requiere un rol específico
func (s *AuthService) RequireRole(role models.UserRole, next http.HandlerFunc) http.HandlerFunc {
	return s.RequireAuth(func(w http.ResponseWriter, r *http.Request) {
		user := r.Context().Value("user").(*models.User)
		if user.Role != role {
			http.Error(w, "Insufficient permissions", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// RequirePermission middleware que verifica permisos específicos
func (s *AuthService) RequirePermission(checkPermission func(models.Permissions) bool, next http.HandlerFunc) http.HandlerFunc {
	return s.RequireAuth(func(w http.ResponseWriter, r *http.Request) {
		user := r.Context().Value("user").(*models.User)
		perms := user.GetPermissions()
		
		if !checkPermission(perms) {
			http.Error(w, "Insufficient permissions", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// GetCurrentUser obtiene el usuario actual del contexto
func GetCurrentUser(r *http.Request) *models.User {
	user, ok := r.Context().Value("user").(*models.User)
	if !ok {
		return nil
	}
	return user
}

func (s *AuthService) redirectToLogin(w http.ResponseWriter, r *http.Request) {
	// Página de login simulada para desarrollo
	html := `
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>Login - Sistema de Compras</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 40px; background: #f5f5f5; }
        .login-container { max-width: 400px; margin: 100px auto; background: white; padding: 30px; border-radius: 5px; box-shadow: 0 2px 10px rgba(0,0,0,0.1); }
        .login-option { display: block; width: 100%; padding: 15px; margin: 10px 0; background: #007cba; color: white; text-decoration: none; text-align: center; border-radius: 3px; }
        .login-option:hover { background: #005a87; }
        h2 { text-align: center; color: #333; }
        .role-description { font-size: 0.9em; color: #666; margin-top: 5px; }
    </style>
</head>
<body>
    <div class="login-container">
        <h2>🔐 Sistema de Compras - Login</h2>
        <p style="text-align: center; color: #666;">Selecciona tu rol para entrar al sistema:</p>
        
        <a href="/login-as/empleado@empresa.com" class="login-option">
            👤 Juan Empleado
            <div class="role-description">Empleado IT - Solo puede solicitar</div>
        </a>
        
        <a href="/login-as/manager@empresa.com" class="login-option">
            👔 Ana Manager  
            <div class="role-description">Manager IT - Puede aprobar hasta €2,000</div>
        </a>
        
        <a href="/login-as/ceo@empresa.com" class="login-option">
            🎖️ Carlos CEO
            <div class="role-description">CEO - Puede aprobar sin límites</div>
        </a>
        
        <a href="/login-as/admin@empresa.com" class="login-option">
            ⚙️ Sofia Admin
            <div class="role-description">Admin Sistema - Acceso completo</div>
        </a>
        
        <p style="text-align: center; font-size: 0.8em; color: #999; margin-top: 30px;">
            En producción esto sería OAuth/SAML con Azure AD o Google Workspace
        </p>
    </div>
</body>
</html>`
	
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(html))
}

// HandleLogin maneja el login simulado
func (s *AuthService) HandleLogin(w http.ResponseWriter, r *http.Request) {
	// Si la URL no contiene "/login-as/", mostrar página de login
	if !strings.HasPrefix(r.URL.Path, "/login-as/") || len(r.URL.Path) <= len("/login-as/") {
		s.redirectToLogin(w, r)
		return
	}

	// Extraer userID de la URL: /login-as/empleado@empresa.com
	userID := r.URL.Path[len("/login-as/"):]
	
	session, err := s.SimulateLogin(userID)
	if err != nil {
		http.Error(w, "Login failed: "+err.Error(), http.StatusBadRequest)
		return
	}

	// En producción sería un JWT seguro
	sessionToken := fmt.Sprintf("session-%s-%d", userID, time.Now().Unix())
	s.sessions[sessionToken] = session

	// Setear cookie de sesión
	cookie := &http.Cookie{
		Name:     "user_session",
		Value:    sessionToken,
		Path:     "/",
		HttpOnly: true,
		MaxAge:   3600 * 8, // 8 horas
	}
	http.SetCookie(w, cookie)

	// Redirect al dashboard
	http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
}

// HandleLogout maneja el logout
func (s *AuthService) HandleLogout(w http.ResponseWriter, r *http.Request) {
	// Limpiar cookie
	cookie := &http.Cookie{
		Name:     "user_session",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		MaxAge:   -1,
	}
	http.SetCookie(w, cookie)

	// Redirect a login
	http.Redirect(w, r, "/", http.StatusSeeOther)
}