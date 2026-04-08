package models

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error string `json:"error"`
}

// User represents a registered user
type User struct {
	ID           string `json:"id"`
	Email        string `json:"email"`
	Name         string `json:"name"`
	IsPublic     bool   `json:"isPublic"`
	PasswordHash string `json:"-"` // never expose in JSON
	CreatedAt    string `json:"createdAt,omitempty"`
	UpdatedAt    string `json:"updatedAt,omitempty"`
}

// AuthRequest for login/register
type RegisterRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Name     string `json:"name"`
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// AuthResponse returned after successful auth
type AuthResponse struct {
	Token string `json:"token"`
	User  User   `json:"user"`
}

// Schedule represents user's availability (replaces AvailabilityRule)
type Schedule struct {
	ID        string   `json:"id"`
	UserID    string   `json:"userId"`
	Type      string   `json:"type"`
	DayOfWeek *int32   `json:"dayOfWeek,omitempty"`
	Date      *string  `json:"date,omitempty"`
	StartTime string   `json:"startTime"`
	EndTime   string   `json:"endTime"`
	IsBlocked bool     `json:"isBlocked"`
	GroupIDs  []string `json:"groupIds,omitempty"`
	CreatedAt string   `json:"createdAt,omitempty"`
}

type CreateScheduleRequest struct {
	Type      string   `json:"type"`
	DayOfWeek *int32   `json:"dayOfWeek,omitempty"`
	Date      *string  `json:"date,omitempty"`
	StartTime string   `json:"startTime"`
	EndTime   string   `json:"endTime"`
	IsBlocked bool     `json:"isBlocked"`
	GroupIDs  []string `json:"groupIds,omitempty"`
}

// VisibilityGroup for access control
type VisibilityGroup struct {
	ID              string `json:"id"`
	OwnerID         string `json:"ownerId"`
	Name            string `json:"name"`
	VisibilityLevel string `json:"visibilityLevel"`
	CreatedAt       string `json:"createdAt,omitempty"`
}

// UpdateUserRequest for updating current user profile
type UpdateUserRequest struct {
	IsPublic *bool   `json:"isPublic,omitempty"`
	Name     *string `json:"name,omitempty"`
}

type AddMemberRequest struct {
	Email  *string `json:"email,omitempty"`
	UserID *string `json:"userId,omitempty"`
}

// GroupMember with user info
type GroupMember struct {
	ID      string `json:"id"`
	GroupID string `json:"groupId"`
	Member  User   `json:"member"`
	AddedBy string `json:"addedBy"`
	AddedAt string `json:"addedAt"`
}

// Booking with user info
type Booking struct {
	ID          string  `json:"id"`
	ScheduleID  string  `json:"scheduleId"`
	Booker      User    `json:"booker"`
	Owner       User    `json:"owner"`
	Date        string  `json:"date"`
	StartTime   string  `json:"startTime"`
	EndTime     string  `json:"endTime"`
	Status      string  `json:"status"`
	CreatedAt   string  `json:"createdAt,omitempty"`
	CancelledAt *string `json:"cancelledAt,omitempty"`
}

type CreateBookingRequest struct {
	OwnerID       string `json:"ownerId"`
	ScheduleID    string `json:"scheduleId"`
	SlotStartTime string `json:"slotStartTime"`
	SlotDate      string `json:"slotDate"`
}

// CreateBooking for legacy compatibility (test support)
type CreateBooking struct {
	SlotDate      string  `json:"slotDate"`
	SlotStartTime string  `json:"slotStartTime"`
	Name          string  `json:"name"`
	Email         string  `json:"email"`
	Recurrence    *string `json:"recurrence,omitempty"`
	DayOfWeek     *int32  `json:"dayOfWeek,omitempty"`
	EndDate       *string `json:"endDate,omitempty"`
}

// Slot for public display
type Slot struct {
	ID        string `json:"id"`
	Date      string `json:"date"`
	StartTime string `json:"startTime"`
	EndTime   string `json:"endTime"`
	IsBooked  bool   `json:"isBooked"`
}
