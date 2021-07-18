package models

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	authJWT "todo/auth"
	"todo/db"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Login struct {
	Email    string `bson:"Email,omitempty" validate:"required,email"`
	Password string `bson:"Password,omitempty" validate:"required"`
}

// User login func, returns JWT token
func (l *Login) Login(rw http.ResponseWriter, r *http.Request) {
	client, err := db.GetClient()
	if err != nil {
		http.Error(rw, "DB connection failed", http.StatusInternalServerError)
		return
	}
	err1 := json.NewDecoder(r.Body).Decode(l)
	if err1 != nil {
		http.Error(rw, err1.Error(), http.StatusBadRequest)
		return
	}
	opts := options.Find().SetProjection(bson.M{"Email": 1, "Password": 1})
	cursor, err2 := client.Database(db.DB).Collection(db.USER).Find(context.TODO(), bson.M{"Email": l.Email}, opts)
	if err2 != nil {
		http.Error(rw, err2.Error(), http.StatusUnauthorized)
		return
	}
	var v []bson.M
	if len(v) < 0 {
		err = errors.New("email does not exist")
		http.Error(rw, err.Error(), http.StatusUnauthorized)
		return
	}
	err = cursor.All(context.TODO(), &v)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusUnauthorized)
		return
	}
	u := User{}
	actualPassword := v[0]["Password"].(string)
	t := authJWT.JwtWrapper{}
	err = u.CheckPasswordHash(l.Password, actualPassword)
	if err != nil {
		err = errors.New("wrong password")
		http.Error(rw, err.Error(), http.StatusUnauthorized)
		return
	}
	token, terr := t.GenerateToken(l.Email)
	if terr != nil {
		http.Error(rw, terr.Error(), http.StatusUnauthorized)
		return
	}
	var j struct {
		Token string
	}
	j.Token = token
	json.NewEncoder(rw).Encode(j)
}

type emailKey struct{} //key struct for adding new context to the request

// Middleware to check user authorization
func (l *Login) MiddlewareAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		clientToken := r.Header.Get("Authorization")
		if clientToken == "" {
			err := errors.New("No Auth provided")
			http.Error(rw, err.Error(), http.StatusUnauthorized)
			return
		}
		t := authJWT.JwtWrapper{}
		claims, err := t.ValidateToken(clientToken)
		if err != nil {
			http.Error(rw, err.Error(), http.StatusUnauthorized)
			return
		}
		e := claims.Email
		ctx := context.WithValue(r.Context(), emailKey{}, e)
		req := r.WithContext(ctx)
		next.ServeHTTP(rw, req)
	})
}
