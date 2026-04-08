package api

import (
	"context"
	"net/http"
	"testing"

	"call-booking/internal/models"
)

func TestGroupsList_Empty(t *testing.T) {
	pool := setupTestDB(t)
	defer pool.Close()
	defer cleanupTestData(t, pool)

	router := NewRouter(pool)

	// Create user
	email := "groups@example.com"
	userID := createTestUser(t, pool, email, "password123", "Groups Test User")
	token := getAuthToken(userID, email)

	rr := makeRequest(router, "GET", "/api/my/groups", nil, token)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp map[string][]models.VisibilityGroup
	parseResponse(t, rr, &resp)

	if resp["groups"] == nil {
		t.Error("expected empty array, not nil")
	}
	if len(resp["groups"]) != 0 {
		t.Errorf("expected 0 groups, got %d", len(resp["groups"]))
	}
}

func TestGroupsList_WithGroups(t *testing.T) {
	pool := setupTestDB(t)
	defer pool.Close()
	defer cleanupTestData(t, pool)

	router := NewRouter(pool)
	ctx := context.Background()

	// Create user
	email := "groups2@example.com"
	userID := createTestUser(t, pool, email, "password123", "Groups Test User")
	token := getAuthToken(userID, email)

	// Insert test groups
	_, err := pool.Exec(ctx, "INSERT INTO visibility_groups (owner_id, name, visibility_level) VALUES ($1, 'Family', 'family')", userID)
	if err != nil {
		t.Fatalf("failed to insert group: %v", err)
	}
	_, err = pool.Exec(ctx, "INSERT INTO visibility_groups (owner_id, name, visibility_level) VALUES ($1, 'Work', 'work')", userID)
	if err != nil {
		t.Fatalf("failed to insert group: %v", err)
	}

	rr := makeRequest(router, "GET", "/api/my/groups", nil, token)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp map[string][]models.VisibilityGroup
	parseResponse(t, rr, &resp)

	if len(resp["groups"]) != 2 {
		t.Errorf("expected 2 groups, got %d", len(resp["groups"]))
	}

	// Check visibility levels
	familyFound := false
	workFound := false
	for _, g := range resp["groups"] {
		if g.VisibilityLevel == "family" {
			familyFound = true
		}
		if g.VisibilityLevel == "work" {
			workFound = true
		}
	}
	if !familyFound {
		t.Error("expected to find family group")
	}
	if !workFound {
		t.Error("expected to find work group")
	}
}

func TestGroupsList_Unauthorized(t *testing.T) {
	pool := setupTestDB(t)
	defer pool.Close()

	router := NewRouter(pool)

	rr := makeRequest(router, "GET", "/api/my/groups", nil, "")

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", rr.Code)
	}
}

func TestGroupsCreate_Success(t *testing.T) {
	pool := setupTestDB(t)
	defer pool.Close()
	defer cleanupTestData(t, pool)

	router := NewRouter(pool)

	// Create user
	email := "groups3@example.com"
	userID := createTestUser(t, pool, email, "password123", "Groups Test User")
	token := getAuthToken(userID, email)

	req := models.CreateGroupRequest{
		Name:            "Test Group",
		VisibilityLevel: "friends",
	}

	rr := makeRequest(router, "POST", "/api/my/groups", req, token)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp models.VisibilityGroup
	parseResponse(t, rr, &resp)

	if resp.OwnerID != userID {
		t.Errorf("expected owner ID %s, got %s", userID, resp.OwnerID)
	}
	if resp.Name != req.Name {
		t.Errorf("expected name %s, got %s", req.Name, resp.Name)
	}
	if resp.VisibilityLevel != req.VisibilityLevel {
		t.Errorf("expected visibility level %s, got %s", req.VisibilityLevel, resp.VisibilityLevel)
	}
}

func TestGroupsCreate_MissingName(t *testing.T) {
	pool := setupTestDB(t)
	defer pool.Close()
	defer cleanupTestData(t, pool)

	router := NewRouter(pool)

	// Create user
	email := "groups4@example.com"
	userID := createTestUser(t, pool, email, "password123", "Groups Test User")
	token := getAuthToken(userID, email)

	req := models.CreateGroupRequest{
		Name:            "",
		VisibilityLevel: "public",
	}

	rr := makeRequest(router, "POST", "/api/my/groups", req, token)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rr.Code)
	}

	errResp := parseErrorResponse(t, rr)
	if errResp.Error != "name is required" {
		t.Errorf("unexpected error message: %s", errResp.Error)
	}
}

func TestGroupsCreate_MissingVisibilityLevel(t *testing.T) {
	pool := setupTestDB(t)
	defer pool.Close()
	defer cleanupTestData(t, pool)

	router := NewRouter(pool)

	// Create user
	email := "groups5@example.com"
	userID := createTestUser(t, pool, email, "password123", "Groups Test User")
	token := getAuthToken(userID, email)

	req := models.CreateGroupRequest{
		Name:            "Test Group",
		VisibilityLevel: "",
	}

	rr := makeRequest(router, "POST", "/api/my/groups", req, token)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rr.Code)
	}

	errResp := parseErrorResponse(t, rr)
	if errResp.Error != "visibilityLevel is required" {
		t.Errorf("unexpected error message: %s", errResp.Error)
	}
}

func TestGroupsCreate_InvalidVisibilityLevel(t *testing.T) {
	pool := setupTestDB(t)
	defer pool.Close()
	defer cleanupTestData(t, pool)

	router := NewRouter(pool)

	// Create user
	email := "groups6@example.com"
	userID := createTestUser(t, pool, email, "password123", "Groups Test User")
	token := getAuthToken(userID, email)

	req := models.CreateGroupRequest{
		Name:            "Test Group",
		VisibilityLevel: "invalid-level",
	}

	rr := makeRequest(router, "POST", "/api/my/groups", req, token)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rr.Code)
	}

	errResp := parseErrorResponse(t, rr)
	if errResp.Error != "visibilityLevel must be 'family', 'work', 'friends', or 'public'" {
		t.Errorf("unexpected error message: %s", errResp.Error)
	}
}

func TestGroupsCreate_Unauthorized(t *testing.T) {
	pool := setupTestDB(t)
	defer pool.Close()

	router := NewRouter(pool)

	req := models.CreateGroupRequest{
		Name:            "Test Group",
		VisibilityLevel: "public",
	}

	rr := makeRequest(router, "POST", "/api/my/groups", req, "")

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", rr.Code)
	}
}

func TestGroupsUpdate_Success(t *testing.T) {
	pool := setupTestDB(t)
	defer pool.Close()
	defer cleanupTestData(t, pool)

	router := NewRouter(pool)
	ctx := context.Background()

	// Create user
	email := "groups7@example.com"
	userID := createTestUser(t, pool, email, "password123", "Groups Test User")
	token := getAuthToken(userID, email)

	// Insert a group
	var groupID string
	err := pool.QueryRow(ctx, "INSERT INTO visibility_groups (owner_id, name, visibility_level) VALUES ($1, 'Original Name', 'family') RETURNING id", userID).Scan(&groupID)
	if err != nil {
		t.Fatalf("failed to insert group: %v", err)
	}

	req := models.CreateGroupRequest{
		Name:            "Updated Name",
		VisibilityLevel: "work",
	}

	rr := makeRequest(router, "PUT", "/api/my/groups/"+groupID, req, token)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp models.VisibilityGroup
	parseResponse(t, rr, &resp)

	if resp.ID != groupID {
		t.Errorf("expected group ID %s, got %s", groupID, resp.ID)
	}
	if resp.Name != req.Name {
		t.Errorf("expected name %s, got %s", req.Name, resp.Name)
	}
	if resp.VisibilityLevel != req.VisibilityLevel {
		t.Errorf("expected visibility level %s, got %s", req.VisibilityLevel, resp.VisibilityLevel)
	}
}

func TestGroupsUpdate_NotFound(t *testing.T) {
	pool := setupTestDB(t)
	defer pool.Close()
	defer cleanupTestData(t, pool)

	router := NewRouter(pool)

	// Create user
	email := "groups8@example.com"
	userID := createTestUser(t, pool, email, "password123", "Groups Test User")
	token := getAuthToken(userID, email)

	req := models.CreateGroupRequest{
		Name:            "Updated Name",
		VisibilityLevel: "public",
	}

	rr := makeRequest(router, "PUT", "/api/my/groups/nonexistent-id", req, token)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", rr.Code)
	}

	errResp := parseErrorResponse(t, rr)
	if errResp.Error != "group not found" {
		t.Errorf("unexpected error message: %s", errResp.Error)
	}
}

func TestGroupsUpdate_NotOwner(t *testing.T) {
	pool := setupTestDB(t)
	defer pool.Close()
	defer cleanupTestData(t, pool)

	router := NewRouter(pool)
	ctx := context.Background()

	// Create two users
	ownerEmail := "groupowner@example.com"
	ownerID := createTestUser(t, pool, ownerEmail, "password123", "Group Owner")

	otherEmail := "groupother@example.com"
	otherID := createTestUser(t, pool, otherEmail, "password123", "Other User")
	otherToken := getAuthToken(otherID, otherEmail)

	// Insert a group for owner
	var groupID string
	err := pool.QueryRow(ctx, "INSERT INTO visibility_groups (owner_id, name, visibility_level) VALUES ($1, 'Private Group', 'family') RETURNING id", ownerID).Scan(&groupID)
	if err != nil {
		t.Fatalf("failed to insert group: %v", err)
	}

	req := models.CreateGroupRequest{
		Name:            "Hacked Name",
		VisibilityLevel: "public",
	}

	rr := makeRequest(router, "PUT", "/api/my/groups/"+groupID, req, otherToken)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected status 404 (not visible to other user), got %d", rr.Code)
	}
}

func TestGroupsDelete_Success(t *testing.T) {
	pool := setupTestDB(t)
	defer pool.Close()
	defer cleanupTestData(t, pool)

	router := NewRouter(pool)
	ctx := context.Background()

	// Create user
	email := "groups9@example.com"
	userID := createTestUser(t, pool, email, "password123", "Groups Test User")
	token := getAuthToken(userID, email)

	// Insert a group
	var groupID string
	err := pool.QueryRow(ctx, "INSERT INTO visibility_groups (owner_id, name, visibility_level) VALUES ($1, 'Group to Delete', 'friends') RETURNING id", userID).Scan(&groupID)
	if err != nil {
		t.Fatalf("failed to insert group: %v", err)
	}

	rr := makeRequest(router, "DELETE", "/api/my/groups/"+groupID, nil, token)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("expected status 204, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestGroupsDelete_NotFound(t *testing.T) {
	pool := setupTestDB(t)
	defer pool.Close()
	defer cleanupTestData(t, pool)

	router := NewRouter(pool)

	// Create user
	email := "groups10@example.com"
	userID := createTestUser(t, pool, email, "password123", "Groups Test User")
	token := getAuthToken(userID, email)

	rr := makeRequest(router, "DELETE", "/api/my/groups/nonexistent-id", nil, token)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", rr.Code)
	}

	errResp := parseErrorResponse(t, rr)
	if errResp.Error != "group not found" {
		t.Errorf("unexpected error message: %s", errResp.Error)
	}
}

func TestGroupsDelete_NotOwner(t *testing.T) {
	pool := setupTestDB(t)
	defer pool.Close()
	defer cleanupTestData(t, pool)

	router := NewRouter(pool)
	ctx := context.Background()

	// Create two users
	ownerEmail := "groupowner2@example.com"
	ownerID := createTestUser(t, pool, ownerEmail, "password123", "Group Owner")

	otherEmail := "groupother2@example.com"
	otherID := createTestUser(t, pool, otherEmail, "password123", "Other User")
	otherToken := getAuthToken(otherID, otherEmail)

	// Insert a group for owner
	var groupID string
	err := pool.QueryRow(ctx, "INSERT INTO visibility_groups (owner_id, name, visibility_level) VALUES ($1, 'Private Group', 'family') RETURNING id", ownerID).Scan(&groupID)
	if err != nil {
		t.Fatalf("failed to insert group: %v", err)
	}

	rr := makeRequest(router, "DELETE", "/api/my/groups/"+groupID, nil, otherToken)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected status 404 (not visible to other user), got %d", rr.Code)
	}
}

func TestGroupsDelete_Unauthorized(t *testing.T) {
	pool := setupTestDB(t)
	defer pool.Close()

	router := NewRouter(pool)

	rr := makeRequest(router, "DELETE", "/api/my/groups/some-id", nil, "")

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", rr.Code)
	}
}

func TestGroupsListMembers_Empty(t *testing.T) {
	pool := setupTestDB(t)
	defer pool.Close()
	defer cleanupTestData(t, pool)

	router := NewRouter(pool)
	ctx := context.Background()

	// Create user
	email := "groups11@example.com"
	userID := createTestUser(t, pool, email, "password123", "Groups Test User")
	token := getAuthToken(userID, email)

	// Insert a group
	var groupID string
	err := pool.QueryRow(ctx, "INSERT INTO visibility_groups (owner_id, name, visibility_level) VALUES ($1, 'Empty Group', 'public') RETURNING id", userID).Scan(&groupID)
	if err != nil {
		t.Fatalf("failed to insert group: %v", err)
	}

	rr := makeRequest(router, "GET", "/api/my/groups/"+groupID+"/members", nil, token)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp map[string][]models.GroupMember
	parseResponse(t, rr, &resp)

	if resp["members"] == nil {
		t.Error("expected empty array, not nil")
	}
	if len(resp["members"]) != 0 {
		t.Errorf("expected 0 members, got %d", len(resp["members"]))
	}
}

func TestGroupsListMembers_WithMembers(t *testing.T) {
	pool := setupTestDB(t)
	defer pool.Close()
	defer cleanupTestData(t, pool)

	router := NewRouter(pool)
	ctx := context.Background()

	// Create owner user
	ownerEmail := "groupowner3@example.com"
	ownerID := createTestUser(t, pool, ownerEmail, "password123", "Group Owner")
	ownerToken := getAuthToken(ownerID, ownerEmail)

	// Create member user
	memberEmail := "groupmember@example.com"
	memberID := createTestUser(t, pool, memberEmail, "password123", "Group Member")

	// Insert a group
	var groupID string
	err := pool.QueryRow(ctx, "INSERT INTO visibility_groups (owner_id, name, visibility_level) VALUES ($1, 'Group with Members', 'family') RETURNING id", ownerID).Scan(&groupID)
	if err != nil {
		t.Fatalf("failed to insert group: %v", err)
	}

	// Add member to group
	_, err = pool.Exec(ctx, "INSERT INTO group_members (group_id, member_id, added_by) VALUES ($1, $2, $1)", groupID, memberID)
	if err != nil {
		t.Fatalf("failed to add member: %v", err)
	}

	rr := makeRequest(router, "GET", "/api/my/groups/"+groupID+"/members", nil, ownerToken)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp map[string][]models.GroupMember
	parseResponse(t, rr, &resp)

	if len(resp["members"]) != 1 {
		t.Errorf("expected 1 member, got %d", len(resp["members"]))
	}

	if len(resp["members"]) > 0 {
		member := resp["members"][0]
		if member.Member.ID != memberID {
			t.Errorf("expected member ID %s, got %s", memberID, member.Member.ID)
		}
		if member.Member.Email != memberEmail {
			t.Errorf("expected member email %s, got %s", memberEmail, member.Member.Email)
		}
	}
}

func TestGroupsListMembers_NotOwner(t *testing.T) {
	pool := setupTestDB(t)
	defer pool.Close()
	defer cleanupTestData(t, pool)

	router := NewRouter(pool)
	ctx := context.Background()

	// Create two users
	ownerEmail := "groupowner4@example.com"
	ownerID := createTestUser(t, pool, ownerEmail, "password123", "Group Owner")

	otherEmail := "groupother3@example.com"
	otherID := createTestUser(t, pool, otherEmail, "password123", "Other User")
	otherToken := getAuthToken(otherID, otherEmail)

	// Insert a group for owner
	var groupID string
	err := pool.QueryRow(ctx, "INSERT INTO visibility_groups (owner_id, name, visibility_level) VALUES ($1, 'Private Group', 'family') RETURNING id", ownerID).Scan(&groupID)
	if err != nil {
		t.Fatalf("failed to insert group: %v", err)
	}

	rr := makeRequest(router, "GET", "/api/my/groups/"+groupID+"/members", nil, otherToken)

	if rr.Code != http.StatusForbidden {
		t.Errorf("expected status 403, got %d", rr.Code)
	}

	errResp := parseErrorResponse(t, rr)
	if errResp.Error != "you don't have access to this group" {
		t.Errorf("unexpected error message: %s", errResp.Error)
	}
}

func TestGroupsAddMember_ByEmail(t *testing.T) {
	pool := setupTestDB(t)
	defer pool.Close()
	defer cleanupTestData(t, pool)

	router := NewRouter(pool)
	ctx := context.Background()

	// Create owner user
	ownerEmail := "groupowner5@example.com"
	ownerID := createTestUser(t, pool, ownerEmail, "password123", "Group Owner")
	ownerToken := getAuthToken(ownerID, ownerEmail)

	// Create user to add as member
	memberEmail := "membertoadd@example.com"
	memberID := createTestUser(t, pool, memberEmail, "password123", "Member to Add")

	// Insert a group
	var groupID string
	err := pool.QueryRow(ctx, "INSERT INTO visibility_groups (owner_id, name, visibility_level) VALUES ($1, 'Group', 'work') RETURNING id", ownerID).Scan(&groupID)
	if err != nil {
		t.Fatalf("failed to insert group: %v", err)
	}

	req := models.AddMemberRequest{
		Email: &memberEmail,
	}

	rr := makeRequest(router, "POST", "/api/my/groups/"+groupID+"/members", req, ownerToken)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp models.GroupMember
	parseResponse(t, rr, &resp)

	if resp.Member.ID != memberID {
		t.Errorf("expected member ID %s, got %s", memberID, resp.Member.ID)
	}
	if resp.GroupID != groupID {
		t.Errorf("expected group ID %s, got %s", groupID, resp.GroupID)
	}
}

func TestGroupsAddMember_ByUserID(t *testing.T) {
	pool := setupTestDB(t)
	defer pool.Close()
	defer cleanupTestData(t, pool)

	router := NewRouter(pool)
	ctx := context.Background()

	// Create owner user
	ownerEmail := "groupowner6@example.com"
	ownerID := createTestUser(t, pool, ownerEmail, "password123", "Group Owner")
	ownerToken := getAuthToken(ownerID, ownerEmail)

	// Create user to add as member
	memberEmail := "membertoadd2@example.com"
	memberID := createTestUser(t, pool, memberEmail, "password123", "Member to Add")

	// Insert a group
	var groupID string
	err := pool.QueryRow(ctx, "INSERT INTO visibility_groups (owner_id, name, visibility_level) VALUES ($1, 'Group', 'work') RETURNING id", ownerID).Scan(&groupID)
	if err != nil {
		t.Fatalf("failed to insert group: %v", err)
	}

	req := models.AddMemberRequest{
		UserID: &memberID,
	}

	rr := makeRequest(router, "POST", "/api/my/groups/"+groupID+"/members", req, ownerToken)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp models.GroupMember
	parseResponse(t, rr, &resp)

	if resp.Member.ID != memberID {
		t.Errorf("expected member ID %s, got %s", memberID, resp.Member.ID)
	}
}

func TestGroupsAddMember_UserNotFound(t *testing.T) {
	pool := setupTestDB(t)
	defer pool.Close()
	defer cleanupTestData(t, pool)

	router := NewRouter(pool)
	ctx := context.Background()

	// Create owner user
	ownerEmail := "groupowner7@example.com"
	ownerID := createTestUser(t, pool, ownerEmail, "password123", "Group Owner")
	ownerToken := getAuthToken(ownerID, ownerEmail)

	// Insert a group
	var groupID string
	err := pool.QueryRow(ctx, "INSERT INTO visibility_groups (owner_id, name, visibility_level) VALUES ($1, 'Group', 'work') RETURNING id", ownerID).Scan(&groupID)
	if err != nil {
		t.Fatalf("failed to insert group: %v", err)
	}

	nonExistentEmail := "nonexistent@example.com"
	req := models.AddMemberRequest{
		Email: &nonExistentEmail,
	}

	rr := makeRequest(router, "POST", "/api/my/groups/"+groupID+"/members", req, ownerToken)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", rr.Code)
	}

	errResp := parseErrorResponse(t, rr)
	if errResp.Error != "user not found" {
		t.Errorf("unexpected error message: %s", errResp.Error)
	}
}

func TestGroupsAddMember_MissingEmailAndUserID(t *testing.T) {
	pool := setupTestDB(t)
	defer pool.Close()
	defer cleanupTestData(t, pool)

	router := NewRouter(pool)
	ctx := context.Background()

	// Create owner user
	ownerEmail := "groupowner8@example.com"
	ownerID := createTestUser(t, pool, ownerEmail, "password123", "Group Owner")
	ownerToken := getAuthToken(ownerID, ownerEmail)

	// Insert a group
	var groupID string
	err := pool.QueryRow(ctx, "INSERT INTO visibility_groups (owner_id, name, visibility_level) VALUES ($1, 'Group', 'work') RETURNING id", ownerID).Scan(&groupID)
	if err != nil {
		t.Fatalf("failed to insert group: %v", err)
	}

	req := models.AddMemberRequest{}

	rr := makeRequest(router, "POST", "/api/my/groups/"+groupID+"/members", req, ownerToken)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rr.Code)
	}

	errResp := parseErrorResponse(t, rr)
	if errResp.Error != "either email or userId is required" {
		t.Errorf("unexpected error message: %s", errResp.Error)
	}
}

func TestGroupsAddMember_NotOwner(t *testing.T) {
	pool := setupTestDB(t)
	defer pool.Close()
	defer cleanupTestData(t, pool)

	router := NewRouter(pool)
	ctx := context.Background()

	// Create two users
	ownerEmail := "groupowner9@example.com"
	ownerID := createTestUser(t, pool, ownerEmail, "password123", "Group Owner")

	otherEmail := "groupother4@example.com"
	otherID := createTestUser(t, pool, otherEmail, "password123", "Other User")
	otherToken := getAuthToken(otherID, otherEmail)

	// Create user to add
	memberEmail := "member@example.com"
	_ = createTestUser(t, pool, memberEmail, "password123", "Member")

	// Insert a group for owner
	var groupID string
	err := pool.QueryRow(ctx, "INSERT INTO visibility_groups (owner_id, name, visibility_level) VALUES ($1, 'Private Group', 'family') RETURNING id", ownerID).Scan(&groupID)
	if err != nil {
		t.Fatalf("failed to insert group: %v", err)
	}

	req := models.AddMemberRequest{
		Email: &memberEmail,
	}

	rr := makeRequest(router, "POST", "/api/my/groups/"+groupID+"/members", req, otherToken)

	if rr.Code != http.StatusForbidden {
		t.Errorf("expected status 403, got %d", rr.Code)
	}

	errResp := parseErrorResponse(t, rr)
	if errResp.Error != "you don't have access to this group" {
		t.Errorf("unexpected error message: %s", errResp.Error)
	}
}

func TestGroupsAddMember_DuplicateMember(t *testing.T) {
	pool := setupTestDB(t)
	defer pool.Close()
	defer cleanupTestData(t, pool)

	router := NewRouter(pool)
	ctx := context.Background()

	// Create owner user
	ownerEmail := "groupowner10@example.com"
	ownerID := createTestUser(t, pool, ownerEmail, "password123", "Group Owner")
	ownerToken := getAuthToken(ownerID, ownerEmail)

	// Create member user
	memberEmail := "duplicatemember@example.com"
	memberID := createTestUser(t, pool, memberEmail, "password123", "Duplicate Member")

	// Insert a group
	var groupID string
	err := pool.QueryRow(ctx, "INSERT INTO visibility_groups (owner_id, name, visibility_level) VALUES ($1, 'Group', 'family') RETURNING id", ownerID).Scan(&groupID)
	if err != nil {
		t.Fatalf("failed to insert group: %v", err)
	}

	// Add member first time
	_, err = pool.Exec(ctx, "INSERT INTO group_members (group_id, member_id, added_by) VALUES ($1, $2, $1)", groupID, memberID)
	if err != nil {
		t.Fatalf("failed to add member: %v", err)
	}

	// Try to add same member again
	req := models.AddMemberRequest{
		Email: &memberEmail,
	}

	rr := makeRequest(router, "POST", "/api/my/groups/"+groupID+"/members", req, ownerToken)

	if rr.Code != http.StatusConflict {
		t.Errorf("expected status 409, got %d", rr.Code)
	}

	errResp := parseErrorResponse(t, rr)
	if errResp.Error != "user is already a member of this group" {
		t.Errorf("unexpected error message: %s", errResp.Error)
	}
}

func TestGroupsRemoveMember_Success(t *testing.T) {
	pool := setupTestDB(t)
	defer pool.Close()
	defer cleanupTestData(t, pool)

	router := NewRouter(pool)
	ctx := context.Background()

	// Create owner user
	ownerEmail := "groupowner11@example.com"
	ownerID := createTestUser(t, pool, ownerEmail, "password123", "Group Owner")
	ownerToken := getAuthToken(ownerID, ownerEmail)

	// Create member user
	memberEmail := "membertoremove@example.com"
	memberID := createTestUser(t, pool, memberEmail, "password123", "Member to Remove")

	// Insert a group
	var groupID string
	err := pool.QueryRow(ctx, "INSERT INTO visibility_groups (owner_id, name, visibility_level) VALUES ($1, 'Group', 'friends') RETURNING id", ownerID).Scan(&groupID)
	if err != nil {
		t.Fatalf("failed to insert group: %v", err)
	}

	// Add member and get the membership ID
	var memberRecordID string
	err = pool.QueryRow(ctx, "INSERT INTO group_members (group_id, member_id, added_by) VALUES ($1, $2, $1) RETURNING id", groupID, memberID).Scan(&memberRecordID)
	if err != nil {
		t.Fatalf("failed to add member: %v", err)
	}

	rr := makeRequest(router, "DELETE", "/api/my/groups/"+groupID+"/members/"+memberRecordID, nil, ownerToken)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("expected status 204, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestGroupsRemoveMember_NotFound(t *testing.T) {
	pool := setupTestDB(t)
	defer pool.Close()
	defer cleanupTestData(t, pool)

	router := NewRouter(pool)
	ctx := context.Background()

	// Create owner user
	ownerEmail := "groupowner12@example.com"
	ownerID := createTestUser(t, pool, ownerEmail, "password123", "Group Owner")
	ownerToken := getAuthToken(ownerID, ownerEmail)

	// Insert a group
	var groupID string
	err := pool.QueryRow(ctx, "INSERT INTO visibility_groups (owner_id, name, visibility_level) VALUES ($1, 'Group', 'public') RETURNING id", ownerID).Scan(&groupID)
	if err != nil {
		t.Fatalf("failed to insert group: %v", err)
	}

	rr := makeRequest(router, "DELETE", "/api/my/groups/"+groupID+"/members/nonexistent-id", nil, ownerToken)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", rr.Code)
	}

	errResp := parseErrorResponse(t, rr)
	if errResp.Error != "member not found" {
		t.Errorf("unexpected error message: %s", errResp.Error)
	}
}

func TestGroupsRemoveMember_NotOwner(t *testing.T) {
	pool := setupTestDB(t)
	defer pool.Close()
	defer cleanupTestData(t, pool)

	router := NewRouter(pool)
	ctx := context.Background()

	// Create two users
	ownerEmail := "groupowner13@example.com"
	ownerID := createTestUser(t, pool, ownerEmail, "password123", "Group Owner")

	otherEmail := "groupother5@example.com"
	otherID := createTestUser(t, pool, otherEmail, "password123", "Other User")
	otherToken := getAuthToken(otherID, otherEmail)

	// Create member user
	memberEmail := "member5@example.com"
	memberID := createTestUser(t, pool, memberEmail, "password123", "Member")

	// Insert a group for owner
	var groupID string
	err := pool.QueryRow(ctx, "INSERT INTO visibility_groups (owner_id, name, visibility_level) VALUES ($1, 'Private Group', 'family') RETURNING id", ownerID).Scan(&groupID)
	if err != nil {
		t.Fatalf("failed to insert group: %v", err)
	}

	// Add member
	var memberRecordID string
	err = pool.QueryRow(ctx, "INSERT INTO group_members (group_id, member_id, added_by) VALUES ($1, $2, $1) RETURNING id", groupID, memberID).Scan(&memberRecordID)
	if err != nil {
		t.Fatalf("failed to add member: %v", err)
	}

	rr := makeRequest(router, "DELETE", "/api/my/groups/"+groupID+"/members/"+memberRecordID, nil, otherToken)

	if rr.Code != http.StatusForbidden {
		t.Errorf("expected status 403, got %d", rr.Code)
	}

	errResp := parseErrorResponse(t, rr)
	if errResp.Error != "you don't have access to this group" {
		t.Errorf("unexpected error message: %s", errResp.Error)
	}
}
