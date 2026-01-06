package handlers

import (
	"database/sql"
	"net/http"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

type SetupHandler struct {
	db *sql.DB
}

func NewSetupHandler(db *sql.DB) *SetupHandler {
	return &SetupHandler{db: db}
}

// CheckDefaultPassword checks if the current user is using the default password
func (h *SetupHandler) CheckDefaultPassword(c *gin.Context) {
	// Get username from Gin context (set by auth middleware)
	username, exists := c.Get("username")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	var passwordHash string
	err := h.db.QueryRow(`
		SELECT password_hash FROM users WHERE username = $1
	`, username).Scan(&passwordHash)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check password"})
		return
	}

	// Check if the current password hash matches default password "admin123"
	err = bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte("admin123"))
	isDefaultPassword := (err == nil)

	c.JSON(http.StatusOK, gin.H{
		"is_default_password": isDefaultPassword,
	})
}

type SetupTask struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Completed   bool   `json:"completed"`
	Priority    string `json:"priority"` // critical, recommended, optional
	ActionURL   string `json:"action_url,omitempty"`
	ActionLabel string `json:"action_label,omitempty"`
}

type SetupStatus struct {
	Tasks               []SetupTask `json:"tasks"`
	AllCriticalComplete bool        `json:"all_critical_complete"`
}

// GetSetupStatus returns the status of initial setup tasks
func (h *SetupHandler) GetSetupStatus(c *gin.Context) {
	username, exists := c.Get("username")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	var tasks []SetupTask

	// Task 1: Change default password
	var isDefaultPassword bool
	var passwordHash string
	err := h.db.QueryRow(`SELECT password_hash FROM users WHERE username = $1`, username).Scan(&passwordHash)
	if err == nil {
		err = bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte("admin123"))
		isDefaultPassword = (err == nil)
	}

	tasks = append(tasks, SetupTask{
		Title:       "Change Default Password",
		Description: "You are using the default password. Please change it to secure your account.",
		Completed:   !isDefaultPassword,
		Priority:    "critical",
		ActionURL:   "#/profile/security",
		ActionLabel: "Change Password",
	})

	// Task 2: Create first repository
	var repoCount int
	h.db.QueryRow(`SELECT COUNT(*) FROM repositories`).Scan(&repoCount)

	tasks = append(tasks, SetupTask{
		Title:       "Create Your First Repository",
		Description: "Set up a repository to start managing your artifacts.",
		Completed:   repoCount > 0,
		Priority:    "recommended",
		ActionURL:   "#/repositories",
		ActionLabel: "Create Repository",
	})

	// Check if all critical tasks are complete
	allCriticalComplete := true
	for _, task := range tasks {
		if task.Priority == "critical" && !task.Completed {
			allCriticalComplete = false
			break
		}
	}

	status := SetupStatus{
		Tasks:               tasks,
		AllCriticalComplete: allCriticalComplete,
	}

	c.JSON(http.StatusOK, status)
}
