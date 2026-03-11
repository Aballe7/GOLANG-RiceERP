package handlers

import (
	"egglayererp/app/middleware"
	"egglayererp/app/models"
	"egglayererp/app/services/auth"
	"time"
)

// LoginRequest holds login credentials from the frontend.
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// CurrentUserResponse is the user object sent to the frontend after login.
type CurrentUserResponse struct {
	ID          uint     `json:"id"`
	Username    string   `json:"username"`
	FullName    string   `json:"full_name"`
	Role        string   `json:"role"`
	Permissions []string `json:"permissions"`
}

func userToResponse(u *models.User) CurrentUserResponse {
	perms := u.GetPermissions()
	if perms == nil {
		perms = []string{} // admin gets empty array — frontend treats nil/[] as full access
	}
	return CurrentUserResponse{
		ID:          u.ID,
		Username:    u.Username,
		FullName:    u.FullName,
		Role:        u.Role,
		Permissions: perms,
	}
}

// Login authenticates a user and stores them in the session.
func Login(req LoginRequest) Response {
	user, err := auth.Login(req.Username, req.Password)
	if err != nil {
		return errResponse(err)
	}
	middleware.Store.Login(user)
	return okResponse("Login successful", userToResponse(user))
}

// Logout clears the current session.
func Logout() Response {
	middleware.Store.Logout()
	return okResponse("Logged out", nil)
}

// GetCurrentUser returns the currently logged-in user.
func GetCurrentUser() Response {
	u := middleware.Store.CurrentUser()
	if u == nil {
		return errMsg("Not logged in")
	}
	return okResponse("", userToResponse(u))
}

// --- User Management (Admin only) ---

type CreateUserRequest struct {
	Username string `json:"username"`
	FullName string `json:"full_name"`
	Role     string `json:"role"`
	Password string `json:"password"`
}

func ListUsers() Response {
	if !middleware.Store.IsAdmin() {
		return unauthorized()
	}
	users, err := auth.ListUsers()
	if err != nil {
		return errResponse(err)
	}
	// Strip password hashes before sending
	type SafeUser struct {
		ID          uint       `json:"id"`
		Username    string     `json:"username"`
		FullName    string     `json:"full_name"`
		Role        string     `json:"role"`
		IsActive    bool       `json:"is_active"`
		CreatedAt   time.Time  `json:"created_at"`
		LastLogin   *time.Time `json:"last_login"`
		Permissions *string    `json:"permissions"`
	}
	safe := make([]SafeUser, len(users))
	for i, u := range users {
		safe[i] = SafeUser{
			ID: u.ID, Username: u.Username, FullName: u.FullName,
			Role: u.Role, IsActive: u.IsActive, CreatedAt: u.CreatedAt,
			LastLogin: u.LastLogin, Permissions: u.Permissions,
		}
	}
	return okResponse("", safe)
}

func CreateUser(req CreateUserRequest) Response {
	if !middleware.Store.IsAdmin() {
		return unauthorized()
	}
	user, err := auth.CreateUser(req.Username, req.FullName, req.Role, req.Password)
	if err != nil {
		return errResponse(err)
	}
	return okResponse("User created successfully", userToResponse(user))
}

type UpdateUserRequest struct {
	ID       uint   `json:"id"`
	FullName string `json:"full_name"`
	Role     string `json:"role"`
	IsActive bool   `json:"is_active"`
}

func UpdateUser(req UpdateUserRequest) Response {
	if !middleware.Store.IsAdmin() {
		return unauthorized()
	}
	if err := auth.UpdateUser(req.ID, req.FullName, req.Role, req.IsActive); err != nil {
		return errResponse(err)
	}
	return okResponse("User updated", nil)
}

type ChangePasswordRequest struct {
	UserID      uint   `json:"user_id"`
	NewPassword string `json:"new_password"`
}

func ChangePassword(req ChangePasswordRequest) Response {
	// Admin can change any password; staff can only change their own
	current := middleware.Store.CurrentUser()
	if current == nil {
		return unauthorized()
	}
	if current.Role != "Admin" && current.ID != req.UserID {
		return forbidden("User Management")
	}
	if err := auth.ChangePassword(req.UserID, req.NewPassword); err != nil {
		return errResponse(err)
	}
	return okResponse("Password changed successfully", nil)
}

type SetPermissionsRequest struct {
	UserID  uint     `json:"user_id"`
	Modules []string `json:"modules"`
}

func SetPermissions(req SetPermissionsRequest) Response {
	if !middleware.Store.IsAdmin() {
		return unauthorized()
	}
	if err := auth.SetPermissions(req.UserID, req.Modules); err != nil {
		return errResponse(err)
	}
	return okResponse("Permissions updated", nil)
}

func DeleteUser(userID uint) Response {
	if !middleware.Store.IsAdmin() {
		return unauthorized()
	}
	if err := auth.DeleteUser(userID); err != nil {
		return errResponse(err)
	}
	return okResponse("User deactivated", nil)
}
