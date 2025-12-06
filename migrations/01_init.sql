CREATE TABLE IF NOT EXISTS ideas (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    reddit_id TEXT UNIQUE NOT NULL,
    title TEXT NOT NULL,
    content TEXT NOT NULL,
    author TEXT,
    subreddit TEXT,
    url TEXT,
    score INTEGER DEFAULT 0,
    created_at TEXT DEFAULT CURRENT_TIMESTAMP,
    updated_at TEXT DEFAULT CURRENT_TIMESTAMP,
    reddit_created_at TEXT,
    categories TEXT
);

CREATE INDEX IF NOT EXISTS idx_subreddit ON ideas(subreddit);
CREATE INDEX IF NOT EXISTS idx_ai_score ON ideas(score);
CREATE INDEX IF NOT EXISTS idx_created_at ON ideas(created_at);
