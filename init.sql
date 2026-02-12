-- Drop tables if they exist (order matters)
DROP TABLE IF EXISTS problem_idea;
DROP TABLE IF EXISTS problem_product;
DROP TABLE IF EXISTS idea_product;
DROP TABLE IF EXISTS source_items;
DROP TABLE IF EXISTS problem_categories;
DROP TABLE IF EXISTS idea_categories;
DROP TABLE IF EXISTS product_categories;
DROP TABLE IF EXISTS categories;
DROP TABLE IF EXISTS ideas;
DROP TABLE IF EXISTS products;
DROP TABLE IF EXISTS problems;

-- Enable foreign keys
PRAGMA foreign_keys = ON;

-- ======================
-- Categories
-- ======================
CREATE TABLE categories (
    slug TEXT PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- ======================
-- Problems
-- ======================
CREATE TABLE problems (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    slug TEXT UNIQUE NOT NULL,
    title TEXT NOT NULL,
    description TEXT NOT NULL,
    pain_points TEXT,
    score INTEGER DEFAULT 0,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    embedding F32_BLOB(768)
);

-- ======================
-- Ideas
-- ======================
CREATE TABLE ideas (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    slug TEXT UNIQUE NOT NULL,
    title TEXT NOT NULL,
    description TEXT NOT NULL,
    features TEXT,
    score INTEGER DEFAULT 0,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- ======================
-- Products
-- ======================
CREATE TABLE products (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    slug TEXT UNIQUE NOT NULL,
    name TEXT NOT NULL,
    description TEXT NOT NULL,
    url TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- ======================
-- Category junctions
-- ======================
CREATE TABLE problem_categories (
    problem_id INTEGER NOT NULL,
    category_slug TEXT NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (problem_id, category_slug),
    FOREIGN KEY (problem_id) REFERENCES problems(id) ON DELETE CASCADE,
    FOREIGN KEY (category_slug) REFERENCES categories(slug) ON DELETE CASCADE
);

CREATE TABLE idea_categories (
    idea_id INTEGER NOT NULL,
    category_slug TEXT NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (idea_id, category_slug),
    FOREIGN KEY (idea_id) REFERENCES ideas(id) ON DELETE CASCADE,
    FOREIGN KEY (category_slug) REFERENCES categories(slug) ON DELETE CASCADE
);

CREATE TABLE product_categories (
    product_id INTEGER NOT NULL,
    category_slug TEXT NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (product_id, category_slug),
    FOREIGN KEY (product_id) REFERENCES products(id) ON DELETE CASCADE,
    FOREIGN KEY (category_slug) REFERENCES categories(slug) ON DELETE CASCADE
);

-- ======================
-- Source items
-- ======================
CREATE TABLE source_items (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    source TEXT NOT NULL,
    source_item_id TEXT NOT NULL,
    title TEXT NOT NULL,
    content TEXT NOT NULL,
    author TEXT,
    url TEXT,
    score INTEGER DEFAULT 0,
    analysis_result TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    source_created_at DATETIME,

    problem_id INTEGER,
    idea_id INTEGER,
    product_id INTEGER,

    FOREIGN KEY (problem_id) REFERENCES problems(id) ON DELETE SET NULL,
    FOREIGN KEY (idea_id) REFERENCES ideas(id) ON DELETE SET NULL,
    FOREIGN KEY (product_id) REFERENCES products(id) ON DELETE SET NULL
);

-- ======================
-- Problem ↔ Idea
-- ======================
CREATE TABLE problem_idea (
    problem_id INTEGER NOT NULL,
    idea_id INTEGER NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (problem_id, idea_id),
    FOREIGN KEY (problem_id) REFERENCES problems(id) ON DELETE CASCADE,
    FOREIGN KEY (idea_id) REFERENCES ideas(id) ON DELETE CASCADE
);

-- ======================
-- Problem ↔ Product
-- ======================
CREATE TABLE problem_product (
    problem_id INTEGER NOT NULL,
    product_id INTEGER NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (problem_id, product_id),
    FOREIGN KEY (problem_id) REFERENCES problems(id) ON DELETE CASCADE,
    FOREIGN KEY (product_id) REFERENCES products(id) ON DELETE CASCADE
);

-- ======================
-- Idea ↔ Product
-- ======================
CREATE TABLE idea_product (
    idea_id INTEGER NOT NULL,
    product_id INTEGER NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (idea_id, product_id),
    FOREIGN KEY (idea_id) REFERENCES ideas(id) ON DELETE CASCADE,
    FOREIGN KEY (product_id) REFERENCES products(id) ON DELETE CASCADE
);

-- ======================
-- Seed categories
-- ======================
INSERT OR IGNORE INTO categories (slug, name) VALUES
('technology', 'Technology'),
('healthcare', 'Healthcare'),
('finance', 'Finance'),
('education', 'Education'),
('e-commerce', 'E-commerce'),
('productivity', 'Productivity'),
('communication', 'Communication'),
('entertainment', 'Entertainment'),
('travel', 'Travel'),
('food-beverage', 'Food & Beverage'),
('fitness', 'Fitness'),
('real-estate', 'Real Estate'),
('transportation', 'Transportation'),
('automotive', 'Automotive'),
('fashion', 'Fashion'),
('beauty', 'Beauty'),
('home-garden', 'Home & Garden'),
('pets', 'Pets'),
('sports', 'Sports'),
('gaming', 'Gaming'),
('music', 'Music'),
('art-design', 'Art & Design'),
('photography', 'Photography'),
('legal', 'Legal'),
('hr-recruiting', 'HR & Recruiting'),
('marketing', 'Marketing'),
('sales', 'Sales'),
('customer-service', 'Customer Service'),
('analytics', 'Analytics'),
('security', 'Security'),
('sustainability', 'Sustainability'),
('social-media', 'Social Media'),
('ai-ml', 'AI & Machine Learning'),
('iot', 'Internet of Things'),
('blockchain', 'Blockchain'),
('saas', 'SaaS'),
('mobile', 'Mobile'),
('web', 'Web'),
('hardware', 'Hardware'),
('infrastructure', 'Infrastructure');
