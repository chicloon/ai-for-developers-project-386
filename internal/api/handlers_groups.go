package api

import (
	"encoding/json"
	"net/http"

	"call-booking/internal/auth"
	"call-booking/internal/models"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type groupsHandler struct {
	pool *pgxpool.Pool
}

func groupsRouter(pool *pgxpool.Pool) http.Handler {
	r := chi.NewRouter()
	h := &groupsHandler{pool: pool}

	r.Use(auth.Middleware)
	r.Get("/", h.list)
	r.Post("/", h.create)
	r.Put("/{id}", h.update)
	r.Delete("/{id}", h.delete)

	// Member management routes
	r.Get("/{id}/members", h.listMembers)
	r.Post("/{id}/members", h.addMember)
	r.Delete("/{id}/members/{memberId}", h.removeMember)

	return r
}

// list returns all groups owned by the current user
func (h *groupsHandler) list(w http.ResponseWriter, r *http.Request) {
	userID := auth.GetUserID(r.Context())

	rows, err := h.pool.Query(r.Context(),
		"SELECT id, owner_id, name, visibility_level, TO_CHAR(created_at, 'YYYY-MM-DD\"T\"HH24:MI:SS\"Z\"') FROM visibility_groups WHERE owner_id = $1 ORDER BY created_at DESC",
		userID)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, "database error")
		return
	}
	defer rows.Close()

	var groups []models.VisibilityGroup
	for rows.Next() {
		var g models.VisibilityGroup
		if err := rows.Scan(&g.ID, &g.OwnerID, &g.Name, &g.VisibilityLevel, &g.CreatedAt); err != nil {
			jsonError(w, http.StatusInternalServerError, "database error")
			return
		}
		groups = append(groups, g)
	}

	if groups == nil {
		groups = []models.VisibilityGroup{}
	}

	jsonResponse(w, http.StatusOK, map[string]interface{}{"groups": groups})
}

// create creates a new visibility group
func (h *groupsHandler) create(w http.ResponseWriter, r *http.Request) {
	userID := auth.GetUserID(r.Context())

	var req models.CreateGroupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Validate
	if req.Name == "" {
		jsonError(w, http.StatusBadRequest, "name is required")
		return
	}
	if req.VisibilityLevel == "" {
		jsonError(w, http.StatusBadRequest, "visibilityLevel is required")
		return
	}
	validLevels := map[string]bool{"family": true, "work": true, "friends": true, "public": true}
	if !validLevels[req.VisibilityLevel] {
		jsonError(w, http.StatusBadRequest, "visibilityLevel must be 'family', 'work', 'friends', or 'public'")
		return
	}

	var g models.VisibilityGroup
	err := h.pool.QueryRow(r.Context(),
		"INSERT INTO visibility_groups (owner_id, name, visibility_level) VALUES ($1, $2, $3) RETURNING id, owner_id, name, visibility_level, TO_CHAR(created_at, 'YYYY-MM-DD\"T\"HH24:MI:SS\"Z\"')",
		userID, req.Name, req.VisibilityLevel).
		Scan(&g.ID, &g.OwnerID, &g.Name, &g.VisibilityLevel, &g.CreatedAt)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, "database error")
		return
	}

	jsonResponse(w, http.StatusCreated, g)
}

// update updates a group (only owner can update)
func (h *groupsHandler) update(w http.ResponseWriter, r *http.Request) {
	userID := auth.GetUserID(r.Context())
	groupID := chi.URLParam(r, "id")

	var req models.CreateGroupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Validate
	if req.Name == "" {
		jsonError(w, http.StatusBadRequest, "name is required")
		return
	}
	if req.VisibilityLevel == "" {
		jsonError(w, http.StatusBadRequest, "visibilityLevel is required")
		return
	}
	validLevels := map[string]bool{"family": true, "work": true, "friends": true, "public": true}
	if !validLevels[req.VisibilityLevel] {
		jsonError(w, http.StatusBadRequest, "visibilityLevel must be 'family', 'work', 'friends', or 'public'")
		return
	}

	var g models.VisibilityGroup
	err := h.pool.QueryRow(r.Context(),
		"UPDATE visibility_groups SET name=$1, visibility_level=$2 WHERE id=$3 AND owner_id=$4 RETURNING id, owner_id, name, visibility_level, TO_CHAR(created_at, 'YYYY-MM-DD\"T\"HH24:MI:SS\"Z\"')",
		req.Name, req.VisibilityLevel, groupID, userID).
		Scan(&g.ID, &g.OwnerID, &g.Name, &g.VisibilityLevel, &g.CreatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			jsonError(w, http.StatusNotFound, "group not found")
			return
		}
		jsonError(w, http.StatusInternalServerError, "database error")
		return
	}

	jsonResponse(w, http.StatusOK, g)
}

// delete deletes a group (only owner can delete)
func (h *groupsHandler) delete(w http.ResponseWriter, r *http.Request) {
	userID := auth.GetUserID(r.Context())
	groupID := chi.URLParam(r, "id")

	result, err := h.pool.Exec(r.Context(),
		"DELETE FROM visibility_groups WHERE id = $1 AND owner_id = $2",
		groupID, userID)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, "database error")
		return
	}

	if result.RowsAffected() == 0 {
		jsonError(w, http.StatusNotFound, "group not found")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// listMembers returns all members of a group (only owner can view)
func (h *groupsHandler) listMembers(w http.ResponseWriter, r *http.Request) {
	userID := auth.GetUserID(r.Context())
	groupID := chi.URLParam(r, "id")

	// Verify ownership first
	var ownerID string
	err := h.pool.QueryRow(r.Context(),
		"SELECT owner_id FROM visibility_groups WHERE id = $1",
		groupID).Scan(&ownerID)
	if err != nil {
		if err == pgx.ErrNoRows {
			jsonError(w, http.StatusNotFound, "group not found")
			return
		}
		jsonError(w, http.StatusInternalServerError, "database error")
		return
	}

	if ownerID != userID {
		jsonError(w, http.StatusForbidden, "you don't have access to this group")
		return
	}

	// Get members with user info
	rows, err := h.pool.Query(r.Context(),
		`SELECT gm.id, gm.group_id, gm.added_by, TO_CHAR(gm.added_at, 'YYYY-MM-DD"T"HH24:MI:SS"Z"'),
			u.id, u.email, u.name, TO_CHAR(u.created_at, 'YYYY-MM-DD"T"HH24:MI:SS"Z"')
		 FROM group_members gm
		 JOIN users u ON gm.member_id = u.id
		 WHERE gm.group_id = $1
		 ORDER BY gm.added_at DESC`,
		groupID)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, "database error")
		return
	}
	defer rows.Close()

	var members []models.GroupMember
	for rows.Next() {
		var m models.GroupMember
		var user models.User
		if err := rows.Scan(&m.ID, &m.GroupID, &m.AddedBy, &m.AddedAt,
			&user.ID, &user.Email, &user.Name, &user.CreatedAt); err != nil {
			jsonError(w, http.StatusInternalServerError, "database error")
			return
		}
		m.Member = user
		members = append(members, m)
	}

	if members == nil {
		members = []models.GroupMember{}
	}

	jsonResponse(w, http.StatusOK, map[string]interface{}{"members": members})
}

// addMember adds a member to a group by email or userId (only owner can add)
func (h *groupsHandler) addMember(w http.ResponseWriter, r *http.Request) {
	userID := auth.GetUserID(r.Context())
	groupID := chi.URLParam(r, "id")

	var req models.AddMemberRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Verify ownership first
	var ownerID string
	err := h.pool.QueryRow(r.Context(),
		"SELECT owner_id FROM visibility_groups WHERE id = $1",
		groupID).Scan(&ownerID)
	if err != nil {
		if err == pgx.ErrNoRows {
			jsonError(w, http.StatusNotFound, "group not found")
			return
		}
		jsonError(w, http.StatusInternalServerError, "database error")
		return
	}

	if ownerID != userID {
		jsonError(w, http.StatusForbidden, "you don't have access to this group")
		return
	}

	// Find member to add by email or userId
	var memberID string
	if req.UserID != nil && *req.UserID != "" {
		// Verify user exists
		err = h.pool.QueryRow(r.Context(),
			"SELECT id FROM users WHERE id = $1",
			*req.UserID).Scan(&memberID)
		if err != nil {
			if err == pgx.ErrNoRows {
				jsonError(w, http.StatusNotFound, "user not found")
				return
			}
			jsonError(w, http.StatusInternalServerError, "database error")
			return
		}
	} else if req.Email != nil && *req.Email != "" {
		// Find user by email
		err = h.pool.QueryRow(r.Context(),
			"SELECT id FROM users WHERE email = $1",
			*req.Email).Scan(&memberID)
		if err != nil {
			if err == pgx.ErrNoRows {
				jsonError(w, http.StatusNotFound, "user not found")
				return
			}
			jsonError(w, http.StatusInternalServerError, "database error")
			return
		}
	} else {
		jsonError(w, http.StatusBadRequest, "either email or userId is required")
		return
	}

	// Cannot add owner as member
	if memberID == ownerID {
		jsonError(w, http.StatusBadRequest, "owner cannot be added as a member")
		return
	}

	// Insert member
	var m models.GroupMember
	var user models.User
	err = h.pool.QueryRow(r.Context(),
		`INSERT INTO group_members (group_id, member_id, added_by) VALUES ($1, $2, $3)
		 RETURNING id, group_id, added_by, TO_CHAR(added_at, 'YYYY-MM-DD"T"HH24:MI:SS"Z"')`,
		groupID, memberID, userID).
		Scan(&m.ID, &m.GroupID, &m.AddedBy, &m.AddedAt)
	if err != nil {
		// Check for unique constraint violation
		if err.Error() == "ERROR: duplicate key value violates unique constraint \"group_members_group_id_member_id_key\" (SQLSTATE 23505)" {
			jsonError(w, http.StatusConflict, "user is already a member of this group")
			return
		}
		jsonError(w, http.StatusInternalServerError, "database error")
		return
	}

	// Get user info
	err = h.pool.QueryRow(r.Context(),
		"SELECT id, email, name, TO_CHAR(created_at, 'YYYY-MM-DD\"T\"HH24:MI:SS\"Z\"') FROM users WHERE id = $1",
		memberID).Scan(&user.ID, &user.Email, &user.Name, &user.CreatedAt)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, "database error")
		return
	}
	m.Member = user

	jsonResponse(w, http.StatusCreated, m)
}

// removeMember removes a member from a group (only owner can remove)
func (h *groupsHandler) removeMember(w http.ResponseWriter, r *http.Request) {
	userID := auth.GetUserID(r.Context())
	groupID := chi.URLParam(r, "id")
	memberID := chi.URLParam(r, "memberId")

	// Verify ownership first
	var ownerID string
	err := h.pool.QueryRow(r.Context(),
		"SELECT owner_id FROM visibility_groups WHERE id = $1",
		groupID).Scan(&ownerID)
	if err != nil {
		if err == pgx.ErrNoRows {
			jsonError(w, http.StatusNotFound, "group not found")
			return
		}
		jsonError(w, http.StatusInternalServerError, "database error")
		return
	}

	if ownerID != userID {
		jsonError(w, http.StatusForbidden, "you don't have access to this group")
		return
	}

	result, err := h.pool.Exec(r.Context(),
		"DELETE FROM group_members WHERE id = $1 AND group_id = $2",
		memberID, groupID)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, "database error")
		return
	}

	if result.RowsAffected() == 0 {
		jsonError(w, http.StatusNotFound, "member not found")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
