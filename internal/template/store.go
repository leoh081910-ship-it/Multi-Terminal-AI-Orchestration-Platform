package template

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type Store struct {
	db *sql.DB
}

func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

type Input struct {
	Name        string `json:"name"`
	Type        string `json:"type"`        // string, number, boolean, file
	Required    bool   `json:"required"`
	Default     any    `json:"default,omitempty"`
	Description string `json:"description,omitempty"`
}

type Output struct {
	Name        string `json:"name"`
	Type        string `json:"type"` // file, artifact, log
	Description string `json:"description,omitempty"`
}

// TaskTemplate represents a reusable task configuration (SQL storage version).
type TaskTemplateSQL struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	ProjectID   string    `json:"project_id"`
	OwnerAgent  string    `json:"owner_agent"`
	TaskType    string    `json:"task_type"`
	Priority    int       `json:"priority"`
	Command     string    `json:"command"`
	WorkDir     string    `json:"work_dir"`
	TimeoutSec  int       `json:"timeout_sec"`
	Tags        []string  `json:"tags"`        // JSON
	Inputs      []Input   `json:"inputs"`      // JSON
	Outputs     []Output  `json:"outputs"`     // JSON
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	CreatedBy   string    `json:"created_by"`
}

func (s *Store) InitTable(ctx context.Context) error {
	query := `
	CREATE TABLE IF NOT EXISTS task_templates (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		description TEXT,
		project_id TEXT,
		owner_agent TEXT DEFAULT 'Claude',
		task_type TEXT DEFAULT 'task',
		priority INTEGER DEFAULT 3,
		command TEXT,
		work_dir TEXT,
		timeout_sec INTEGER DEFAULT 1800,
		tags TEXT,
		inputs TEXT,
		outputs TEXT,
		created_at DATETIME NOT NULL,
		updated_at DATETIME NOT NULL,
		created_by TEXT
	);
	CREATE INDEX IF NOT EXISTS idx_task_templates_project_id ON task_templates(project_id);
	`
	_, err := s.db.ExecContext(ctx, query)
	return err
}

func (s *Store) Create(ctx context.Context, tpl *TaskTemplateSQL) error {
	if tpl.ID == "" {
		tpl.ID = uuid.New().String()
	}
	tpl.CreatedAt = time.Now()
	tpl.UpdatedAt = tpl.CreatedAt

	tagsJSON, err := json.Marshal(tpl.Tags)
	if err != nil {
		return err
	}
	inputsJSON, err := json.Marshal(tpl.Inputs)
	if err != nil {
		return err
	}
	outputsJSON, err := json.Marshal(tpl.Outputs)
	if err != nil {
		return err
	}

	_, err = s.db.ExecContext(ctx, `
		INSERT INTO task_templates (
			id, name, description, project_id, owner_agent, task_type, priority,
			command, work_dir, timeout_sec, tags, inputs, outputs, created_at, updated_at, created_by
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		tpl.ID, tpl.Name, tpl.Description, tpl.ProjectID, tpl.OwnerAgent, tpl.TaskType, tpl.Priority,
		tpl.Command, tpl.WorkDir, tpl.TimeoutSec, string(tagsJSON), string(inputsJSON), string(outputsJSON),
		tpl.CreatedAt, tpl.UpdatedAt, tpl.CreatedBy,
	)
	return err
}

func (s *Store) Get(ctx context.Context, id string) (*TaskTemplateSQL, error) {
	var tpl TaskTemplateSQL
	var tagsStr, inputsStr, outputsStr sql.NullString

	err := s.db.QueryRowContext(ctx, `
		SELECT id, name, description, project_id, owner_agent, task_type, priority,
			command, work_dir, timeout_sec, tags, inputs, outputs, created_at, updated_at, created_by
		FROM task_templates WHERE id = ?
	`, id).Scan(
		&tpl.ID, &tpl.Name, &tpl.Description, &tpl.ProjectID, &tpl.OwnerAgent, &tpl.TaskType, &tpl.Priority,
		&tpl.Command, &tpl.WorkDir, &tpl.TimeoutSec, &tagsStr, &inputsStr, &outputsStr,
		&tpl.CreatedAt, &tpl.UpdatedAt, &tpl.CreatedBy,
	)
	if err != nil {
		return nil, err
	}

	if tagsStr.Valid && len(tagsStr.String) > 0 {
		json.Unmarshal([]byte(tagsStr.String), &tpl.Tags)
	}
	if inputsStr.Valid && len(inputsStr.String) > 0 {
		json.Unmarshal([]byte(inputsStr.String), &tpl.Inputs)
	}
	if outputsStr.Valid && len(outputsStr.String) > 0 {
		json.Unmarshal([]byte(outputsStr.String), &tpl.Outputs)
	}
	return &tpl, nil
}

func (s *Store) List(ctx context.Context, projectID string, limit, offset int) ([]TaskTemplateSQL, int, error) {
	countQuery := `SELECT COUNT(*) FROM task_templates`
	countArgs := []interface{}{}
	if projectID != "" {
		countQuery += " WHERE project_id = ?"
		countArgs = append(countArgs, projectID)
	}
	var total int
	if err := s.db.QueryRowContext(ctx, countQuery, countArgs...).Scan(&total); err != nil {
		return nil, 0, err
	}

	query := `SELECT id, name, description, project_id, owner_agent, task_type, priority,
		command, work_dir, timeout_sec, tags, inputs, outputs, created_at, updated_at, created_by
		FROM task_templates`
	args := []interface{}{}
	if projectID != "" {
		query += " WHERE project_id = ?"
		args = append(args, projectID)
	}
	query += " ORDER BY created_at DESC LIMIT ? OFFSET ?"
	args = append(args, limit, offset)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var tpls []TaskTemplateSQL
	for rows.Next() {
		var tpl TaskTemplateSQL
		var tagsStr, inputsStr, outputsStr sql.NullString
		if err := rows.Scan(
			&tpl.ID, &tpl.Name, &tpl.Description, &tpl.ProjectID, &tpl.OwnerAgent, &tpl.TaskType, &tpl.Priority,
			&tpl.Command, &tpl.WorkDir, &tpl.TimeoutSec, &tagsStr, &inputsStr, &outputsStr,
			&tpl.CreatedAt, &tpl.UpdatedAt, &tpl.CreatedBy,
		); err != nil {
			return nil, 0, err
		}
		if tagsStr.Valid && len(tagsStr.String) > 0 {
			json.Unmarshal([]byte(tagsStr.String), &tpl.Tags)
		}
		if inputsStr.Valid && len(inputsStr.String) > 0 {
			json.Unmarshal([]byte(inputsStr.String), &tpl.Inputs)
		}
		if outputsStr.Valid && len(outputsStr.String) > 0 {
			json.Unmarshal([]byte(outputsStr.String), &tpl.Outputs)
		}
		tpls = append(tpls, tpl)
	}
	return tpls, total, nil
}

func (s *Store) Update(ctx context.Context, id string, updates map[string]interface{}) error {
	updates["updated_at"] = time.Now()
	cols := make([]string, 0, len(updates))
	args := make([]interface{}, 0, len(updates))
	for k, v := range updates {
		cols = append(cols, k+" = ?")
		args = append(args, v)
	}
	args = append(args, id)
	query := "UPDATE task_templates SET " + join(cols, ", ") + " WHERE id = ?"
	_, err := s.db.ExecContext(ctx, query, args...)
	return err
}

func (s *Store) Delete(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM task_templates WHERE id = ?", id)
	return err
}

func join(strs []string, sep string) string {
	if len(strs) == 0 {
		return ""
	}
	res := strs[0]
	for i := 1; i < len(strs); i++ {
		res += sep + strs[i]
	}
	return res
}