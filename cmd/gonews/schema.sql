DROP TABLE IF EXISTS news;
CREATE TABLE IF NOT EXISTS news (
    id SERIAL PRIMARY KEY,
    title TEXT NOT NULL DEFAULT 'empty',
    content TEXT NOT NULL DEFAULT 'empty',
    pub_time INTEGER DEFAULT extract (epoch from now()),
    link TEXT NOT NULL UNIQUE
);