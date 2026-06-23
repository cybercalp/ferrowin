package adapters

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"ferrowin/internal/security/domain"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// ---------------------------------------------------------------------------
// Repository interfaces (subset required by the controller)
// ---------------------------------------------------------------------------

type rbacUserRepository interface {
	GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error)
	List(ctx context.Context) ([]domain.User, error)
	Save(ctx context.Context, user *domain.User) error
	Delete(ctx context.Context, id uuid.UUID) error
}

type rbacGroupRepository interface {
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Group, error)
	List(ctx context.Context) ([]domain.Group, error)
	Save(ctx context.Context, group *domain.Group) error
	Delete(ctx context.Context, id uuid.UUID) error
	AssignGroupToUser(ctx context.Context, userID uuid.UUID, groupID uuid.UUID) error
	RemoveGroupFromUser(ctx context.Context, userID uuid.UUID, groupID uuid.UUID) error
}

type rbacRoleSetRepository interface {
	GetByID(ctx context.Context, id uuid.UUID) (*domain.RoleSet, error)
	List(ctx context.Context) ([]domain.RoleSet, error)
	Save(ctx context.Context, roleSet *domain.RoleSet) error
	Delete(ctx context.Context, id uuid.UUID) error
	AssignRoleSetToGroup(ctx context.Context, groupID uuid.UUID, roleSetID uuid.UUID) error
	RemoveRoleSetFromGroup(ctx context.Context, groupID uuid.UUID, roleSetID uuid.UUID) error
}

type rbacRoleRepository interface {
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Role, error)
	List(ctx context.Context) ([]domain.Role, error)
	Save(ctx context.Context, role *domain.Role) error
	Delete(ctx context.Context, id uuid.UUID) error
	AssignRoleToRoleSet(ctx context.Context, roleSetID uuid.UUID, roleID uuid.UUID) error
	RemoveRoleFromRoleSet(ctx context.Context, roleSetID uuid.UUID, roleID uuid.UUID) error
}

// ---------------------------------------------------------------------------
// Controller
// ---------------------------------------------------------------------------

// RBACController handles HTTP CRUD endpoints for the RBAC system.
type RBACController struct {
	userRepo    rbacUserRepository
	groupRepo   rbacGroupRepository
	roleSetRepo rbacRoleSetRepository
	roleRepo    rbacRoleRepository
}

// NewRBACController creates a new RBACController.
func NewRBACController(
	userRepo rbacUserRepository,
	groupRepo rbacGroupRepository,
	roleSetRepo rbacRoleSetRepository,
	roleRepo rbacRoleRepository,
) *RBACController {
	return &RBACController{
		userRepo:    userRepo,
		groupRepo:   groupRepo,
		roleSetRepo: roleSetRepo,
		roleRepo:    roleRepo,
	}
}

// ---------------------------------------------------------------------------
// Request / response types
// ---------------------------------------------------------------------------

type listResponse struct {
	Data  interface{} `json:"data"`
	Total int         `json:"total"`
}

// User JSON types (never expose password_hash)

type userItemJSON struct {
	ID       string `json:"id"`
	Username string `json:"username"`
}

type userDetailJSON struct {
	ID       string           `json:"id"`
	Username string           `json:"username"`
	Groups   []groupDetailJSON `json:"groups,omitempty"`
}

type groupItemJSON struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type groupDetailJSON struct {
	ID       string             `json:"id"`
	Name     string             `json:"name"`
	RoleSets []roleSetDetailJSON `json:"role_sets,omitempty"`
}

type roleSetItemJSON struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type roleSetDetailJSON struct {
	ID    string         `json:"id"`
	Name  string         `json:"name"`
	Roles []roleItemJSON `json:"roles,omitempty"`
}

type roleItemJSON struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// Request bodies

type createUserRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type patchUserRequest struct {
	Username *string `json:"username"`
	Password *string `json:"password"`
}

type createGroupRequest struct {
	Name string `json:"name"`
}

type patchGroupRequest struct {
	Name *string `json:"name"`
}

type createRoleSetRequest struct {
	Name string `json:"name"`
}

type patchRoleSetRequest struct {
	Name *string `json:"name"`
}

type createRoleRequest struct {
	Name string `json:"name"`
}

type patchRoleRequest struct {
	Name *string `json:"name"`
}

type assignGroupRequest struct {
	GroupID string `json:"group_id"`
}

type assignRoleSetRequest struct {
	RoleSetID string `json:"role_set_id"`
}

type assignRoleRequest struct {
	RoleID string `json:"role_id"`
}

// ---------------------------------------------------------------------------
// HTTP dispatch
// ---------------------------------------------------------------------------

func (c *RBACController) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	path := r.URL.Path

	// --- Roles ---
	if path == "/api/v1/security/roles" {
		switch r.Method {
		case http.MethodGet:
			c.handleListRoles(w, r)
		case http.MethodPost:
			c.handleCreateRole(w, r)
		default:
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		}
		return
	}
	if strings.HasPrefix(path, "/api/v1/security/roles/") {
		id, err := parseUUIDFromPrefix(path, "/api/v1/security/roles/")
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid role ID")
			return
		}
		switch r.Method {
		case http.MethodGet:
			c.handleGetRole(w, r, id)
		case http.MethodPatch:
			c.handlePatchRole(w, r, id)
		case http.MethodDelete:
			c.handleDeleteRole(w, r, id)
		default:
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		}
		return
	}

	// --- RoleSets ---
	if path == "/api/v1/security/role-sets" {
		switch r.Method {
		case http.MethodGet:
			c.handleListRoleSets(w, r)
		case http.MethodPost:
			c.handleCreateRoleSet(w, r)
		default:
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		}
		return
	}
	if strings.HasPrefix(path, "/api/v1/security/role-sets/") {
		id, sub, subID, ok := parseNestedPath(path, "/api/v1/security/role-sets/")
		if !ok {
			writeError(w, http.StatusBadRequest, "invalid role set ID")
			return
		}
		if sub == "" {
			switch r.Method {
			case http.MethodGet:
				c.handleGetRoleSet(w, r, id)
			case http.MethodPatch:
				c.handlePatchRoleSet(w, r, id)
			case http.MethodDelete:
				c.handleDeleteRoleSet(w, r, id)
			default:
				writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			}
			return
		}
		if sub == "roles" && subID == uuid.Nil && r.Method == http.MethodPost {
			c.handleAssignRoleToRoleSet(w, r, id)
			return
		}
		if sub == "roles" && subID != uuid.Nil && r.Method == http.MethodDelete {
			c.handleRemoveRoleFromRoleSet(w, r, id, subID)
			return
		}
		http.NotFound(w, r)
		return
	}

	// --- Groups ---
	if path == "/api/v1/security/groups" {
		switch r.Method {
		case http.MethodGet:
			c.handleListGroups(w, r)
		case http.MethodPost:
			c.handleCreateGroup(w, r)
		default:
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		}
		return
	}
	if strings.HasPrefix(path, "/api/v1/security/groups/") {
		id, sub, subID, ok := parseNestedPath(path, "/api/v1/security/groups/")
		if !ok {
			writeError(w, http.StatusBadRequest, "invalid group ID")
			return
		}
		if sub == "" {
			switch r.Method {
			case http.MethodGet:
				c.handleGetGroup(w, r, id)
			case http.MethodPatch:
				c.handlePatchGroup(w, r, id)
			case http.MethodDelete:
				c.handleDeleteGroup(w, r, id)
			default:
				writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			}
			return
		}
		if sub == "role-sets" && subID == uuid.Nil && r.Method == http.MethodPost {
			c.handleAssignRoleSetToGroup(w, r, id)
			return
		}
		if sub == "role-sets" && subID != uuid.Nil && r.Method == http.MethodDelete {
			c.handleRemoveRoleSetFromGroup(w, r, id, subID)
			return
		}
		http.NotFound(w, r)
		return
	}

	// --- Users ---
	if path == "/api/v1/security/users" {
		switch r.Method {
		case http.MethodGet:
			c.handleListUsers(w, r)
		case http.MethodPost:
			c.handleCreateUser(w, r)
		default:
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		}
		return
	}
	if strings.HasPrefix(path, "/api/v1/security/users/") {
		id, sub, subID, ok := parseNestedPath(path, "/api/v1/security/users/")
		if !ok {
			writeError(w, http.StatusBadRequest, "invalid user ID")
			return
		}
		if sub == "" {
			switch r.Method {
			case http.MethodGet:
				c.handleGetUser(w, r, id)
			case http.MethodPatch:
				c.handlePatchUser(w, r, id)
			case http.MethodDelete:
				c.handleDeleteUser(w, r, id)
			default:
				writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			}
			return
		}
		if sub == "groups" && subID == uuid.Nil && r.Method == http.MethodPost {
			c.handleAssignGroupToUser(w, r, id)
			return
		}
		if sub == "groups" && subID != uuid.Nil && r.Method == http.MethodDelete {
			c.handleRemoveGroupFromUser(w, r, id, subID)
			return
		}
		http.NotFound(w, r)
		return
	}

	http.NotFound(w, r)
}

// ---------------------------------------------------------------------------
// Path parsing helpers
// ---------------------------------------------------------------------------

// parseUUIDFromPrefix extracts the first UUID segment from a path prefix.
// e.g. parseUUIDFromPrefix("/api/v1/security/roles/", "/api/v1/security/roles/abc123")
// returns abc123. It expects exactly one segment after the prefix.
func parseUUIDFromPrefix(path, prefix string) (uuid.UUID, error) {
	remaining := strings.TrimPrefix(path, prefix)
	parts := strings.Split(remaining, "/")
	if len(parts) == 0 || parts[0] == "" {
		return uuid.Nil, errors.New("missing UUID in path")
	}
	return uuid.Parse(parts[0])
}

// parseNestedPath extracts the main ID, optional sub-resource name, and optional
// sub-resource ID from a path like /prefix/{id}/sub/{subId}.
// Returns (id, subresource, subID, ok).
func parseNestedPath(path, prefix string) (uuid.UUID, string, uuid.UUID, bool) {
	remaining := strings.TrimPrefix(path, prefix)
	raw := strings.Split(remaining, "/")

	// Filter empty segments
	var parts []string
	for _, p := range raw {
		if p != "" {
			parts = append(parts, p)
		}
	}

	if len(parts) == 0 {
		return uuid.Nil, "", uuid.Nil, false
	}

	id, err := uuid.Parse(parts[0])
	if err != nil {
		return uuid.Nil, "", uuid.Nil, false
	}

	if len(parts) >= 2 {
		if len(parts) >= 3 {
			subID, err := uuid.Parse(parts[2])
			if err != nil {
				return uuid.Nil, "", uuid.Nil, false
			}
			return id, parts[1], subID, true
		}
		return id, parts[1], uuid.Nil, true
	}

	return id, "", uuid.Nil, true
}

// ---------------------------------------------------------------------------
// Helper: write a JSON error response
// ---------------------------------------------------------------------------

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

// ---------------------------------------------------------------------------
// Password hashing
// ---------------------------------------------------------------------------

func hashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

// ---------------------------------------------------------------------------
// Conversion helpers (domain -> JSON)
// ---------------------------------------------------------------------------

func toUserItem(u domain.User) userItemJSON {
	return userItemJSON{
		ID:       u.ID.String(),
		Username: u.Username,
	}
}

func toUserDetail(u *domain.User) userDetailJSON {
	d := userDetailJSON{
		ID:       u.ID.String(),
		Username: u.Username,
		Groups:   make([]groupDetailJSON, 0, len(u.Groups)),
	}
	for _, g := range u.Groups {
		d.Groups = append(d.Groups, toGroupDetail(g))
	}
	return d
}

func toGroupItem(g domain.Group) groupItemJSON {
	return groupItemJSON{
		ID:   g.ID.String(),
		Name: g.Name,
	}
}

func toGroupDetail(g domain.Group) groupDetailJSON {
	d := groupDetailJSON{
		ID:       g.ID.String(),
		Name:     g.Name,
		RoleSets: make([]roleSetDetailJSON, 0, len(g.RoleSets)),
	}
	for _, rs := range g.RoleSets {
		d.RoleSets = append(d.RoleSets, toRoleSetDetail(rs))
	}
	return d
}

func toRoleSetItem(rs domain.RoleSet) roleSetItemJSON {
	return roleSetItemJSON{
		ID:   rs.ID.String(),
		Name: rs.Name,
	}
}

func toRoleSetDetail(rs domain.RoleSet) roleSetDetailJSON {
	d := roleSetDetailJSON{
		ID:    rs.ID.String(),
		Name:  rs.Name,
		Roles: make([]roleItemJSON, 0, len(rs.Roles)),
	}
	for _, r := range rs.Roles {
		d.Roles = append(d.Roles, toRoleItem(r))
	}
	return d
}

func toRoleItem(r domain.Role) roleItemJSON {
	return roleItemJSON{
		ID:   r.ID.String(),
		Name: r.Name,
	}
}

// ---------------------------------------------------------------------------
// Users handlers
// ---------------------------------------------------------------------------

func (c *RBACController) handleListUsers(w http.ResponseWriter, r *http.Request) {
	users, err := c.userRepo.List(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	items := make([]userItemJSON, 0, len(users))
	for _, u := range users {
		items = append(items, toUserItem(u))
	}
	writeJSON(w, http.StatusOK, listResponse{Data: items, Total: len(items)})
}

func (c *RBACController) handleGetUser(w http.ResponseWriter, r *http.Request, id uuid.UUID) {
	user, err := c.userRepo.GetByID(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if user == nil {
		writeError(w, http.StatusNotFound, "user not found")
		return
	}
	writeJSON(w, http.StatusOK, toUserDetail(user))
}

func (c *RBACController) handleCreateUser(w http.ResponseWriter, r *http.Request) {
	var req createUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Username == "" || req.Password == "" {
		writeError(w, http.StatusBadRequest, "username and password are required")
		return
	}

	hash, err := hashPassword(req.Password)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to hash password")
		return
	}

	user := &domain.User{
		ID:           uuid.New(),
		Username:     req.Username,
		PasswordHash: hash,
	}

	if err := c.userRepo.Save(r.Context(), user); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, toUserItem(*user))
}

func (c *RBACController) handlePatchUser(w http.ResponseWriter, r *http.Request, id uuid.UUID) {
	current, err := c.userRepo.GetByID(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if current == nil {
		writeError(w, http.StatusNotFound, "user not found")
		return
	}

	var req patchUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Username != nil {
		current.Username = *req.Username
	}
	if req.Password != nil {
		hash, err := hashPassword(*req.Password)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to hash password")
			return
		}
		current.PasswordHash = hash
	}

	if err := c.userRepo.Save(r.Context(), current); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, toUserItem(*current))
}

func (c *RBACController) handleDeleteUser(w http.ResponseWriter, r *http.Request, id uuid.UUID) {
	if err := c.userRepo.Delete(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (c *RBACController) handleAssignGroupToUser(w http.ResponseWriter, r *http.Request, userID uuid.UUID) {
	var req assignGroupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	groupID, err := uuid.Parse(req.GroupID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid group_id format")
		return
	}

	if err := c.groupRepo.AssignGroupToUser(r.Context(), userID, groupID); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (c *RBACController) handleRemoveGroupFromUser(w http.ResponseWriter, r *http.Request, userID uuid.UUID, groupID uuid.UUID) {
	if err := c.groupRepo.RemoveGroupFromUser(r.Context(), userID, groupID); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ---------------------------------------------------------------------------
// Groups handlers
// ---------------------------------------------------------------------------

func (c *RBACController) handleListGroups(w http.ResponseWriter, r *http.Request) {
	groups, err := c.groupRepo.List(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	items := make([]groupItemJSON, 0, len(groups))
	for _, g := range groups {
		items = append(items, toGroupItem(g))
	}
	writeJSON(w, http.StatusOK, listResponse{Data: items, Total: len(items)})
}

func (c *RBACController) handleGetGroup(w http.ResponseWriter, r *http.Request, id uuid.UUID) {
	group, err := c.groupRepo.GetByID(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if group == nil {
		writeError(w, http.StatusNotFound, "group not found")
		return
	}
	writeJSON(w, http.StatusOK, toGroupDetail(*group))
}

func (c *RBACController) handleCreateGroup(w http.ResponseWriter, r *http.Request) {
	var req createGroupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}

	group := &domain.Group{
		ID:   uuid.New(),
		Name: req.Name,
	}

	if err := c.groupRepo.Save(r.Context(), group); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, toGroupItem(*group))
}

func (c *RBACController) handlePatchGroup(w http.ResponseWriter, r *http.Request, id uuid.UUID) {
	current, err := c.groupRepo.GetByID(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if current == nil {
		writeError(w, http.StatusNotFound, "group not found")
		return
	}

	var req patchGroupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Name != nil {
		current.Name = *req.Name
	}

	if err := c.groupRepo.Save(r.Context(), current); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, toGroupItem(*current))
}

func (c *RBACController) handleDeleteGroup(w http.ResponseWriter, r *http.Request, id uuid.UUID) {
	if err := c.groupRepo.Delete(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (c *RBACController) handleAssignRoleSetToGroup(w http.ResponseWriter, r *http.Request, groupID uuid.UUID) {
	var req assignRoleSetRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	roleSetID, err := uuid.Parse(req.RoleSetID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid role_set_id format")
		return
	}

	if err := c.roleSetRepo.AssignRoleSetToGroup(r.Context(), groupID, roleSetID); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (c *RBACController) handleRemoveRoleSetFromGroup(w http.ResponseWriter, r *http.Request, groupID uuid.UUID, roleSetID uuid.UUID) {
	if err := c.roleSetRepo.RemoveRoleSetFromGroup(r.Context(), groupID, roleSetID); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ---------------------------------------------------------------------------
// RoleSets handlers
// ---------------------------------------------------------------------------

func (c *RBACController) handleListRoleSets(w http.ResponseWriter, r *http.Request) {
	roleSets, err := c.roleSetRepo.List(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	items := make([]roleSetItemJSON, 0, len(roleSets))
	for _, rs := range roleSets {
		items = append(items, toRoleSetItem(rs))
	}
	writeJSON(w, http.StatusOK, listResponse{Data: items, Total: len(items)})
}

func (c *RBACController) handleGetRoleSet(w http.ResponseWriter, r *http.Request, id uuid.UUID) {
	roleSet, err := c.roleSetRepo.GetByID(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if roleSet == nil {
		writeError(w, http.StatusNotFound, "role set not found")
		return
	}
	writeJSON(w, http.StatusOK, toRoleSetDetail(*roleSet))
}

func (c *RBACController) handleCreateRoleSet(w http.ResponseWriter, r *http.Request) {
	var req createRoleSetRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}

	roleSet := &domain.RoleSet{
		ID:   uuid.New(),
		Name: req.Name,
	}

	if err := c.roleSetRepo.Save(r.Context(), roleSet); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, toRoleSetItem(*roleSet))
}

func (c *RBACController) handlePatchRoleSet(w http.ResponseWriter, r *http.Request, id uuid.UUID) {
	current, err := c.roleSetRepo.GetByID(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if current == nil {
		writeError(w, http.StatusNotFound, "role set not found")
		return
	}

	var req patchRoleSetRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Name != nil {
		current.Name = *req.Name
	}

	if err := c.roleSetRepo.Save(r.Context(), current); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, toRoleSetItem(*current))
}

func (c *RBACController) handleDeleteRoleSet(w http.ResponseWriter, r *http.Request, id uuid.UUID) {
	if err := c.roleSetRepo.Delete(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (c *RBACController) handleAssignRoleToRoleSet(w http.ResponseWriter, r *http.Request, roleSetID uuid.UUID) {
	var req assignRoleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	roleID, err := uuid.Parse(req.RoleID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid role_id format")
		return
	}

	if err := c.roleRepo.AssignRoleToRoleSet(r.Context(), roleSetID, roleID); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (c *RBACController) handleRemoveRoleFromRoleSet(w http.ResponseWriter, r *http.Request, roleSetID uuid.UUID, roleID uuid.UUID) {
	if err := c.roleRepo.RemoveRoleFromRoleSet(r.Context(), roleSetID, roleID); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ---------------------------------------------------------------------------
// Roles handlers
// ---------------------------------------------------------------------------

func (c *RBACController) handleListRoles(w http.ResponseWriter, r *http.Request) {
	roles, err := c.roleRepo.List(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	items := make([]roleItemJSON, 0, len(roles))
	for _, role := range roles {
		items = append(items, toRoleItem(role))
	}
	writeJSON(w, http.StatusOK, listResponse{Data: items, Total: len(items)})
}

func (c *RBACController) handleGetRole(w http.ResponseWriter, r *http.Request, id uuid.UUID) {
	role, err := c.roleRepo.GetByID(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if role == nil {
		writeError(w, http.StatusNotFound, "role not found")
		return
	}
	writeJSON(w, http.StatusOK, toRoleItem(*role))
}

func (c *RBACController) handleCreateRole(w http.ResponseWriter, r *http.Request) {
	var req createRoleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}

	role := &domain.Role{
		ID:   uuid.New(),
		Name: req.Name,
	}

	if err := c.roleRepo.Save(r.Context(), role); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, toRoleItem(*role))
}

func (c *RBACController) handlePatchRole(w http.ResponseWriter, r *http.Request, id uuid.UUID) {
	current, err := c.roleRepo.GetByID(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if current == nil {
		writeError(w, http.StatusNotFound, "role not found")
		return
	}

	var req patchRoleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Name != nil {
		current.Name = *req.Name
	}

	if err := c.roleRepo.Save(r.Context(), current); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, toRoleItem(*current))
}

func (c *RBACController) handleDeleteRole(w http.ResponseWriter, r *http.Request, id uuid.UUID) {
	if err := c.roleRepo.Delete(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
