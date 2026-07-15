package repository

import (
	"context"
	"errors"
	"time"

	"github.com/luismgoyone/go-practice/internal/model"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type ProjectRepo struct {
	col *mongo.Collection
}

func NewProjectRepo(db *mongo.Database) *ProjectRepo {
	return &ProjectRepo{col: db.Collection("projects")}
}

func (r *ProjectRepo) Create(ctx context.Context, p model.Project) (model.Project, error) {
	p.ID = primitive.NewObjectID()
	now := time.Now().UTC()
	p.CreatedAt = now
	p.UpdatedAt = now

	_, err := r.col.InsertOne(ctx, p)
	if err != nil {
		return model.Project{}, err
	}
	return p, nil
}

func (r *ProjectRepo) List(ctx context.Context) ([]model.Project, error) {
	cursor, err := r.col.Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	projects := make([]model.Project, 0)
	if err := cursor.All(ctx, &projects); err != nil {
		return nil, err
	}
	return projects, nil
}

func (r *ProjectRepo) GetByID(ctx context.Context, id primitive.ObjectID) (model.Project, error) {
	var p model.Project
	err := r.col.FindOne(ctx, bson.M{"_id": id}).Decode(&p)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return model.Project{}, ErrNotFound
		}
		return model.Project{}, err
	}
	return p, nil
}

func (r *ProjectRepo) Update(ctx context.Context, p model.Project) (model.Project, error) {
	p.UpdatedAt = time.Now().UTC()
	res, err := r.col.UpdateOne(
		ctx,
		bson.M{"_id": p.ID},
		bson.M{"$set": bson.M{
			"name":        p.Name,
			"description": p.Description,
			"updated_at":  p.UpdatedAt,
		}},
	)
	if err != nil {
		return model.Project{}, err
	}
	if res.MatchedCount == 0 {
		return model.Project{}, ErrNotFound
	}
	return p, nil
}

func (r *ProjectRepo) Delete(ctx context.Context, id primitive.ObjectID) error {
	res, err := r.col.DeleteOne(ctx, bson.M{"_id": id})
	if err != nil {
		return err
	}
	if res.DeletedCount == 0 {
		return ErrNotFound
	}
	return nil
}
