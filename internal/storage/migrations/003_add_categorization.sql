-- Categories table
CREATE TABLE IF NOT EXISTS categories (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    parent_category TEXT,
    color TEXT,
    usage_count INTEGER DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(name, parent_category)
);

-- Bookmark categories mapping
CREATE TABLE IF NOT EXISTS bookmark_categories (
    bookmark_id TEXT REFERENCES bookmarks(id) ON DELETE CASCADE,
    category_id INTEGER REFERENCES categories(id) ON DELETE CASCADE,
    is_primary BOOLEAN DEFAULT FALSE,
    confidence_score REAL,
    user_approved BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (bookmark_id, category_id)
);

-- Tags table (normalized)
CREATE TABLE IF NOT EXISTS tags (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT UNIQUE NOT NULL,
    usage_count INTEGER DEFAULT 0
);

-- Bookmark tags mapping
CREATE TABLE IF NOT EXISTS bookmark_tags (
    bookmark_id TEXT REFERENCES bookmarks(id) ON DELETE CASCADE,
    tag_id INTEGER REFERENCES tags(id) ON DELETE CASCADE,
    PRIMARY KEY (bookmark_id, tag_id)
);

-- Add categorization metadata to bookmarks
ALTER TABLE bookmarks ADD COLUMN categorization_date TIMESTAMP;
ALTER TABLE bookmarks ADD COLUMN categorization_confidence REAL;
ALTER TABLE bookmarks ADD COLUMN categorization_status TEXT DEFAULT 'pending';

-- Indexes for performance
CREATE INDEX IF NOT EXISTS idx_bookmark_categories_bookmark_id ON bookmark_categories(bookmark_id);
CREATE INDEX IF NOT EXISTS idx_bookmark_tags_bookmark_id ON bookmark_tags(bookmark_id);
CREATE INDEX IF NOT EXISTS idx_categories_name ON categories(name);
CREATE INDEX IF NOT EXISTS idx_tags_name ON tags(name);