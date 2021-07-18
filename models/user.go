package models

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"
	"todo/db"

	"github.com/go-playground/validator"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"golang.org/x/crypto/bcrypt"
)

type User struct {
	ID       primitive.ObjectID `bson:"_id,omitempty" validate:"isdefault"`
	Name     string             `bson:"Name,omitempty" validate:"required"`
	Email    string             `bson:"Email,omitempty" validate:"required,email"`
	Password string             `bson:"Password,omitempty" validate:"required"`
}

//creating a new user or sign up
func (u *User) RegisterUser(rw http.ResponseWriter, r *http.Request) {
	client, err := db.GetClient()
	if err != nil {
		http.Error(rw, "DB connection failed", http.StatusInternalServerError)
		return
	}
	data := r.Context().Value(keyUser{}).(*User)
	ctx, _ := context.WithTimeout(context.Background(), time.Second*10)
	//checking user already exist or not
	err = u.checkUser(client, data.Email)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusBadRequest)
		return
	}
	data.Password, _ = u.HashPassword(data.Password)
	//creating the user in the database
	_, err = client.Database(db.DB).Collection(db.USER).InsertOne(ctx, data)
	if err != nil {
		http.Error(rw, fmt.Sprintf("Insertion failed:%s", err), http.StatusBadRequest)
		return
	}
	rw.Write([]byte("Created the user"))

}

//Hash the password
func (u *User) HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 14)
	return string(bytes), err
}

//check the password
func (u *User) CheckPasswordHash(password, hash string) error {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	if err != nil {
		return err
	}
	return nil
}

//key struct for conetext.withValue func
type keyUser struct{}

//Middleware to validate user data sent from API
func (u *User) MiddlewareValidateUser(next http.Handler) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		j := json.NewDecoder(r.Body)
		if err := j.Decode(u); err != nil {
			http.Error(rw, "Unable to` decode JSON", http.StatusBadRequest)
			return
		}
		v := validator.New()
		err := v.Struct(u)
		if err != nil {
			http.Error(rw, err.Error(), http.StatusBadRequest)
			return
		}
		ctx := context.WithValue(r.Context(), keyUser{}, u)
		req := r.WithContext(ctx)
		next.ServeHTTP(rw, req)
	})
}

//check whether user already exist in db
func (u *User) checkUser(m *mongo.Client, e string) error {
	c, err := m.Database(db.DB).Collection(db.USER).CountDocuments(context.TODO(), bson.M{"Email": u.Email})
	if err != nil {
		return err
	}
	if c > 0 {
		err = errors.New("Email already exist")
		return err
	}
	return nil
}
