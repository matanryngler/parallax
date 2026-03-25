-- Test database initialization
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    email VARCHAR(100) UNIQUE NOT NULL,
    active BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE tasks (
    id SERIAL PRIMARY KEY,
    title VARCHAR(200) NOT NULL,
    user_id INTEGER REFERENCES users(id),
    status VARCHAR(20) DEFAULT 'pending',
    priority INTEGER DEFAULT 1,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Insert test data
INSERT INTO users (name, email, active) VALUES
    ('Alice Johnson', 'alice@example.com', true),
    ('Bob Smith', 'bob@example.com', true),
    ('Charlie Brown', 'charlie@example.com', false),
    ('Diana Prince', 'diana@example.com', true),
    ('Eve Adams', 'eve@example.com', true);

INSERT INTO tasks (title, user_id, status, priority) VALUES
    ('Setup development environment', 1, 'completed', 1),
    ('Write API documentation', 1, 'in_progress', 2),
    ('Fix authentication bug', 2, 'pending', 3),
    ('Deploy to staging', 2, 'pending', 1),
    ('Review pull requests', 4, 'in_progress', 2),
    ('Update dependencies', 4, 'completed', 1),
    ('Database migration', 5, 'pending', 3);

-- Create test views for complex queries
CREATE VIEW active_users AS 
SELECT id, name, email FROM users WHERE active = true;

CREATE VIEW high_priority_tasks AS
SELECT t.id, t.title, u.name as assignee 
FROM tasks t 
JOIN users u ON t.user_id = u.id 
WHERE t.priority >= 2 AND t.status != 'completed';