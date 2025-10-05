package main

import (
	"encoding/json"
	"log"
	"net/http"
)

type Message struct {
	Content string `json:"content"`
}

func messagesHandler(w http.ResponseWriter, r *http.Request) {
	messages := []Message{
		{Content: "mock"},
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(messages); err != nil {
		log.Println("Error encoding json:", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

func main() {
	http.HandleFunc("/messages", messagesHandler)
	log.Println("Server starting on port 8080...")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}