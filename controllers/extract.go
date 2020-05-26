package controllers

import (
	"fmt"
	"net/http"
)

type server struct{}

var collection *mongo.Collection

func (*server) CreateRecord(w http.ResponseWriter, r *http.Request) {

	fmt.Println("I am in extract createRecord controller")

}
