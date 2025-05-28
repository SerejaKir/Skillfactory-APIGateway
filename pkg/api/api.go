// API приложения GoNews.
package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"Skillfactory-APIGateway/censorship"
	dbComments "Skillfactory-APIGateway/comments/storage"
	"Skillfactory-APIGateway/pkg/storage"

	"github.com/gorilla/mux"
)

type API struct {
	db         *storage.DB
	dbComments *dbComments.DB
	r          *mux.Router
}

// Конструктор API.
func New(db *storage.DB, dbComments *dbComments.DB) *API {
	a := API{db: db, dbComments: dbComments, r: mux.NewRouter()}
	a.r.Use(a.requestIDMiddleware)
	a.r.Use(a.loggingMiddleware)
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
	// получить страницу с определенным номером: http://localhost/news/latest?page=4&s=Go или /news/latest?page=1
	api.r.HandleFunc("/news/latest", api.newsLatestHandler).Methods(http.MethodGet, http.MethodOptions)
	// поиск новости с комментарием по id: http://localhost/news/detailed?id=1
	api.r.HandleFunc("/news/detailed", api.newsDetailedHandler).Methods(http.MethodGet, http.MethodOptions)
	// получить n последних новостей
	api.r.HandleFunc("/news/{n}", api.posts).Methods(http.MethodGet, http.MethodOptions)

	// обработчиков комментариев http://localhost/comments?news_id=1
	api.r.HandleFunc("/comments/add", api.addCommentHandler).Methods(http.MethodPost, http.MethodOptions)
	api.r.HandleFunc("/comments/del", api.deletePostHandler).Methods(http.MethodDelete, http.MethodOptions)
	api.r.HandleFunc("/comments", api.commentsHandler).Methods(http.MethodGet, http.MethodOptions)

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

// Получение страницу с определенным номером и поиск
func (api *API) newsLatestHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	pageParam := r.URL.Query().Get("page")
	page := 1
	if pageParam != "" {
		var err error
		page, err = strconv.Atoi(pageParam)
		if err != nil {
			http.Error(w, "Invalid page parameter", http.StatusBadRequest)
			return
		}
	}

	searchQuery := r.URL.Query().Get("s")

	var posts []storage.Post
	var err error
	var pagination storage.Pagination

	if searchQuery != "" {
		// Поиск с пагинацией
		posts, pagination, err = api.db.PostSearchILIKE(searchQuery, 10, (page-1)*10)
	} else {
		// Обычный список с пагинацией
		posts, err = api.db.Posts((page - 1) * 10)
		// Для простоты считаем что у нас фиксированное количество страниц
		// В реальной системе нужно делать COUNT запрос
		pagination = storage.Pagination{
			Page:       page,
			Limit:      10,
			NumOfPages: 10,
		}
	}

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"news":       posts,
		"pagination": pagination,
	}

	json.NewEncoder(w).Encode(response)
}

// Получение публикаций по id
func (api *API) newsDetailedHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	idParam := r.URL.Query().Get("id")
	id, err := strconv.Atoi(idParam)
	if err != nil {
		http.Error(w, "Invalid id parameter", http.StatusBadRequest)
		return
	}

	// Получаем новость
	post, err := api.db.PostDetal(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	// Получаем комментарии
	comments, err := api.dbComments.AllComments(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"news":     post,
		"comments": comments,
	}

	json.NewEncoder(w).Encode(response)
}

// commentsHandler, который выводит комментарий по id статьи
func (api *API) commentsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	parseId := r.URL.Query().Get("news_id")
	newsId, err := strconv.Atoi(parseId)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	comments, err := api.dbComments.AllComments(newsId)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	err = json.NewEncoder(w).Encode(comments)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// Добавление комментария
func (api *API) addCommentHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	var c dbComments.Comment
	err := json.NewDecoder(r.Body).Decode(&c)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Проверка цензуры
	allowed, err := api.checkCensorship(c.Content)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if !allowed {
		http.Error(w, "Comment contains forbidden words", http.StatusBadRequest)
		return
	}

	err = api.dbComments.AddComment(c)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusCreated)
}

func (api *API) checkCensorship(comment string) (bool, error) {
	reqBody, err := json.Marshal(map[string]string{"comment": comment})
	if err != nil {
		return false, err
	}

	resp, err := http.Post("http://localhost:8082/check", "application/json", bytes.NewBuffer(reqBody))
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	var result censorship.Response
	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		return false, err
	}

	return result.Allowed, nil
}

// Удаление комента.
func (api *API) deletePostHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	var c dbComments.Comment
	err := json.NewDecoder(r.Body).Decode(&c)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	err = api.dbComments.DeleteComment(c)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

// Middleware для логирования
func (api *API) loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		duration := time.Since(start)

		requestID := r.Context().Value("request_id")
		if requestID == nil {
			requestID = "unknown"
		}

		log.Printf(
			"Request: %s %s, duration: %v, request_id: %v",
			r.Method,
			r.RequestURI,
			duration,
			requestID,
		)
	})
}

// Middleware для request_id
func (api *API) requestIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := r.URL.Query().Get("request_id")
		if requestID == "" {
			requestID = generateRequestID()
		}

		ctx := context.WithValue(r.Context(), "request_id", requestID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func generateRequestID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}
