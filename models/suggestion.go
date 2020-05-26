package models

import "go.mongodb.org/mongo-driver/bson/primitive"

type Suggestion struct {
	ID     primitive.ObjectID `bson:"_id,omitempty"`
	Email  string             `bson:"email"`
	Detail string             `bson:"detail"`
	Date   string             `bson:"date"`
}
