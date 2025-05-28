DROP TABLE IF EXISTS comments;
CREATE TABLE IF NOT EXISTS comments (
    id SERIAL PRIMARY KEY,
    news_id INT,
    content TEXT NOT NULL DEFAULT 'empty',
    pub_time INTEGER DEFAULT extract (epoch from now())
);

INSERT INTO comments(news_id,content)  VALUES (1,'тестовый комментарий 1');

INSERT INTO comments(news_id,content)  VALUES (1,'тестовый комментарий 2');