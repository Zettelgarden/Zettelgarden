package handlers

import (
	"encoding/json"
	"fmt"
	"go-backend/models"
	"go-backend/services"
	"log"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
)

func (s *Handler) GetTags(userID int) ([]models.Tag, error) {
	return services.QueryTags(s.GetDB(), userID)
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

	// Tag INSERT + sync_log emit in one transaction (bead Zettelgarden-5ry).
	tx, err := s.BeginTx()
	if err != nil {
		http.Error(w, "unable to start transaction", http.StatusInternalServerError)
		return
	}
	tag, err := services.CreateTag(tx, userID, tagData)
	if err != nil {
		s.rollbackTx(tx)
		http.Error(w, fmt.Sprintf("Error creating tag: %v", err), http.StatusInternalServerError)
		return
	}
	if err := s.commitTx(tx); err != nil {
		log.Printf("create tag: commit: %v", err)
		http.Error(w, "unable to commit", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tag)
}

func (s *Handler) AddTagToTask(userID int, tagName string, taskPK int) error {
	return services.AddTagToTask(s.GetDB(), userID, tagName, taskPK)
}

func (s *Handler) RemoveAllTagsFromTask(userID, taskPK int) error {
	return services.RemoveAllTagsFromTask(s.GetDB(), userID, taskPK)
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
		_, err := services.CreateTag(s.GetDB(), userID, params)
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
	return services.QueryTagsForTask(s.GetDB(), userID, taskPK)
}

// DeleteTag soft-deletes a tag on db (the caller's transaction).
func (s *Handler) DeleteTag(db models.Database, userID, id int) error {
	return services.DeleteTag(db, userID, id)
}

func (s *Handler) DeleteTagRoute(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("current_user").(int)
	id, err := strconv.Atoi(mux.Vars(r)["id"])
	if err != nil {
		http.Error(w, "Invalid id", http.StatusBadRequest)
		return
	}
	// Tag DELETE + sync_log emit in one transaction (bead Zettelgarden-5ry).
	tx, err := s.BeginTx()
	if err != nil {
		http.Error(w, "unable to start transaction", http.StatusInternalServerError)
		return
	}
	err = s.DeleteTag(tx, userID, id)
	if err != nil {
		s.rollbackTx(tx)
		http.Error(w, fmt.Sprintf("unable to delete tag: %v", err.Error()), http.StatusBadRequest)
		return
	}
	if err := s.commitTx(tx); err != nil {
		log.Printf("delete tag: commit: %v", err)
		http.Error(w, "unable to commit", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
