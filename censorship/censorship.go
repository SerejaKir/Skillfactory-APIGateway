package censorship

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
)

type Request struct {
	Comment string `json:"comment"`
}

type Response struct {
	Allowed bool `json:"allowed"`
}

func main() {
	http.HandleFunc("/check", censorHandler)
	log.Println("Censorship service started on :8082")
	log.Fatal(http.ListenAndServe(":8082", nil))
}

func censorHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req Request
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Проверка на запрещенные слова
	forbiddenWords := []string{"qwerty", "йцукен", "zxcvbn"}
	allowed := true
	for _, word := range forbiddenWords {
		if strings.Contains(strings.ToLower(req.Comment), strings.ToLower(word)) {
			allowed = false
			break
		}
	}

	json.NewEncoder(w).Encode(Response{Allowed: allowed})
}
