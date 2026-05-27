package templates

import (
	"errors"
	"time"

	"github.com/google/uuid"
	db "github.com/reddybashanaguru/opsowl/internal/db/generated"
)


var (
	ErrNotFound     = errors.New("template not found")
	ErrNameRequired = errors.New("template name is required")
)

type TaskStub struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Priority    string `json:"priority"`
	SortOrder   int32  `json:"sort_order"`
}

type Template struct {
	ID                  uuid.UUID  `json:"id"`
	Name                string     `json:"name"`
	DescriptionSkeleton string     `json:"description_skeleton"`
	Priority            string     `json:"priority"`
	Tags                []string   `json:"tags"`
	IsActive            bool       `json:"is_active"`
	Tasks               []TaskStub `json:"tasks"`
	CreatedAt           time.Time  `json:"created_at"`
	UpdatedAt           time.Time  `json:"updated_at"`
}

type CreateTemplateRequest struct {
	Name                string     `json:"name"`
	DescriptionSkeleton string     `json:"description_skeleton"`
	Priority            string     `json:"priority"`
	Tags                []string   `json:"tags"`
	IsActive            *bool      `json:"is_active"`
	Tasks               []TaskStub `json:"tasks"`
}

type UpdateTemplateRequest = CreateTemplateRequest

func templateFromRow(t db.CaseTemplate, tasks []db.CaseTemplateTask) *Template {
	stubs := make([]TaskStub, len(tasks))
	for i, tk := range tasks {
		stubs[i] = TaskStub{
			Title:       tk.Title,
			Description: tk.Description,
			Priority:    string(tk.Priority),
			SortOrder:   tk.SortOrder,
		}
	}
	return &Template{
		ID:                  t.ID,
		Name:                t.Name,
		DescriptionSkeleton: t.DescriptionSkeleton,
		Priority:            string(t.Priority),
		Tags:                t.Tags,
		IsActive:            t.IsActive,
		Tasks:               stubs,
		CreatedAt:           t.CreatedAt,
		UpdatedAt:           t.UpdatedAt,
	}
}
