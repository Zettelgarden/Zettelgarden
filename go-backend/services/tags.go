package services

import (
	"database/sql"
	"go-backend/models"
	"log"
	"regexp"
	"strings"
)

func QueryTagsForCard(db *sql.DB, userID int, cardPK int) ([]models.Tag, error) {
	tags := []models.Tag{}

	query := `
        SELECT t.id, t.name, t.user_id, t.color
        FROM tags t
        JOIN card_tags ct ON t.id = ct.tag_id
        WHERE ct.card_pk = $1 AND t.user_id = $2;
        `
	var rows *sql.Rows
	var err error

	rows, err = db.Query(query, cardPK, userID)
	if err != nil {
		log.Printf("err %v", err)
		return tags, err
	}
	for rows.Next() {
		var tag models.Tag
		if err := rows.Scan(
			&tag.ID,
			&tag.Name,
			&tag.UserID,
			&tag.Color,
		); err != nil {
			log.Printf("err %v", err)
			return tags, err
		}
		tags = append(tags, tag)
	}
	return tags, nil

}
func IdentifyParentTags(db *sql.DB, userID int, card models.PartialCard) ([]models.Tag, error) {
	if card.ParentID == card.ID {
		// card is its own parent, no further work needed
		return QueryTagsForCard(db, userID, card.ID)
	}
	parent, err := GetPartialCard(db, userID, card.ParentID)

	parent_tags, err := IdentifyParentTags(db, userID, parent)
	if err != nil {
		return []models.Tag{}, err
	}
	tags, err := QueryTagsForCard(db, userID, card.ID)
	if err != nil {
		return []models.Tag{}, err
	}

	results := parent_tags
	for _, tag := range tags {
		results = append(results, tag)
	}

	return results, nil
}

func ParseTagsFromCardBody(body string) ([]string, error) {
	if body == "" {
		return []string{}, nil
	}

	// Regular expression to match hashtags
	re := regexp.MustCompile(`(?:^|\s)(#[\w-]+)`)
	matches := re.FindAllString(body, -1)

	// Process matched tags
	var tags []string
	for _, match := range matches {
		tag := strings.TrimSpace(match)
		if tag != "" {
			tags = append(tags, strings.TrimPrefix(tag, "#"))
		}
	}

	return tags, nil
}
func iterateCreateTagsForCard(db *sql.DB, userID int, cardPK int, tagNames []string) error {

	for _, tagName := range tagNames {
		params := models.EditTagParams{
			Name:  tagName,
			Color: "black",
		}
		_, err := CreateTag(db, userID, params)
		if err != nil {
			return err
		}
		err = AddTagToCard(db, userID, tagName, cardPK)
		if err != nil {
			return err
		}
	}
	return nil
}
func RemoveAllTagsFromCard(db *sql.DB, userID, cardPK int) error {
	query := `DELETE FROM card_tags WHERE card_pk = $1`
	_, err := db.Exec(query, cardPK)
	return err
}
func AddTagsFromCard(db *sql.DB, userID, cardPK int) error {
	card, err := GetFullCard(db, userID, cardPK)
	if err != nil {
		return err
	}
	RemoveAllTagsFromCard(db, userID, cardPK)
	tags, err := ParseTagsFromCardBody(card.Body)
	if err != nil {
		return err
	}
	err = iterateCreateTagsForCard(db, userID, cardPK, tags)
	if err != nil {
		return err
	}
	if card.ParentID == card.ID {
		// card is its own parent, no need to go on
		return nil
	}
	parent_tags, err := IdentifyParentTags(db, userID, models.ConvertCardToPartialCard(card))
	for _, tag := range parent_tags {
		if contains(tags, tag.Name) {
			log.Printf("skip")
			continue
		}
		err = AddTagToCard(db, userID, tag.Name, cardPK)
		if err != nil {
			return err
		}
	}

	return nil
}
func contains[T comparable](collection []T, target T) bool {
	for _, v := range collection {
		if v == target {
			return true
		}
	}
	return false
}
func AddTagToCard(db *sql.DB, userID int, tagName string, cardPK int) error {
	var count int
	countQuery := `
        SELECT COUNT(*)
        FROM card_tags ct
        JOIN tags t ON ct.tag_id = t.id
        WHERE t.name = $1 AND ct.card_pk = $2 AND t.user_id = $3;
        `
	_ = db.QueryRow(countQuery, tagName, cardPK, userID).Scan(&count)
	if count > 0 {
		log.Printf("?")
		return nil
	}
	query := `
        INSERT INTO card_tags (card_pk, tag_id)
        SELECT $1, t.id
        FROM tags t
        WHERE t.name = $2 AND t.user_id = $3
	`
	_, err := db.Exec(query, cardPK, tagName, userID)
	if err != nil {
		log.Printf("add tag err %v", err)
		return err
	}

	return nil
}

func CreateTag(db *sql.DB, userID int, tagData models.EditTagParams) (models.Tag, error) {

	_, err := GetTagMaybeDeleted(db, userID, tagData.Name)
	if err == nil {
		log.Printf("tag exists, going to edit it instead")
		return EditTag(db, userID, tagData.Name, tagData)
	}

	query := `INSERT INTO tags (name, color, user_id, created_at, updated_at) VALUES ($1, $2, $3, NOW(), NOW())`
	_, err = db.Exec(query, tagData.Name, tagData.Color, userID)
	if err != nil {
		log.Printf("create tag err %v", err)
		return models.Tag{}, nil
	}
	tag, err := GetTag(db, userID, tagData.Name)

	return tag, nil
}

func EditTag(db *sql.DB, userID int, tagName string, tagData models.EditTagParams) (models.Tag, error) {

	query := `UPDATE tags SET name = $1, color = $2, is_deleted = FALSE WHERE user_id = $3 AND name = $4`
	_, err := db.Exec(query, tagData.Name, tagData.Color, userID, tagName)
	if err != nil {
		log.Printf("update tag err %v", err)
		return models.Tag{}, nil
	}
	tag, err := GetTag(db, userID, tagData.Name)
	if err != nil {
		log.Printf("update tag get err %v", err)
		return models.Tag{}, nil
	}
	return tag, nil
}
func GetTagMaybeDeleted(db *sql.DB, userID int, tagName string) (models.Tag, error) {

	var tag models.Tag
	query := `
            select id, name, user_id, color
            from tags
            where user_id = $1 and name = $2
        `
	err := db.QueryRow(query, userID, tagName).Scan(
		&tag.ID,
		&tag.Name,
		&tag.UserID,
		&tag.Color,
	)
	if err != nil {
		log.Printf("err %v", err)
		return models.Tag{}, err
	}
	return tag, nil
}

func GetTag(db *sql.DB, userID int, tagName string) (models.Tag, error) {
	var tag models.Tag
	query := `
            select id, name, user_id, color
            from tags
            where is_deleted = FALSE AND user_id = $1 and name = $2
        `
	err := db.QueryRow(query, userID, tagName).Scan(
		&tag.ID,
		&tag.Name,
		&tag.UserID,
		&tag.Color,
	)
	if err != nil {
		log.Printf("err %v", err)
		return models.Tag{}, err
	}
	return tag, nil

}
