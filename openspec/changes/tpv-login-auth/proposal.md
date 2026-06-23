# Propuesta: Autenticación y login en TPV

## Intención

El TPV de Ferrowin inicia directamente sin ninguna autenticación. Cualquier persona
con acceso físico al terminal puede operar la caja sin identificarse. Esto es un riesgo
operativo y de control interno. Se necesita una pantalla de login que obligue al
operador a autenticarse antes de usar el POS.

## Alcance

### Incluido
- Endpoint `POST /api/v1/auth/login` en backend Go: valida username/password, retorna JWT
- Generación de JWT con `user_id` y `role_ids` en claims
- Pantalla de login en React (formulario username + password)
- Comando Tauri que llama al endpoint y almacena el token
- AuthContext en React que protege las rutas del POS
- Cierre de sesión (logout)
- Persistencia del token (localStorage) para mantener sesión entre reinicios
- Redirección automática al login si el token expiró o no existe

### Excluido
- Login offline (caching de credenciales en SQLite local)
- UI basada en roles (ocultar funciones según permiso)
- CRUD de usuarios (gestión de usuarios)
- Flujo de recuperación de contraseña

## Capacidades

### Nuevas Capacidades
- `tpv-login-auth`: Autenticación de operadores en el TPV mediante login con
  username/password, emisión y validación de JWT, y protección de rutas del POS.

### Capacidades Modificadas
- Ninguna. La capacidad `user-auth-rbac` existente cubre autorización (authZ)
  y no cambia su comportamiento con esta propuesta.

## Enfoque Técnico

Autenticación stateless via JWT. El backend Go verifica la contraseña con bcrypt,
emite un JWT firmado con los datos del usuario y sus roles. El TPV almacena el
token en localStorage, lo envía en cada request vía header `Authorization: Bearer`.
El middleware de seguridad existente en Go (`HasPermission`) ya valida tokens JWT
(implementado en cambios anteriores). React usa un AuthContext que checkea la
validez del token al cargar la app y redirige a `LoginScreen` si es necesario.

## Áreas Afectadas

| Área | Impacto | Descripción |
|------|---------|-------------|
| `internal/api/routes.go` | Modificado | Nueva ruta `POST /api/v1/auth/login` (pública) |
| `internal/api/handlers/auth/` | Nuevo | Handler de login con validación de credenciales |
| `internal/security/jwt.go` | Modificado | Ajustes si es necesario para login flow |
| `tpv-client/src/App.tsx` | Modificado | AuthContext wrapping toda la app |
| `tpv-client/src/screens/LoginScreen.tsx` | Nuevo | Formulario de login |
| `tpv-client/src/context/AuthContext.tsx` | Nuevo | Contexto de autenticación |
| `tpv-client/src-tauri/src/commands/auth.rs` | Nuevo | Comando Tauri para login |

## Riesgos

| Riesgo | Probabilidad | Mitigación |
|--------|-------------|------------|
| Secreto JWT hardcodeado en código | Media | Usar variable de entorno `JWT_SECRET`, fallback a archivo de configuración |
| Token expirado durante uso activo | Baja | Refresh implícito no está en alcance; se redirige a login al expirar |
| Sin login offline bloquea TPV sin backend | Alta | Aceptado para este cambio; offline login se documenta como trabajo futuro |

## Plan de Rollback

1. Revertir commit del endpoint de login
2. Revertir commit del comando Tauri y componente LoginScreen
3. TPV vuelve a iniciar sin autenticación

## Dependencias

- `golang.org/x/crypto/bcrypt` — verificación de hash de contraseña
- `github.com/golang-jwt/jwt/v5` — creación y validación de JWT
- API `POST /api/v1/auth/login` disponible en backend

## Criterios de Éxito

- [ ] Operador ingresa credenciales válidas y accede al POS
- [ ] Operador ingresa credenciales inválidas y ve mensaje de error
- [ ] Token expirado redirige a pantalla de login
- [ ] Sesión persiste al reiniciar el TPV
- [ ] Cerrar sesión elimina el token y vuelve al login
