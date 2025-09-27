package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"go-backend/models"
	"go-backend/services"
	"log"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
)

// written for testing, not used elsewhere
func (s *Handler) getTagByID(userID int, tagID int) (models.Tag, error) {
	var tag models.Tag
	query := `
            select id, name, user_id, color
            from tags
            where user_id = $1 and id = $2
        `
	err := s.DB.QueryRow(query, userID, tagID).Scan(
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

func (s *Handler) GetTags(userID int) ([]models.Tag, error) {
	tags := []models.Tag{}
	query := `
        SELECT 
            t.id, 
            t.name, 
            t.user_id, 
            t.color
        FROM tags t
        WHERE t.is_deleted = false AND t.user_id = $1
        GROUP BY t.id, t.name, t.user_id, t.color
    `
	var rows *sql.Rows
	var err error

	rows, err = s.DB.Query(query, userID)
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

func (s *Handler) GetTagsRoute(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("current_user").(int)

	tasks, err := s.GetTags(userID)

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return

	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tasks)

}

func (s *Handler) CreateTagRoute(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("current_user").(int)

	var tagData models.EditTagParams
	if err := json.NewDecoder(r.Body).Decode(&tagData); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if tagData.Name == "" {
		http.Error(w, "Tag name is required", http.StatusBadRequest)
		return
	}

	tag, err := services.CreateTag(s.DB, userID, tagData)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error creating tag: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tag)
}

func (s *Handler) AddTagToTask(userID int, tagName string, taskPK int) error {

	query := `
        INSERT INTO task_tags (task_pk, tag_id)
        SELECT $1, t.id
        FROM tags t
        WHERE t.name = $2 AND t.user_id = $3
	`
	_, err := s.DB.Exec(query, taskPK, tagName, userID)
	if err != nil {
		log.Printf("add tag err %v", err)
		return err
	}

	return nil
}

func (s *Handler) RemoveAllTagsFromTask(userID, taskPK int) error {
	query := `DELETE FROM task_tags WHERE task_pk = $1`
	_, err := s.DB.Exec(query, taskPK)
	return err
}

func (s *Handler) AddTagsFromTask(userID, taskPK int) error {
	task, err := s.QueryTask(userID, taskPK)
	if err != nil {
		return err
	}
	s.RemoveAllTagsFromTask(userID, taskPK)

	tags, err := services.ParseTagsFromCardBody(task.Title)
	if err != nil {
		return err
	}
	for _, tagName := range tags {
		params := models.EditTagParams{
			Name:  tagName,
			Color: "black",
		}
		_, err := services.CreateTag(s.DB, userID, params)
		if err != nil {
			return err
		}
		err = s.AddTagToTask(userID, tagName, taskPK)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *Handler) QueryTagsForTask(userID int, taskPK int) ([]models.Tag, error) {
	tags := []models.Tag{}

	query := `
        SELECT t.id, t.name, t.user_id, t.color
        FROM tags t
        JOIN task_tags tt ON t.id = tt.tag_id
        WHERE tt.task_pk = $1 AND t.user_id = $2;
        `
	var rows *sql.Rows
	var err error

	rows, err = s.DB.Query(query, taskPK, userID)
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

func (s *Handler) DeleteTag(userID, id int) error {

	_, err := s.DB.Exec(`
UPDATE tags SET is_deleted = TRUE, updated_at = NOW() WHERE id =  $1 AND user_id = $2
`, id, userID)
	return err
}

func (s *Handler) DeleteTagRoute(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("current_user").(int)
	id, err := strconv.Atoi(mux.Vars(r)["id"])
	if err != nil {
		http.Error(w, "Invalid id", http.StatusBadRequest)
		return
	}
	err = s.DeleteTag(userID, id)
	if err != nil {
		http.Error(w, fmt.Sprintf("unable to delete tag: %v", err.Error()), http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
