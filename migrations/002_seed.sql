

INSERT INTO users (age_group, department, salary, signup_date, last_active_date) VALUES
    ('18-24', 'engineering', 68000,  '2024-01-15', '2026-07-10'),
    ('18-24', 'engineering', 71000,  '2024-03-02', '2026-07-15'),
    ('18-24', 'sales',       55000,  '2024-02-20', '2026-05-01'),
    ('25-34', 'engineering', 95000,  '2023-11-05', '2026-07-18'),
    ('25-34', 'engineering', 102000, '2023-08-19', '2026-07-17'),
    ('25-34', 'sales',       78000,  '2024-01-30', '2026-06-20'),
    ('25-34', 'marketing',   72000,  '2023-12-11', '2026-07-01'),
    ('35-44', 'engineering', 135000, '2022-06-14', '2026-07-16'),
    ('35-44', 'sales',       88000,  '2023-03-22', '2026-04-10'),
    ('35-44', 'marketing',   91000,  '2022-09-09', '2026-07-12'),
    ('45-54', 'engineering', 148000, '2021-01-11', '2026-07-05'),
    ('45-54', 'sales',       97000,  '2021-07-23', '2026-03-15');

-- Events: a handful per user, spread across a few days, so
-- daily_event_counts has multiple groups to aggregate over.
INSERT INTO events (user_id, event_date)
SELECT u.id, d::date
FROM users u
CROSS JOIN generate_series('2026-07-14'::date, '2026-07-18'::date, '1 day') AS d
WHERE random() < 0.6;

-- Purchases: a few per user across a small set of categories, with
-- amounts bounded well under the $1000 sensitivity used for the
-- purchase_totals_by_category template.
INSERT INTO purchases (user_id, category, amount)
SELECT u.id, c.category, round((random() * 400 + 20)::numeric, 2)
FROM users u
CROSS JOIN (VALUES ('electronics'), ('groceries'), ('books'), ('travel')) AS c(category)
WHERE random() < 0.5;