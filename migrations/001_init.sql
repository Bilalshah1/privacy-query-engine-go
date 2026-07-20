

CREATE TABLE IF NOT EXISTS users (
    id               SERIAL PRIMARY KEY,
    age_group        TEXT NOT NULL,       -- e.g. '18-24', '25-34'
    department       TEXT NOT NULL,       -- e.g. 'engineering', 'sales'
    salary           NUMERIC(12, 2) NOT NULL,
    signup_date      DATE NOT NULL,
    last_active_date DATE NOT NULL
);

CREATE TABLE IF NOT EXISTS events (
    id         SERIAL PRIMARY KEY,
    user_id    INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    event_date DATE NOT NULL
);

CREATE TABLE IF NOT EXISTS purchases (
    id       SERIAL PRIMARY KEY,
    user_id  INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    category TEXT NOT NULL,               -- e.g. 'electronics', 'groceries'
    amount   NUMERIC(10, 2) NOT NULL
);

-- Indexes on the columns each query template groups by, since these
-- are aggregate queries that will run repeatedly against the whole
-- table.
CREATE INDEX IF NOT EXISTS idx_users_age_group    ON users (age_group);
CREATE INDEX IF NOT EXISTS idx_users_department   ON users (department);
CREATE INDEX IF NOT EXISTS idx_users_signup_date  ON users (signup_date);
CREATE INDEX IF NOT EXISTS idx_events_event_date  ON events (event_date);
CREATE INDEX IF NOT EXISTS idx_purchases_category ON purchases (category);