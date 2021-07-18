package db

import (
	"context"
	"sync"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

//*mongo client
var clientInstance *mongo.Client

// monogo client error
var clientInstanceError error

//Used to excute the client creation function only once
var mongoOnce sync.Once

const (
	CONNECTIONSTRING = "mongodb://localhost:27017"
	DB               = "ToDoList"
	TODO             = "todo"
	USER             = "User"
)

func GetClient() (*mongo.Client, error) {
	//Connection creation will be done only once
	mongoOnce.Do(func() {
		//mongo client options
		clientOptions := options.Client().ApplyURI(CONNECTIONSTRING)
		// Connect to MongoDB
		client, err := mongo.Connect(context.TODO(), clientOptions)
		if err != nil {
			clientInstanceError = err
		}
		// Check the connection
		err = client.Ping(context.TODO(), nil)
		if err != nil {
			clientInstanceError = err
		}
		clientInstance = client
	})
	return clientInstance, clientInstanceError
}
