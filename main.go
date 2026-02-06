package main

import(
	"net/http"
)

func main() {
	fs := http.FileServer(http.Dir("./frontend")
	http.Handle("/", fs)

	http.HandleFunc("/api/messages", func(w http.ResponseWriter, r *http.Request){
		messages := []map[string]string{
			{"user:": "Alice", "text": "Привет!"},
			{"user:": "Bob", "text": "Привет, как дела?"},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(messages)
	})

	log.Println("Запуск сервера на http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

