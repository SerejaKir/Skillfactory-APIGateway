// Сервер GoNews.
package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	commentStorege "Skillfactory-APIGateway/comments/storage"
	"Skillfactory-APIGateway/pkg/api"
	"Skillfactory-APIGateway/pkg/rss"
	"Skillfactory-APIGateway/pkg/storage"
)

// конфигурация приложения
type config struct {
	URLS   []string `json:"rss"`
	Period int      `json:"request_period"`
}

func main() {
	// инициализация зависимостей приложения
	db, err := storage.New()
	if err != nil {
		log.Fatal(err)
	}
	defer db.Pool.Close()
	dbComment, err := commentStorege.New()
	if err != nil {
		log.Fatal(err)
	}
	defer dbComment.Pool.Close()
	api := api.New(db, dbComment)

	// чтение и раскодирование файла конфигурации
	b, err := ioutil.ReadFile("./config.json")
	if err != nil {
		log.Fatal(err)
	}
	var config config
	err = json.Unmarshal(b, &config)
	if err != nil {
		log.Fatal(err)
	}

	// запуск парсинга новостей в отдельном потоке
	// для каждой ссылки
	chPosts := make(chan []storage.Post)
	chErrs := make(chan error)
	for _, url := range config.URLS {
		go parseURL(url, chPosts, chErrs, config.Period)
	}

	// запись потока новостей в БД
	go func() {
		for posts := range chPosts {
			db.StoreNews(posts)
			if err != nil {
				log.Println(err)
			}
		}
	}()

	// обработка потока ошибок
	go func() {
		for err := range chErrs {
			log.Println("ошибка:", err)
		}
	}()

	// запуск веб-сервера с API и приложением
	err = http.ListenAndServe(":80", api.Router())
	if err != nil {
		log.Fatal(err)
	}
}

// Асинхронное чтение потока RSS. Раскодированные
// новости и ошибки пишутся в каналы.
func parseURL(url string, posts chan<- []storage.Post, errs chan<- error, period int) {
	for {
		news, err := rss.Parse(url)
		if err != nil {
			errs <- err
			continue
		}
		posts <- news
		time.Sleep(time.Minute * time.Duration(period))
	}
}
