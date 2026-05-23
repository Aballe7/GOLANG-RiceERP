package audit

import (
	"fmt"
	"net"
	"os"
	"sync"
	"time"

	"ricemill/app/db"
	"ricemill/app/middleware"
	"ricemill/app/models"
)

// localIP caches the machine's outbound LAN IP so we only resolve it once.
var (
	localIP     string
	localIPOnce sync.Once
)

// getLocalIP returns the machine's preferred outbound LAN IPv4 address.
// It uses a UDP "dial" (no data is sent) to determine which network interface
// the OS would use to reach the outside world, then returns that interface's
// address. Falls back to "127.0.0.1" if no suitable interface is found.
func getLocalIP() string {
	localIPOnce.Do(func() {
		// Dialing UDP doesn't send any packets; it just asks the OS which
		// local address it would use to reach 8.8.8.8:80.
		conn, err := net.Dial("udp", "8.8.8.8:80")
		if err != nil {
			localIP = "127.0.0.1"
			return
		}
		defer conn.Close()
		localIP = conn.LocalAddr().(*net.UDPAddr).IP.String()
	})
	return localIP
}

// Log writes an audit entry. It is fire-and-forget — failures print to stderr
// but never block or fail the primary operation.
func Log(action, module, entityType, entityRef, description string, entityID *uint) {
	if db.DB == nil {
		fmt.Fprintf(os.Stderr, "[audit] db.DB is nil — skipping %s %s/%s\n", action, module, entityRef)
		return
	}
	entry := models.AuditLog{
		Timestamp:   time.Now(),
		Action:      action,
		Module:      module,
		EntityType:  entityType,
		EntityID:    entityID,
		EntityRef:   entityRef,
		Description: description,
		IPAddress:   getLocalIP(),
	}
	if u := middleware.Store.CurrentUser(); u != nil {
		entry.UserID = &u.ID
		entry.Username = u.Username
	}
	if result := db.DB.Create(&entry); result.Error != nil {
		fmt.Fprintf(os.Stderr, "[audit] write failed (%s %s/%s): %v\n", action, module, entityRef, result.Error)
	}
}

// uid is a convenience helper — returns a pointer to the given uint.
func uid(id uint) *uint { return &id }

// Convenience wrappers for common action verbs.

func Create(module, entityType, entityRef, desc string, entityID uint) {
	Log("CREATE", module, entityType, entityRef, desc, uid(entityID))
}

func Update(module, entityType, entityRef, desc string, entityID uint) {
	Log("UPDATE", module, entityType, entityRef, desc, uid(entityID))
}

func Delete(module, entityType, entityRef, desc string, entityID uint) {
	Log("DELETE", module, entityType, entityRef, desc, uid(entityID))
}

func Void(module, entityType, entityRef, desc string, entityID uint) {
	Log("VOID", module, entityType, entityRef, desc, uid(entityID))
}

func Cancel(module, entityType, entityRef, desc string, entityID uint) {
	Log("CANCEL", module, entityType, entityRef, desc, uid(entityID))
}

func Confirm(module, entityType, entityRef, desc string, entityID uint) {
	Log("CONFIRM", module, entityType, entityRef, desc, uid(entityID))
}

func LoginLog(username string) {
	Log("LOGIN", "Admin", "User", username, "User logged in", nil)
}

func LogoutLog(username string) {
	Log("LOGOUT", "Admin", "User", username, "User logged out", nil)
}
