// API приложения GoNews.
package api

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"Skillfactory-APIGateway/pkg/storage"

	"github.com/gorilla/mux"
)

type API struct {
	db *storage.DB
	r  *mux.Router
}

// Конструктор API.
func New(db *storage.DB) *API {
	a := API{db: db, r: mux.NewRouter()}
	a.endpoints()
	return &a
}

// Router возвращает маршрутизатор для использования
// в качестве аргумента HTTP-сервера.
func (api *API) Router() *mux.Router {
	return api.r
}

// Регистрация методов API в маршрутизаторе запросов.
func (api *API) endpoints() {
	// получить страницу с определенным номером
	api.r.HandleFunc("/news/latest", api.newsLatestHandler).Methods(http.MethodGet, http.MethodOptions)
	// поиск новости по id
	api.r.HandleFunc("/news/search", api.newsDetailedHandler).Methods(http.MethodGet, http.MethodOptions)
	// получить n последних новостей
	api.r.HandleFunc("/news/{n}", api.posts).Methods(http.MethodGet, http.MethodOptions)
	// все публикации
	api.r.PathPrefix("/").Handler(http.StripPrefix("/", http.FileServer(http.Dir("./webapp"))))
}

func (api *API) posts(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	if r.Method == http.MethodOptions {
		return
	}
	s := mux.Vars(r)["n"]
	n, _ := strconv.Atoi(s)
	news, err := api.db.News(n)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(news)
}

// Получение страницу с определенным номером
func (api *API) newsLatestHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	pageParam := r.URL.Query().Get("page")
	if pageParam == "" {
		pageParam = "1"
	}

	page, err := strconv.Atoi(pageParam)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	posts, err := api.db.Posts(page)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	err = json.NewEncoder(w).Encode(posts)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// Получение публикаций по id
func (api *API) newsDetailedHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	idParam := r.URL.Query().Get("id")

	log.Println(idParam)

	id, err := strconv.Atoi(idParam)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	post, err := api.db.PostDetal(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	err = json.NewEncoder(w).Encode(post)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

}
