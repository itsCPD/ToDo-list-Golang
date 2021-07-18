package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"
	"todo/db"
	"todo/models"

	"github.com/gorilla/mux"
)

func main() {
	var wait time.Duration
	flag.DurationVar(&wait, "graceful-timeout", time.Second*15, "the duration for which the server gracefully wait for existing connections to finish - e.g. 15s or 1m")
	flag.Parse()

	r := mux.NewRouter()
	to := models.ToDoList{} //struct for todolist
	login := models.Login{} //struct for login

	// Todo list routers
	//read all active todoLists in DB
	getAllTodo := r.Methods(http.MethodGet).Subrouter()
	getAllTodo.HandleFunc("/getalltodo", to.GetAll)
	getAllTodo.Use(login.MiddlewareAuth)

	//get a single todolist
	getTodo := r.Methods(http.MethodGet).Subrouter()
	getTodo.HandleFunc("/gettodo/{id}", to.GetTodo)
	getTodo.Use(login.MiddlewareAuth, to.MiddlewareValidateID)

	//create a new todoList
	createroute := r.Methods(http.MethodPost).Subrouter()
	createroute.HandleFunc("/createtodo", to.Create)
	createroute.Use(login.MiddlewareAuth, to.MiddlewareValidateToDo)

	//update a single todolist
	updateTodo := r.Methods(http.MethodPut).Subrouter()
	updateTodo.HandleFunc("/updatetodo/{id}", to.UpdateToDo)
	updateTodo.Use(login.MiddlewareAuth, to.MiddlewareValidateID, to.MiddlewareValidateToDo)

	//Close a todolist
	closeTodo := r.Methods(http.MethodPatch).Subrouter()
	closeTodo.HandleFunc("/closetodo/{id}", to.CloseTodo)
	closeTodo.Use(login.MiddlewareAuth, to.MiddlewareValidateID)

	//User and auth routers
	//create a user
	user := models.User{}
	createUser := r.Methods(http.MethodPost).Subrouter()
	createUser.HandleFunc("/register", user.RegisterUser)
	createUser.Use(user.MiddlewareValidateUser)

	// login routes
	loginUser := r.Methods(http.MethodGet).Subrouter()
	loginUser.HandleFunc("/login", login.Login)

	//connect to database
	client, err := db.GetClient()
	if err != nil {
		fmt.Println("DB connection failed")
		return
	} else {
		fmt.Println("DB connected")
	}
	defer client.Disconnect(context.Background())

	//server setup starts here
	srv := &http.Server{
		Addr:         "0.0.0.0:9090",
		WriteTimeout: time.Second * 15,
		ReadTimeout:  time.Second * 15,
		IdleTimeout:  time.Second * 60,
		Handler:      r,
	}

	// Run our server in a goroutine so that it doesn't block.
	go func() {
		if err := srv.ListenAndServe(); err != nil {
			log.Println(err)
		}
	}()

	c := make(chan os.Signal, 1)
	// We'll accept graceful shutdowns when quit via SIGINT (Ctrl+C)
	// SIGKILL, SIGQUIT or SIGTERM (Ctrl+/) will not be caught.
	signal.Notify(c, os.Interrupt)

	// Block until we receive our signal.
	<-c

	// Create a deadline to wait for.
	ctx, cancel := context.WithTimeout(context.Background(), wait)
	defer cancel()
	// Doesn't block if no connections, but will otherwise wait
	// until the timeout deadline.
	srv.Shutdown(ctx)
	// Optionally, you could run srv.Shutdown in a goroutine and block on
	// <-ctx.Done() if your application should wait for other services
	// to finalize based on context cancellation.
	log.Println("shutting down")
	os.Exit(0)
}
