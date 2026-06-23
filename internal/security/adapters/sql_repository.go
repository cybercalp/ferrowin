package adapters

import (
	"context"
	"database/sql"
	"ferrowin/internal/security/domain"

	"github.com/google/uuid"
)

type txKey struct{}

// WithTx returns a new context containing the transaction.
func WithTx(ctx context.Context, tx *sql.Tx) context.Context {
	return context.WithValue(ctx, txKey{}, tx)
}

type dbExecutor interface {
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row
}

func getExecutor(ctx context.Context, db *sql.DB) dbExecutor {
	if tx, ok := ctx.Value(txKey{}).(*sql.Tx); ok {
		return tx
	}
	return db
}

type tempRole struct {
	ID   uuid.UUID
	Name string
}

type tempRoleSet struct {
	ID    uuid.UUID
	Name  string
	Roles []*tempRole
}

type tempGroup struct {
	ID       uuid.UUID
	Name     string
	RoleSets []*tempRoleSet
}

// SQLUserRepository implements ports.UserRepository and domain.UserRepositoryRequired.
type SQLUserRepository struct {
	db       *sql.DB
	isSQLite bool
}

// NewSQLUserRepository creates a new SQLUserRepository.
func NewSQLUserRepository(db *sql.DB, isSQLite bool) *SQLUserRepository {
	return &SQLUserRepository{
		db:       db,
		isSQLite: isSQLite,
	}
}

// GetByID retrieves a user and their complete nested hierarchy in a single query.
func (r *SQLUserRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	var query string
	if r.isSQLite {
		query = `SELECT 
			u.id AS user_id, 
			u.username AS user_username, 
			u.password_hash AS user_password_hash,
			g.id AS group_id, 
			g.name AS group_name,
			rs.id AS role_set_id, 
			rs.name AS role_set_name,
			role.id AS role_id, 
			role.name AS role_name
		FROM users u
		LEFT JOIN user_groups ug ON u.id = ug.user_id
		LEFT JOIN groups g ON ug.group_id = g.id
		LEFT JOIN group_role_sets grs ON g.id = grs.group_id
		LEFT JOIN role_sets rs ON grs.role_set_id = rs.id
		LEFT JOIN role_set_roles rsr ON rs.id = rsr.role_set_id
		LEFT JOIN roles role ON rsr.role_id = role.id
		WHERE u.id = ?`
	} else {
		query = `SELECT 
			u.id AS user_id, 
			u.username AS user_username, 
			u.password_hash AS user_password_hash,
			g.id AS group_id, 
			g.name AS group_name,
			rs.id AS role_set_id, 
			rs.name AS role_set_name,
			role.id AS role_id, 
			role.name AS role_name
		FROM users u
		LEFT JOIN user_groups ug ON u.id = ug.user_id
		LEFT JOIN groups g ON ug.group_id = g.id
		LEFT JOIN group_role_sets grs ON g.id = grs.group_id
		LEFT JOIN role_sets rs ON grs.role_set_id = rs.id
		LEFT JOIN role_set_roles rsr ON rs.id = rsr.role_set_id
		LEFT JOIN roles role ON rsr.role_id = role.id
		WHERE u.id = $1`
	}

	rows, err := getExecutor(ctx, r.db).QueryContext(ctx, query, id.String())
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanUserHierarchy(rows)
}

// GetByUsername retrieves a user and their complete nested hierarchy by username in a single query.
func (r *SQLUserRepository) GetByUsername(ctx context.Context, username string) (*domain.User, error) {
	var query string
	if r.isSQLite {
		query = `SELECT 
			u.id AS user_id, 
			u.username AS user_username, 
			u.password_hash AS user_password_hash,
			g.id AS group_id, 
			g.name AS group_name,
			rs.id AS role_set_id, 
			rs.name AS role_set_name,
			role.id AS role_id, 
			role.name AS role_name
		FROM users u
		LEFT JOIN user_groups ug ON u.id = ug.user_id
		LEFT JOIN groups g ON ug.group_id = g.id
		LEFT JOIN group_role_sets grs ON g.id = grs.group_id
		LEFT JOIN role_sets rs ON grs.role_set_id = rs.id
		LEFT JOIN role_set_roles rsr ON rs.id = rsr.role_set_id
		LEFT JOIN roles role ON rsr.role_id = role.id
		WHERE u.username = ?`
	} else {
		query = `SELECT 
			u.id AS user_id, 
			u.username AS user_username, 
			u.password_hash AS user_password_hash,
			g.id AS group_id, 
			g.name AS group_name,
			rs.id AS role_set_id, 
			rs.name AS role_set_name,
			role.id AS role_id, 
			role.name AS role_name
		FROM users u
		LEFT JOIN user_groups ug ON u.id = ug.user_id
		LEFT JOIN groups g ON ug.group_id = g.id
		LEFT JOIN group_role_sets grs ON g.id = grs.group_id
		LEFT JOIN role_sets rs ON grs.role_set_id = rs.id
		LEFT JOIN role_set_roles rsr ON rs.id = rsr.role_set_id
		LEFT JOIN roles role ON rsr.role_id = role.id
		WHERE u.username = $1`
	}

	rows, err := getExecutor(ctx, r.db).QueryContext(ctx, query, username)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanUserHierarchy(rows)
}

// List returns all users without hierarchy, ordered by username.
func (r *SQLUserRepository) List(ctx context.Context) ([]domain.User, error) {
	query := "SELECT id, username, password_hash FROM users ORDER BY username"
	rows, err := getExecutor(ctx, r.db).QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []domain.User
	for rows.Next() {
		var u domain.User
		var idStr string
		if err := rows.Scan(&idStr, &u.Username, &u.PasswordHash); err != nil {
			return nil, err
		}
		u.ID, err = uuid.Parse(idStr)
		if err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return users, nil
}

// Delete removes a user by ID. Cascade handles junction table cleanup.
func (r *SQLUserRepository) Delete(ctx context.Context, id uuid.UUID) error {
	var query string
	if r.isSQLite {
		query = "DELETE FROM users WHERE id = ?"
	} else {
		query = "DELETE FROM users WHERE id = $1"
	}
	_, err := getExecutor(ctx, r.db).ExecContext(ctx, query, id.String())
	return err
}

// Save inserts or updates a user.
func (r *SQLUserRepository) Save(ctx context.Context, user *domain.User) error {
	var query string
	if r.isSQLite {
		query = `INSERT INTO users (id, username, password_hash) VALUES (?, ?, ?)
                 ON CONFLICT(id) DO UPDATE SET username = excluded.username, password_hash = excluded.password_hash`
	} else {
		query = `INSERT INTO users (id, username, password_hash) VALUES ($1, $2, $3)
                 ON CONFLICT(id) DO UPDATE SET username = EXCLUDED.username, password_hash = EXCLUDED.password_hash`
	}

	_, err := getExecutor(ctx, r.db).ExecContext(ctx, query, user.ID.String(), user.Username, user.PasswordHash)
	return err
}

func scanUserHierarchy(rows *sql.Rows) (*domain.User, error) {
	var userIDStr, username, passwordHash string
	var groups []*tempGroup
	groupsMap := make(map[uuid.UUID]*tempGroup)
	groupRoleSetsMap := make(map[uuid.UUID]map[uuid.UUID]*tempRoleSet)
	roleSetRolesMap := make(map[uuid.UUID]map[uuid.UUID]map[uuid.UUID]bool)

	hasRows := false
	for rows.Next() {
		hasRows = true
		var uID, uName, uPass string
		var gIDNull, gNameNull sql.NullString
		var rsIDNull, rsNameNull sql.NullString
		var rIDNull, rNameNull sql.NullString

		err := rows.Scan(
			&uID, &uName, &uPass,
			&gIDNull, &gNameNull,
			&rsIDNull, &rsNameNull,
			&rIDNull, &rNameNull,
		)
		if err != nil {
			return nil, err
		}

		if userIDStr == "" {
			userIDStr = uID
			username = uName
			passwordHash = uPass
		}

		if gIDNull.Valid && gIDNull.String != "" {
			gID, err := uuid.Parse(gIDNull.String)
			if err == nil {
				group, exists := groupsMap[gID]
				if !exists {
					group = &tempGroup{
						ID:   gID,
						Name: gNameNull.String,
					}
					groupsMap[gID] = group
					groups = append(groups, group)
					groupRoleSetsMap[gID] = make(map[uuid.UUID]*tempRoleSet)
					roleSetRolesMap[gID] = make(map[uuid.UUID]map[uuid.UUID]bool)
				}

				if rsIDNull.Valid && rsIDNull.String != "" {
					rsID, err := uuid.Parse(rsIDNull.String)
					if err == nil {
						roleSet, rsExists := groupRoleSetsMap[gID][rsID]
						if !rsExists {
							roleSet = &tempRoleSet{
								ID:   rsID,
								Name: rsNameNull.String,
							}
							groupRoleSetsMap[gID][rsID] = roleSet
							group.RoleSets = append(group.RoleSets, roleSet)
							roleSetRolesMap[gID][rsID] = make(map[uuid.UUID]bool)
						}

						if rIDNull.Valid && rIDNull.String != "" {
							rID, err := uuid.Parse(rIDNull.String)
							if err == nil {
								if !roleSetRolesMap[gID][rsID][rID] {
									roleSetRolesMap[gID][rsID][rID] = true
									roleSet.Roles = append(roleSet.Roles, &tempRole{
										ID:   rID,
										Name: rNameNull.String,
									})
								}
							}
						}
					}
				}
			}
		}
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	if !hasRows {
		return nil, nil
	}

	domainGroups := make([]domain.Group, len(groups))
	for i, tg := range groups {
		domainRoleSets := make([]domain.RoleSet, len(tg.RoleSets))
		for j, trs := range tg.RoleSets {
			domainRoles := make([]domain.Role, len(trs.Roles))
			for k, tr := range trs.Roles {
				domainRoles[k] = domain.Role{
					ID:   tr.ID,
					Name: tr.Name,
				}
			}
			domainRoleSets[j] = domain.RoleSet{
				ID:    trs.ID,
				Name:  trs.Name,
				Roles: domainRoles,
			}
		}
		domainGroups[i] = domain.Group{
			ID:       tg.ID,
			Name:     tg.Name,
			RoleSets: domainRoleSets,
		}
	}

	userUUID, err := uuid.Parse(userIDStr)
	if err != nil {
		return nil, err
	}

	return &domain.User{
		ID:           userUUID,
		Username:     username,
		PasswordHash: passwordHash,
		Groups:       domainGroups,
	}, nil
}

// SQLGroupRepository implements ports.GroupRepository.
type SQLGroupRepository struct {
	db       *sql.DB
	isSQLite bool
}

// NewSQLGroupRepository creates a new SQLGroupRepository.
func NewSQLGroupRepository(db *sql.DB, isSQLite bool) *SQLGroupRepository {
	return &SQLGroupRepository{
		db:       db,
		isSQLite: isSQLite,
	}
}

// GetByID retrieves a group and its nested role sets and roles in a single query.
func (r *SQLGroupRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Group, error) {
	var query string
	if r.isSQLite {
		query = `SELECT 
			g.id AS group_id, 
			g.name AS group_name,
			rs.id AS role_set_id, 
			rs.name AS role_set_name,
			role.id AS role_id, 
			role.name AS role_name
		FROM groups g
		LEFT JOIN group_role_sets grs ON g.id = grs.group_id
		LEFT JOIN role_sets rs ON grs.role_set_id = rs.id
		LEFT JOIN role_set_roles rsr ON rs.id = rsr.role_set_id
		LEFT JOIN roles role ON rsr.role_id = role.id
		WHERE g.id = ?`
	} else {
		query = `SELECT 
			g.id AS group_id, 
			g.name AS group_name,
			rs.id AS role_set_id, 
			rs.name AS role_set_name,
			role.id AS role_id, 
			role.name AS role_name
		FROM groups g
		LEFT JOIN group_role_sets grs ON g.id = grs.group_id
		LEFT JOIN role_sets rs ON grs.role_set_id = rs.id
		LEFT JOIN role_set_roles rsr ON rs.id = rsr.role_set_id
		LEFT JOIN roles role ON rsr.role_id = role.id
		WHERE g.id = $1`
	}

	rows, err := getExecutor(ctx, r.db).QueryContext(ctx, query, id.String())
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var groupIDStr, name string
	var roleSets []*tempRoleSet
	roleSetsMap := make(map[uuid.UUID]*tempRoleSet)
	roleSetRolesMap := make(map[uuid.UUID]map[uuid.UUID]bool)

	hasRows := false
	for rows.Next() {
		hasRows = true
		var gID, gName string
		var rsIDNull, rsNameNull sql.NullString
		var rIDNull, rNameNull sql.NullString

		err := rows.Scan(
			&gID, &gName,
			&rsIDNull, &rsNameNull,
			&rIDNull, &rNameNull,
		)
		if err != nil {
			return nil, err
		}

		if groupIDStr == "" {
			groupIDStr = gID
			name = gName
		}

		if rsIDNull.Valid && rsIDNull.String != "" {
			rsID, err := uuid.Parse(rsIDNull.String)
			if err == nil {
				roleSet, rsExists := roleSetsMap[rsID]
				if !rsExists {
					roleSet = &tempRoleSet{
						ID:   rsID,
						Name: rsNameNull.String,
					}
					roleSetsMap[rsID] = roleSet
					roleSets = append(roleSets, roleSet)
					roleSetRolesMap[rsID] = make(map[uuid.UUID]bool)
				}

				if rIDNull.Valid && rIDNull.String != "" {
					rID, err := uuid.Parse(rIDNull.String)
					if err == nil {
						if !roleSetRolesMap[rsID][rID] {
							roleSetRolesMap[rsID][rID] = true
							roleSet.Roles = append(roleSet.Roles, &tempRole{
								ID:   rID,
								Name: rNameNull.String,
							})
						}
					}
				}
			}
		}
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	if !hasRows {
		return nil, nil
	}

	domainRoleSets := make([]domain.RoleSet, len(roleSets))
	for j, trs := range roleSets {
		domainRoles := make([]domain.Role, len(trs.Roles))
		for k, tr := range trs.Roles {
			domainRoles[k] = domain.Role{
				ID:   tr.ID,
				Name: tr.Name,
			}
		}
		domainRoleSets[j] = domain.RoleSet{
			ID:    trs.ID,
			Name:  trs.Name,
			Roles: domainRoles,
		}
	}

	groupUUID, err := uuid.Parse(groupIDStr)
	if err != nil {
		return nil, err
	}

	return &domain.Group{
		ID:       groupUUID,
		Name:     name,
		RoleSets: domainRoleSets,
	}, nil
}

// List returns all groups ordered by name.
func (r *SQLGroupRepository) List(ctx context.Context) ([]domain.Group, error) {
	query := "SELECT id, name FROM groups ORDER BY name"
	rows, err := getExecutor(ctx, r.db).QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var groups []domain.Group
	for rows.Next() {
		var g domain.Group
		var idStr string
		if err := rows.Scan(&idStr, &g.Name); err != nil {
			return nil, err
		}
		g.ID, err = uuid.Parse(idStr)
		if err != nil {
			return nil, err
		}
		groups = append(groups, g)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return groups, nil
}

// Delete removes a group by ID. Cascade handles junction table cleanup.
func (r *SQLGroupRepository) Delete(ctx context.Context, id uuid.UUID) error {
	var query string
	if r.isSQLite {
		query = "DELETE FROM groups WHERE id = ?"
	} else {
		query = "DELETE FROM groups WHERE id = $1"
	}
	_, err := getExecutor(ctx, r.db).ExecContext(ctx, query, id.String())
	return err
}

// Save inserts or updates a group.
func (r *SQLGroupRepository) Save(ctx context.Context, group *domain.Group) error {
	var query string
	if r.isSQLite {
		query = `INSERT INTO groups (id, name) VALUES (?, ?)
                 ON CONFLICT(id) DO UPDATE SET name = excluded.name`
	} else {
		query = `INSERT INTO groups (id, name) VALUES ($1, $2)
                 ON CONFLICT(id) DO UPDATE SET name = EXCLUDED.name`
	}

	_, err := getExecutor(ctx, r.db).ExecContext(ctx, query, group.ID.String(), group.Name)
	return err
}

// AssignGroupToUser associates a user with a group.
func (r *SQLGroupRepository) AssignGroupToUser(ctx context.Context, userID uuid.UUID, groupID uuid.UUID) error {
	var query string
	if r.isSQLite {
		query = `INSERT INTO user_groups (user_id, group_id) VALUES (?, ?)
                 ON CONFLICT DO NOTHING`
	} else {
		query = `INSERT INTO user_groups (user_id, group_id) VALUES ($1, $2)
                 ON CONFLICT DO NOTHING`
	}

	_, err := getExecutor(ctx, r.db).ExecContext(ctx, query, userID.String(), groupID.String())
	return err
}

// RemoveGroupFromUser removes a group assignment from a user.
func (r *SQLGroupRepository) RemoveGroupFromUser(ctx context.Context, userID uuid.UUID, groupID uuid.UUID) error {
	var query string
	if r.isSQLite {
		query = "DELETE FROM user_groups WHERE user_id = ? AND group_id = ?"
	} else {
		query = "DELETE FROM user_groups WHERE user_id = $1 AND group_id = $2"
	}
	_, err := getExecutor(ctx, r.db).ExecContext(ctx, query, userID.String(), groupID.String())
	return err
}

// SQLRoleSetRepository implements ports.RoleSetRepository.
type SQLRoleSetRepository struct {
	db       *sql.DB
	isSQLite bool
}

// NewSQLRoleSetRepository creates a new SQLRoleSetRepository.
func NewSQLRoleSetRepository(db *sql.DB, isSQLite bool) *SQLRoleSetRepository {
	return &SQLRoleSetRepository{
		db:       db,
		isSQLite: isSQLite,
	}
}

// GetByID retrieves a role set and its associated roles in a single query.
func (r *SQLRoleSetRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.RoleSet, error) {
	var query string
	if r.isSQLite {
		query = `SELECT 
			rs.id AS role_set_id, 
			rs.name AS role_set_name,
			role.id AS role_id, 
			role.name AS role_name
		FROM role_sets rs
		LEFT JOIN role_set_roles rsr ON rs.id = rsr.role_set_id
		LEFT JOIN roles role ON rsr.role_id = role.id
		WHERE rs.id = ?`
	} else {
		query = `SELECT 
			rs.id AS role_set_id, 
			rs.name AS role_set_name,
			role.id AS role_id, 
			role.name AS role_name
		FROM role_sets rs
		LEFT JOIN role_set_roles rsr ON rs.id = rsr.role_set_id
		LEFT JOIN roles role ON rsr.role_id = role.id
		WHERE rs.id = $1`
	}

	rows, err := getExecutor(ctx, r.db).QueryContext(ctx, query, id.String())
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var roleSetIDStr, name string
	var roles []domain.Role
	rolesMap := make(map[uuid.UUID]bool)

	hasRows := false
	for rows.Next() {
		hasRows = true
		var rsID, rsName string
		var rIDNull, rNameNull sql.NullString

		err := rows.Scan(
			&rsID, &rsName,
			&rIDNull, &rNameNull,
		)
		if err != nil {
			return nil, err
		}

		if roleSetIDStr == "" {
			roleSetIDStr = rsID
			name = rsName
		}

		if rIDNull.Valid && rIDNull.String != "" {
			rID, err := uuid.Parse(rIDNull.String)
			if err == nil {
				if !rolesMap[rID] {
					rolesMap[rID] = true
					roles = append(roles, domain.Role{
						ID:   rID,
						Name: rNameNull.String,
					})
				}
			}
		}
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	if !hasRows {
		return nil, nil
	}

	rsUUID, err := uuid.Parse(roleSetIDStr)
	if err != nil {
		return nil, err
	}

	return &domain.RoleSet{
		ID:    rsUUID,
		Name:  name,
		Roles: roles,
	}, nil
}

// List returns all role sets ordered by name.
func (r *SQLRoleSetRepository) List(ctx context.Context) ([]domain.RoleSet, error) {
	query := "SELECT id, name FROM role_sets ORDER BY name"
	rows, err := getExecutor(ctx, r.db).QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var roleSets []domain.RoleSet
	for rows.Next() {
		var rs domain.RoleSet
		var idStr string
		if err := rows.Scan(&idStr, &rs.Name); err != nil {
			return nil, err
		}
		rs.ID, err = uuid.Parse(idStr)
		if err != nil {
			return nil, err
		}
		roleSets = append(roleSets, rs)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return roleSets, nil
}

// Delete removes a role set by ID. Cascade handles junction table cleanup.
func (r *SQLRoleSetRepository) Delete(ctx context.Context, id uuid.UUID) error {
	var query string
	if r.isSQLite {
		query = "DELETE FROM role_sets WHERE id = ?"
	} else {
		query = "DELETE FROM role_sets WHERE id = $1"
	}
	_, err := getExecutor(ctx, r.db).ExecContext(ctx, query, id.String())
	return err
}

// Save inserts or updates a role set.
func (r *SQLRoleSetRepository) Save(ctx context.Context, roleSet *domain.RoleSet) error {
	var query string
	if r.isSQLite {
		query = `INSERT INTO role_sets (id, name) VALUES (?, ?)
                 ON CONFLICT(id) DO UPDATE SET name = excluded.name`
	} else {
		query = `INSERT INTO role_sets (id, name) VALUES ($1, $2)
                 ON CONFLICT(id) DO UPDATE SET name = EXCLUDED.name`
	}

	_, err := getExecutor(ctx, r.db).ExecContext(ctx, query, roleSet.ID.String(), roleSet.Name)
	return err
}

// AssignRoleSetToGroup associates a role set with a group.
func (r *SQLRoleSetRepository) AssignRoleSetToGroup(ctx context.Context, groupID uuid.UUID, roleSetID uuid.UUID) error {
	var query string
	if r.isSQLite {
		query = `INSERT INTO group_role_sets (group_id, role_set_id) VALUES (?, ?)
                 ON CONFLICT DO NOTHING`
	} else {
		query = `INSERT INTO group_role_sets (group_id, role_set_id) VALUES ($1, $2)
                 ON CONFLICT DO NOTHING`
	}

	_, err := getExecutor(ctx, r.db).ExecContext(ctx, query, groupID.String(), roleSetID.String())
	return err
}

// RemoveRoleSetFromGroup removes a role set assignment from a group.
func (r *SQLRoleSetRepository) RemoveRoleSetFromGroup(ctx context.Context, groupID uuid.UUID, roleSetID uuid.UUID) error {
	var query string
	if r.isSQLite {
		query = "DELETE FROM group_role_sets WHERE group_id = ? AND role_set_id = ?"
	} else {
		query = "DELETE FROM group_role_sets WHERE group_id = $1 AND role_set_id = $2"
	}
	_, err := getExecutor(ctx, r.db).ExecContext(ctx, query, groupID.String(), roleSetID.String())
	return err
}

// SQLRoleRepository implements ports.RoleRepository.
type SQLRoleRepository struct {
	db       *sql.DB
	isSQLite bool
}

// NewSQLRoleRepository creates a new SQLRoleRepository.
func NewSQLRoleRepository(db *sql.DB, isSQLite bool) *SQLRoleRepository {
	return &SQLRoleRepository{
		db:       db,
		isSQLite: isSQLite,
	}
}

// GetByID retrieves a role by its ID.
func (r *SQLRoleRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Role, error) {
	var query string
	if r.isSQLite {
		query = "SELECT id, name FROM roles WHERE id = ?"
	} else {
		query = "SELECT id, name FROM roles WHERE id = $1"
	}

	var role domain.Role
	var idStr string
	err := getExecutor(ctx, r.db).QueryRowContext(ctx, query, id.String()).Scan(&idStr, &role.Name)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	parsedID, err := uuid.Parse(idStr)
	if err != nil {
		return nil, err
	}
	role.ID = parsedID

	return &role, nil
}

// List returns all roles ordered by name.
func (r *SQLRoleRepository) List(ctx context.Context) ([]domain.Role, error) {
	query := "SELECT id, name FROM roles ORDER BY name"
	rows, err := getExecutor(ctx, r.db).QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var roles []domain.Role
	for rows.Next() {
		var role domain.Role
		var idStr string
		if err := rows.Scan(&idStr, &role.Name); err != nil {
			return nil, err
		}
		role.ID, err = uuid.Parse(idStr)
		if err != nil {
			return nil, err
		}
		roles = append(roles, role)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return roles, nil
}

// Delete removes a role by ID. Cascade handles junction table cleanup.
func (r *SQLRoleRepository) Delete(ctx context.Context, id uuid.UUID) error {
	var query string
	if r.isSQLite {
		query = "DELETE FROM roles WHERE id = ?"
	} else {
		query = "DELETE FROM roles WHERE id = $1"
	}
	_, err := getExecutor(ctx, r.db).ExecContext(ctx, query, id.String())
	return err
}

// Save inserts or updates a role.
func (r *SQLRoleRepository) Save(ctx context.Context, role *domain.Role) error {
	var query string
	if r.isSQLite {
		query = `INSERT INTO roles (id, name) VALUES (?, ?)
                 ON CONFLICT(id) DO UPDATE SET name = excluded.name`
	} else {
		query = `INSERT INTO roles (id, name) VALUES ($1, $2)
                 ON CONFLICT(id) DO UPDATE SET name = EXCLUDED.name`
	}

	_, err := getExecutor(ctx, r.db).ExecContext(ctx, query, role.ID.String(), role.Name)
	return err
}

// AssignRoleToRoleSet associates a role with a role set.
func (r *SQLRoleRepository) AssignRoleToRoleSet(ctx context.Context, roleSetID uuid.UUID, roleID uuid.UUID) error {
	var query string
	if r.isSQLite {
		query = `INSERT INTO role_set_roles (role_set_id, role_id) VALUES (?, ?)
                 ON CONFLICT DO NOTHING`
	} else {
		query = `INSERT INTO role_set_roles (role_set_id, role_id) VALUES ($1, $2)
                 ON CONFLICT DO NOTHING`
	}

	_, err := getExecutor(ctx, r.db).ExecContext(ctx, query, roleSetID.String(), roleID.String())
	return err
}

// RemoveRoleFromRoleSet removes a role assignment from a role set.
func (r *SQLRoleRepository) RemoveRoleFromRoleSet(ctx context.Context, roleSetID uuid.UUID, roleID uuid.UUID) error {
	var query string
	if r.isSQLite {
		query = "DELETE FROM role_set_roles WHERE role_set_id = ? AND role_id = ?"
	} else {
		query = "DELETE FROM role_set_roles WHERE role_set_id = $1 AND role_id = $2"
	}
	_, err := getExecutor(ctx, r.db).ExecContext(ctx, query, roleSetID.String(), roleID.String())
	return err
}
