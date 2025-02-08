package db

import (
	"context"
	"errors"
	"log"
	"strings"
	"time"

	"github.com/dartt0n/realtime-chat-backend/models"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// verify MongoDB implements database interface in compile time
var _ Database = (*MongoDB)(nil)

const (
	USER_COLL = "users"
)

type MongoDB struct {
	client *mongo.Client
	db     string
}

func InitMongo(conn string, db string) error {
	client, err := mongo.Connect(options.Client().ApplyURI(conn))
	if err != nil {
		return err
	}

	_globalDB = &MongoDB{client: client, db: db}
	return nil
}

func (m *MongoDB) CreateUser(user CreateUser) (models.User, error) {
	dbuser := models.User{
		CreatedAt: time.Now().Unix(),
		UpdatedAt: time.Now().Unix(),
		DeletedAt: 0,
		Email:     user.Email,
		Password:  user.PwdHash,
	}

	result, err := m.client.Database(m.db).Collection(USER_COLL).InsertOne(context.TODO(), dbuser)
	if err != nil {
		log.Printf("failed to insert item into database: %v", err)
		return models.User{}, err
	}
	log.Printf("database insertion result: %v", result.InsertedID.(bson.ObjectID))

	dbuser.ID, err = models.ParseUserID(result.InsertedID.(bson.ObjectID).Hex())
	if err != nil {
		return models.User{}, err
	}

	log.Printf("datbabase user sucessfully created: %s", dbuser.ID.String())
	return dbuser, nil

}

func (m *MongoDB) EmailExists(email string) (bool, error) {
	user, err := m.FindByEmail(email)
	if err != nil && errors.Is(err, mongo.ErrNoDocuments) {
		return false, nil
	}

	if err != nil {
		return false, err
	}

	return user.ID.String() != "", nil
}

func (m *MongoDB) FindByEmail(email string) (user models.User, err error) {
	email = strings.ToLower(strings.Trim(email, " "))
	err = m.client.Database(m.db).Collection(USER_COLL).FindOne(context.TODO(), bson.D{{Key: "email", Value: email}}).Decode(&user)
	return user, err
}

func (m *MongoDB) GetUser(id models.UserID) (user models.User, err error) {
	m.client.Database(m.db).Collection(USER_COLL).FindOne(context.TODO(), bson.D{{Key: "_id", Value: id.String()}}).Decode(&user)
	return user, err
}
