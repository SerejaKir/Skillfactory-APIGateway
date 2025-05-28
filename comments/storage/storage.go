// Пакет для работы с БД приложения GoNews.
package storage

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
)

// База данных.
type DB struct {
	Pool *pgxpool.Pool
}

// конфигурация подключения к PostgreSQL
type sqlPostgres struct {
	Host           string `json:"host"`
	portPostgres   int    `json:"portPostgres"`
	UserDB         string `json:"userDB"`
	Password       string `json:"password"`
	DBnamePostges  string `json:"dbnamePostges"`
	CollectionName string `json:"collectionName"`
}

type Comment struct {
	ID      int    `json:"ID,omitempty"`
	NewsID  int    `json:"newsID,omitempty"`
	Content string `json:"content,omitempty"`
	PubTime int64  `json:"pubTime,omitempty"`
}

// Запись в БД новых новостей
func New() (*DB, error) {
	// Чтение конфигурации базы данных файла
	b, err := ioutil.ReadFile("./sqlPostgres.json")
	if err != nil {
		log.Fatalf("не удалось прочитать файл sqlPostgres.json: %v", err)
	}
	var sqlParams sqlPostgres
	err = json.Unmarshal(b, &sqlParams)
	if err != nil {
		log.Fatal(err)
	}

	os.Setenv(sqlParams.CollectionName, "postgres://"+sqlParams.UserDB+":"+sqlParams.Password+"@"+sqlParams.Host+"/"+sqlParams.DBnamePostges)
	connstr := os.Getenv(sqlParams.CollectionName)
	if connstr == "" {
		return nil, errors.New("не указано подключение к БД комментариев")
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
		return nil, fmt.Errorf("ошибка инициализации схемы комментариев: %v", err)
	} else {
		fmt.Println("создана БД комментариев")
	}

	return &db, nil
}

// initSchema выполняет SQL-скрипт из файла для инициализации БД
func (db *DB) initSchema() error {
	// Чтение файла для создания схемы базы данных
	sqlBytes, err := ioutil.ReadFile("./schemaComments.sql")
	if err != nil {
		return fmt.Errorf("не удалось прочитать файл schemaComments.sql: %v", err)
	}

	// Выполнение SQL-скрипта
	_, err = db.Pool.Exec(context.Background(), string(sqlBytes))
	if err != nil {
		return fmt.Errorf("ошибка выполнения SQL-скрипта: %v", err)
	}
	return nil
}

// AllComments выводит все коменты.
func (db *DB) AllComments(newsID int) ([]Comment, error) {
	rows, err := db.Pool.Query(context.Background(), "SELECT * FROM comments WHERE news_id = $1;", newsID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var comments []Comment
	for rows.Next() {
		var c Comment
		err = rows.Scan(&c.ID, &c.NewsID, &c.Content, &c.PubTime)
		if err != nil {
			return nil, err
		}
		comments = append(comments, c)
	}
	return comments, rows.Err()
}

// AddComment добавляет коменты.
func (db *DB) AddComment(c Comment) error {
	_, err := db.Pool.Exec(context.Background(),
		"INSERT INTO comments (news_id,content) VALUES ($1,$2);", c.NewsID, c.Content)
	if err != nil {
		return err
	}
	return nil
}

// DeleteComment удаляет коменты.
func (db *DB) DeleteComment(c Comment) error {
	_, err := db.Pool.Exec(context.Background(),
		"DELETE FROM comments WHERE id=$1;", c.ID)
	if err != nil {
		return err
	}
	return nil
}
