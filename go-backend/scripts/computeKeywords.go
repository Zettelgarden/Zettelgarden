package main

import (
	"database/sql"
	"fmt"
	"go-backend/models"
	"log"
	"os"

	_ "github.com/lib/pq"
)

// TODO: This script uses llms.ComputeCardKeywords which no longer exists.
// The llms package was merged into services. This script needs to be updated
// to use the correct services package or removed if obsolete.

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

func queryCards(db *sql.DB, userID int) ([]models.Card, error) {

	cards := []models.Card{}
	query := `

    SELECT 
		id, card_id, user_id, title, body, link, created_at, updated_at 
    FROM 
        cards
    WHERE
		user_id = $1 AND is_deleted = FALSE`

	// Add condition for searchTerm
	var rows *sql.Rows
	var err error

	rows, err = db.Query(query, userID)
	if err != nil {
		log.Printf("err %v", err)
		return cards, err
	}

	for rows.Next() {
		var card models.Card
		if err := rows.Scan(
			&card.ID,
			&card.CardID,
			&card.UserID,
			&card.Title,
			&card.Body,
			&card.Link,
			&card.CreatedAt,
			&card.UpdatedAt,
		); err != nil {
			log.Printf("err %v", err)
			return cards, err
		}
		cards = append(cards, card)
	}
	return cards, nil
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

	rows, _ := db.Query("SELECT id FROM users")
	for rows.Next() {
		var userID int
		if err := rows.Scan(&userID); err != nil {
			log.Fatalf("something is wrong: %v", err.Error())
			return
		}

		log.Printf("user %v", userID)
		cards, _ := queryCards(db, userID)
		log.Printf("cards %v", len(cards))
		for _, card := range cards {
			log.Printf("%v %v - %v", card.ID, card.CardID, card.Title)
			// TODO: ComputeCardKeywords no longer exists - this script is obsolete
		}
	}

}
