package main

import (
	"database/sql"
	"fmt"
	"github.com/gin-contrib/cors"
	"github.com/joho/godotenv"
	"log"
	"net/http"
	"os"
	"strconv"

	_ "github.com/denisenkom/go-mssqldb"
	"github.com/gin-gonic/gin"
)

var db *sql.DB

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	server := os.Getenv("DB_SERVER")
	port, _ := strconv.Atoi(os.Getenv("DB_PORT"))
	user := os.Getenv("DB_USER")
	password := os.Getenv("DB_PASSWORD")
	database := os.Getenv("DB_NAME")

	connString := fmt.Sprintf("server=%s;user id=%s;password=%s;port=%d;database=%s",
		server, user, password, port, database)

	db, err = sql.Open("sqlserver", connString)
	if err != nil {
		log.Fatal("Error creating connection pool: ", err.Error())
	}

	err = db.Ping()
	if err != nil {
		log.Fatal("Error connecting to database: ", err.Error())
	}

	fmt.Println("Connected to SQL Server!")

	mode := os.Getenv("GIN_MODE")
	switch mode {
	case gin.DebugMode, gin.ReleaseMode, gin.TestMode:
		gin.SetMode(mode)
	default:
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.Default()
	router.Use(cors.Default())

	router.GET("/emails", getEmails) // Read all
	//router.GET("/emails/:id", getEmail)      // Read one
	router.POST("/emails", createEmail) // Create
	// router.PUT("/emails/:id", updateEmail)   // Update
	router.DELETE("/emails/:id", deleteEmail) // Delete

	serverPort := os.Getenv("PORT")
	if serverPort == "" {
		serverPort = "5050"
	}

	log.Printf("Starting server on port %s...\n", serverPort)
	router.Run(":" + serverPort)
}

type Email struct {
	ID        int     `json:"id"`
	Conta     string  `json:"conta"`
	EmpresaId int     `json:"empresa_id"`
	CrdId     *string `json:"crd_id"`
	TipoConta string  `json:"tipo_conta"`
}

func getEmails(c *gin.Context) {
	rows, err := db.Query("SELECT id, conta, empresa_id, crd_id, tipo_conta FROM contas_email")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var emails []Email
	for rows.Next() {
		var email Email
		if err := rows.Scan(&email.ID, &email.Conta, &email.EmpresaId, &email.CrdId, &email.TipoConta); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		emails = append(emails, email)
	}

	c.JSON(http.StatusOK, emails)
}

func getEmail(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
		return
	}

	var email Email
	err = db.QueryRow("SELECT id FROM contas_email WHERE id = @p1", id).Scan(&email.ID)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Email not found"})
		return
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, email)
}

func createEmail(c *gin.Context) {
	var email Email
	if err := c.ShouldBindJSON(&email); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON"})
		return
	}

	var query string
	var result sql.Result
	var err error

	if email.CrdId != nil {
		query = "INSERT INTO contas_email (conta, empresa_id, crd_id, tipo_conta) VALUES (@p1, @p2, @p3, @p4)"
		result, err = db.Exec(query, email.Conta, email.EmpresaId, *email.CrdId, email.TipoConta)
	} else {
		query = "INSERT INTO contas_email (conta, empresa_id, tipo_conta) VALUES (@p1, @p2, @p3)"
		result, err = db.Exec(query, email.Conta, email.EmpresaId, email.TipoConta)
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	id, _ := result.LastInsertId()
	email.ID = int(id)
	c.JSON(http.StatusCreated, email)
}

func updateEmail(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
		return
	}

	var email Email
	if err := c.ShouldBindJSON(&email); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON"})
		return
	}

	_, err = db.Exec("UPDATE contas_email SET id = @p1 WHERE id = @p2", email.ID, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	email.ID = id
	c.JSON(http.StatusOK, email)
}

// Delete an email by ID
func deleteEmail(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
		return
	}

	_, err = db.Exec("DELETE FROM contas_email WHERE id = @p1", id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Email deleted"})
}
