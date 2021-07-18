package models

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
	"todo/db"

	"github.com/go-playground/validator"
	"github.com/gorilla/mux"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type ToDoList struct {
	ID     primitive.ObjectID `bson:"_id,omitempty" validate:"isdefault"`
	Email  string             `bson:"Email"`
	Name   string             `bson:"Name,omitempty" validate:"required"`
	Todo   string             `bson:"Todo,omitempty" validate:"required"`
	Active bool               `bson:"Active"`
}

func (m *ToDoList) Create(rw http.ResponseWriter, r *http.Request) {
	client, err := db.GetClient()
	if err != nil {
		http.Error(rw, "DB connection failed", http.StatusInternalServerError)
		return
	}
	//capturing the data came from MiddlewareValidateToDo
	data := r.Context().Value(keyToDO{}).(*ToDoList)
	e := r.Context().Value(emailKey{}).(string)
	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
	//Setting the todo Active value to true
	data.Active = true
	data.Email = e
	_, err = client.Database(db.DB).Collection(db.TODO).InsertOne(ctx, data)
	if err != nil {
		http.Error(rw, fmt.Sprintf("Insertion failed:%s", err), http.StatusBadRequest)
		return
	}
	rw.Write([]byte("Created the todo"))
}

//func to get all the active todos
func (m *ToDoList) GetAll(rw http.ResponseWriter, r *http.Request) {
	e := r.Context().Value(emailKey{})
	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
	client, err := db.GetClient()
	if err != nil {
		http.Error(rw, "DB connection failed", http.StatusInternalServerError)
		return
	}
	var v []bson.M
	//get active todos only
	s, err := client.Database(db.DB).Collection(db.TODO).Find(ctx, bson.M{"Active": true, "Email": e})
	if err != nil {
		http.Error(rw, fmt.Sprintf("No records found:%s", err), http.StatusBadRequest)
		return
	}
	if err = s.All(ctx, &v); err != nil {
		http.Error(rw, fmt.Sprintf("Error in deserializing:%s", err), http.StatusBadRequest)
		return
	}
	//If no data available
	if len(v) < 1 {
		http.Error(rw, "No Todo found", http.StatusNoContent)
		return
	}
	j := json.NewEncoder(rw)
	j.Encode(v)
}

//Get a single todo
func (m *ToDoList) GetTodo(rw http.ResponseWriter, r *http.Request) {
	v := r.Context().Value(keyToDOID{})
	j := json.NewEncoder(rw)
	j.Encode(v)
}

//Update a todo list
func (m *ToDoList) UpdateToDo(rw http.ResponseWriter, r *http.Request) {
	client, err := db.GetClient()
	if err != nil {
		http.Error(rw, "DB connection failed", http.StatusInternalServerError)
		return
	}
	v := r.Context().Value(keyToDOID{}).(*ToDoList)  //old data just for id
	data := r.Context().Value(keyToDO{}).(*ToDoList) // new data to be updated
	_, err1 := client.Database(db.DB).Collection(db.TODO).UpdateOne(context.TODO(), bson.M{"_id": v.ID},
		bson.M{"$set": bson.M{
			"Name":   data.Name,
			"Todo":   data.Todo,
			"Active": true,
		}})
	if err1 != nil {
		http.Error(rw, fmt.Sprintf("An Error occured:%s", err), http.StatusInternalServerError)
		return
	}
	rw.Write([]byte("Todo updated"))
}

//func close a todo list
func (m *ToDoList) CloseTodo(rw http.ResponseWriter, r *http.Request) {
	client, err := db.GetClient()
	if err != nil {
		http.Error(rw, "DB connection failed", http.StatusInternalServerError)
		return
	}
	v := r.Context().Value(keyToDOID{}).(*ToDoList)
	_, err1 := client.Database(db.DB).Collection(db.TODO).UpdateOne(context.TODO(), bson.M{"_id": v.ID},
		bson.M{"$set": bson.M{
			"Active": false,
		}})
	if err1 != nil {
		http.Error(rw, fmt.Sprintf("An Error occured:%s", err), http.StatusInternalServerError)
		return
	}
	rw.Write([]byte("Todo Closed"))
}

//key struct for conetext.withValue func
type keyToDO struct{}
type keyToDOID struct{}

//Validation middleware for creating and updating ToDoList
func (m *ToDoList) MiddlewareValidateToDo(next http.Handler) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		j := json.NewDecoder(r.Body)
		if err := j.Decode(m); err != nil {
			http.Error(rw, "Unable to decode JSON", http.StatusBadRequest)
			return
		}
		v := validator.New()
		err := v.Struct(m)
		if err != nil {
			http.Error(rw, err.Error(), http.StatusBadRequest)
			return
		}
		ctx := context.WithValue(r.Context(), keyToDO{}, m)
		req := r.WithContext(ctx)
		next.ServeHTTP(rw, req)
	})
}

//Validation middleware to check ID is available in the DB
func (m *ToDoList) MiddlewareValidateID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		objID, err := primitive.ObjectIDFromHex(vars["id"])
		if err != nil {
			http.Error(rw, "Invalid Todo ID", http.StatusBadRequest)
			return
		}
		ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
		client, err := db.GetClient()
		if err != nil {
			http.Error(rw, "DB connection failed", http.StatusInternalServerError)
			return
		}
		var v []bson.M
		s, err1 := client.Database(db.DB).Collection(db.TODO).Find(ctx, bson.M{"_id": objID})
		if err1 != nil {
			http.Error(rw, fmt.Sprintf("Error occured:%s", err1), http.StatusBadRequest)
			return
		}
		if err = s.All(ctx, &v); err != nil {
			http.Error(rw, fmt.Sprintf("Error in deserializing:%s", err), http.StatusBadRequest)
			return
		}
		//If no data available
		if len(v) < 1 {
			http.Error(rw, fmt.Sprintf("No todo with id:%s", objID), http.StatusNoContent)
			return
		} else {
			data := &ToDoList{}
			v1, err := bson.Marshal(v[0]) //Marshal 1st element in the slice
			if err != nil {
				http.Error(rw, err.Error(), http.StatusBadRequest)
				return
			}
			bson.Unmarshal(v1, &data)
			if data.Email != r.Context().Value(emailKey{}) {
				http.Error(rw, "Not authorised", http.StatusUnauthorized)
				return
			}
			ctx := context.WithValue(r.Context(), keyToDOID{}, data)
			req := r.WithContext(ctx)
			next.ServeHTTP(rw, req)
		}

	})
}
