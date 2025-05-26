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

type Pagination struct {
	NumOfPages int
	Page       int
	Limit      int
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

// вставка новой записи
func (db *DB) StoreNews(news []Post) error {
	var id int //проверка, что записалось
	for _, post := range news {
		//err := db.Pool.QueryRow(context.Background(), `
		_, err := db.Pool.Exec(context.Background(), `
		INSERT INTO news(title, content, pub_time, link)
		VALUES ($1, $2, $3, $4)
        RETURNING id`,
			post.Title,
			post.Content,
			post.PubTime,
			post.Link,
		) //.Scan(&id)
		if err != nil {
			return err
		}
		id++
		//fmt.Printf("Добавлена новость с ID: %d\n", id)
	}
	fmt.Printf("Добавлено %d записи с сайта %s\n", id, news[0].Link)
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

// PostSearchILIKE Поиск по заголовку
func (db *DB) PostSearchILIKE(pattern string, limit, offset int) ([]Post, Pagination, error) {
	pattern = "%" + pattern + "%"

	pagination := Pagination{
		Page:  offset/limit + 1,
		Limit: limit,
	}
	row := db.Pool.QueryRow(context.Background(), "SELECT count(*) FROM news WHERE title ILIKE $1;", pattern)
	err := row.Scan(&pagination.NumOfPages)

	if pagination.NumOfPages%limit > 0 {
		pagination.NumOfPages = pagination.NumOfPages/limit + 1
	} else {
		pagination.NumOfPages /= limit
	}

	if err != nil {
		return nil, Pagination{}, err
	}

	rows, err := db.Pool.Query(context.Background(), "SELECT * FROM news WHERE title ILIKE $1 ORDER BY pub_time DESC LIMIT $2 OFFSET $3;", pattern, limit, offset)
	if err != nil {
		return nil, Pagination{}, err
	}
	defer rows.Close()
	var posts []Post
	for rows.Next() {
		var p Post
		err = rows.Scan(&p.ID, &p.Title, &p.Content, &p.PubTime, &p.Link)
		if err != nil {
			return nil, Pagination{}, err
		}
		posts = append(posts, p)
	}
	return posts, pagination, rows.Err()
}

// Posts Получение странице с определенным номером
func (db *DB) Posts(Page int) ([]Post, error) {
	if Page < 1 {
		err := errors.New("invalid value - must be greater than zero")
		return nil, err
	}
	rows, err := db.Pool.Query(context.Background(), `
	SELECT * FROM news
	ORDER BY pub_time DESC LIMIT 10 OFFSET $1
	`,
		Page,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var posts []Post
	// итерированное по результату выполнения запроса
	// и сканирование каждой строки в переменную
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
		// добавление переменной в массив результатов
		posts = append(posts, p)

	}
	// ВАЖНО не забыть проверить rows.Err()
	return posts, rows.Err()
}

// PostDetal Получение публикаций по id
func (db *DB) PostDetal(id int) (Post, error) {
	if id < 1 {
		err := errors.New("invalid id - must be greater than zero")
		return Post{}, err
	}
	row := db.Pool.QueryRow(context.Background(), `
	SELECT * FROM news 
    WHERE id =$1;
	`, id)
	var post Post
	err := row.Scan(
		&post.ID,
		&post.Title,
		&post.Content,
		&post.PubTime,
		&post.Link)
	if err != nil {
		return Post{}, err
	}
	return post, nil
}
