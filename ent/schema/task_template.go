package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"time"
)

// TaskTemplate holds the schema definition for the TaskTemplate entity.
type TaskTemplate struct {
	ent.Schema
}

func (TaskTemplate) Fields() []ent.Field {
	return []ent.Field{
		field.String("id").MaxLen(36).Unique(),
		field.String("name").MaxLen(255).NotEmpty(),
		field.Text("description").Optional(),
		field.String("project_id").MaxLen(36).Optional(),
		field.String("owner_agent").MaxLen(100).Default("Claude"),
		field.String("task_type").MaxLen(100).Default("task"),
		field.Int("priority").Default(3),
		field.Text("command").Optional(),
		field.String("work_dir").MaxLen(500).Optional(),
		field.Int("timeout_sec").Default(1800),
		field.JSON("tags", []string{}).Optional(),
		field.JSON("inputs", []map[string]interface{}{}).Optional(),
		field.JSON("outputs", []map[string]interface{}{}).Optional(),
		field.String("created_by").MaxLen(100).Optional(),
		field.Time("created_at").Default(time.Now),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
	}
}

func (TaskTemplate) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("project_id"),
		index.Fields("name"),
	}
}
