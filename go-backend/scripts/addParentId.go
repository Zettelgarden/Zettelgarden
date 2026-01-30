//go:build ignore
// +build ignore

package main

import (
	"database/sql"
	"fmt"
	"go-backend/models"
	"log"
	"os"

	_ "github.com/lib/pq"
)

func ConnectToDatabase(dbConfig models.DatabaseConfig) (*sql.DB, error) {
	psqlInfo := fmt.Sprintf("host=%v port=%v user=%v "+
		"password=%v dbname=%v sslmode=disable",
		dbConfig.Host, dbConfig.Port, dbConfig.User, dbConfig.Password, dbConfig.DatabaseName)

	db, err := sql.Open("postgres", psqlInfo)
	if err != nil {
		log.Fatalf("Unable to connect to the database: %v\n", err)
	}
	if err := db.Ping(); err != nil {
		log.Fatal(err)
	}
	return db, err
}

func getParentIdAlternating(cardID string) string {
	parts := []string{}
	currentPart := ""

	for _, char := range cardID {
		if char == '/' || char == '.' {
			parts = append(parts, currentPart)
			currentPart = ""
		} else {
			currentPart += string(char)
		}
	}

	if currentPart != "" {
		parts = append(parts, currentPart)
	}

	if len(parts) == 1 {
		return cardID
	}

	parentID := ""
	for i := 0; i < len(parts)-1; i++ {
		parentID += parts[i]
		if i < len(parts)-2 {
			if i%2 == 0 {
				parentID += "/"
			} else {
				parentID += "."
			}
		}
	}

	return parentID
}

func main() {
	dbConfig := models.DatabaseConfig{}
	dbConfig.Host = os.Getenv("DB_HOST")
	dbConfig.Port = os.Getenv("DB_PORT")
	dbConfig.User = os.Getenv("DB_USER")
	dbConfig.Password = os.Getenv("DB_PASS")
	dbConfig.DatabaseName = os.Getenv("DB_NAME")

	db, err := ConnectToDatabase(dbConfig)
	if err != nil {
		log.Fatalf("unable to connect to db: %v", err.Error())
		return
	}
	rows, _ := db.Query("SELECT id, card_id FROM cards WHERE user_id = 1 AND is_deleted = FALSE")
	for rows.Next() {
		var id int
		var cardID string
		if err := rows.Scan(&id, &cardID); err != nil {
			log.Fatalf("something is wrong: %v", err.Error())
			return
		}

		parent := getParentIdAlternating(cardID)
		query := `UPDATE cards c1
SET parent_id = (
    SELECT c2.id
    FROM cards c2
    WHERE c2.card_id = $1
) WHERE c1.id = $2`
		_, err = db.Exec(query, parent, id)
		if err != nil {
			log.Printf("err %v", err)
			continue
		}

	}
}
