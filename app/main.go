package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	repo "github.com/MElenaO/aws_backend/repository"
	"golang.org/x/exp/slog"
)

type repository interface {
	WriteItem(ctx context.Context, key, value string) error
	ReadItem(ctx context.Context, key string) (string, error)
}

type postValue struct {
	Greeting string `json:"greeting" validate:"required"`
}

func (p postValue) validate() error {
	if p.Greeting == "" {
		return fmt.Errorf("Missing Greeting field in request")
	}
	return nil
}

func main() {

	http.HandleFunc("/greeting/", greetingHandler)
	http.HandleFunc("/ping", pingHandler)
	err := http.ListenAndServe(":8080", nil)
	log.Fatal(err)
}

func getKey(url string) string {
	pathParts := strings.Split(url, "/")
	return pathParts[len(pathParts)-1]
}

func greetingHandler(res http.ResponseWriter, req *http.Request) {
	repo := repo.New()
	switch req.Method {
	case "GET":
		key := getKey(req.URL.Path)
		item, err := repo.ReadItem(context.Background(), key)

		if err != nil {
			slog.Error(err.Error())
			http.Error(res, "No greeting stored in this path...", http.StatusNotFound)
		} else {
			fmt.Fprintf(res, "Today's greeting is: %v", item)
		}
	case "POST":
		var value postValue
		if err := json.NewDecoder(req.Body).Decode(&value); err != nil {
			http.Error(res, err.Error(), http.StatusBadRequest)
		}
		if err := value.validate(); err != nil {
			http.Error(res, fmt.Sprintf("%v", err), http.StatusBadRequest)
		} else {
			// Pass stored value to db for store in table
			key := getKey(req.URL.Path)
			err = repo.WriteItem(context.Background(), key, value.Greeting)
			if err != nil {
				slog.Error(err.Error())
				fmt.Fprintf(res, "cannot store received value = %s\n", value)
			}
		}
	default:
		fmt.Fprint(res, "Only GET and POST methods are supported.")
	}
}

func pingHandler(res http.ResponseWriter, req *http.Request) {
	fmt.Fprint(res, "Health check")
}
