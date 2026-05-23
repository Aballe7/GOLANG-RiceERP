package middleware

import (
	"sync"

	"ricemill/app/models"
)

// SessionStore is an in-memory, single-user session for the desktop app.
// No tokens or cookies needed — only one user is logged in at a time.
type SessionStore struct {
	mu          sync.RWMutex
	currentUser *models.User
}

var Store = &SessionStore{}

func (s *SessionStore) Login(u *models.User) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.currentUser = u
}

func (s *SessionStore) Logout() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.currentUser = nil
}

func (s *SessionStore) CurrentUser() *models.User {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.currentUser
}

func (s *SessionStore) IsLoggedIn() bool {
	return s.CurrentUser() != nil
}

func (s *SessionStore) IsAdmin() bool {
	u := s.CurrentUser()
	return u != nil && u.Role == "Admin"
}

func (s *SessionStore) CanAccess(module string) bool {
	u := s.CurrentUser()
	if u == nil {
		return false
	}
	return u.CanAccess(module)
}

func (s *SessionStore) UserID() *uint {
	u := s.CurrentUser()
	if u == nil {
		return nil
	}
	return &u.ID
}
