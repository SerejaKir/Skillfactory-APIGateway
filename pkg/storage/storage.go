// Пакет для работы с БД приложения GoNews.
package storage

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
)

// База данных.
type DB struct {
	Pool *pgxpool.Pool
}

// Публикация, получаемая из RSS.
type Post struct {
	ID      int    // номер записи
	Title   string // заголовок публикации
	Content string // содержание публикации
	PubTime int64  // время публикации
	Link    string // ссылка на источник
}

const (
	host           = "172.16.87.117"
	portPostges    = 5432
	userDB         = "sergey"
	password       = "password"
	dbnamePostges  = "postgres"
	collectionName = "newsdb"
)

// Запись в БД новых новостей
func New() (*DB, error) {
	os.Setenv("newsdb", "postgres://"+userDB+":"+password+"@"+host+"/"+dbnamePostges)
	connstr := os.Getenv("newsdb")
	if connstr == "" {
		return nil, errors.New("не указано подключение к БД")
	}
	pool, err := pgxpool.New(context.Background(), connstr)
	if err != nil {
		return nil, err
	}
	db := DB{
		Pool: pool,
	}

	// Выполнение SQL-скрипта
	if err := db.initSchema(); err != nil {
		return nil, fmt.Errorf("ошибка инициализации схемы: %v", err)
	}

	return &db, nil
}

// initSchema выполняет SQL-скрипт из файла для инициализации БД
func (db *DB) initSchema() error {
	// Чтение файла schema.sql
	sqlBytes, err := ioutil.ReadFile("./schema.sql")
	if err != nil {
		return fmt.Errorf("не удалось прочитать файл schema.sql: %v", err)
	}

	// Выполнение SQL-скрипта
	_, err = db.Pool.Exec(context.Background(), string(sqlBytes))
	if err != nil {
		return fmt.Errorf("ошибка выполнения SQL-скрипта: %v", err)
	}

	return nil
}

func (db *DB) StoreNews(news []Post) error {
	for _, post := range news {
		_, err := db.Pool.Exec(context.Background(), `
		INSERT INTO news(title, content, pub_time, link)
		VALUES ($1, $2, $3, $4)`,
			post.Title,
			post.Content,
			post.PubTime,
			post.Link,
		)
		if err != nil {
			return err
		}
	}
	return nil
}

// News возвращает последние новости из БД.
func (db *DB) News(n int) ([]Post, error) {
	if n == 0 {
		n = 10
	}
	rows, err := db.Pool.Query(context.Background(), `
	SELECT id, title, content, pub_time, link FROM news
	ORDER BY pub_time DESC
	LIMIT $1
	`,
		n,
	)
	if err != nil {
		return nil, err
	}
	var news []Post
	for rows.Next() {
		var p Post
		err = rows.Scan(
			&p.ID,
			&p.Title,
			&p.Content,
			&p.PubTime,
			&p.Link,
		)
		if err != nil {
			return nil, err
		}
		news = append(news, p)
	}
	return news, rows.Err()
}

// Закрытие БД
func (db *DB) Close() {
	if db.Pool != nil {
		db.Pool.Close()
	}
}
