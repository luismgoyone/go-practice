package model

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Task is stored in the "tasks" collection.
type Task struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	ProjectID primitive.ObjectID `bson:"project_id" json:"project_id"`
	Title     string             `bson:"title" json:"title"`
	Status    string             `bson:"status" json:"status"` // todo | doing | done
	CreatedAt time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt time.Time          `bson:"updated_at" json:"updated_at"`
}
