// Package query holds the pre-approved query whitelist. Arbitrary
// user-supplied SQL is never allowed: computing the sensitivity of an
// arbitrary query is an unsolved problem in general, and an
// incorrectly-bounded sensitivity silently breaks the privacy
// guarantee even though the code runs fine. So instead, every query a
// caller can run is one of a small, fixed set of templates with a
// sensitivity value that's been worked out by hand ahead of time.
package query

import "fmt"

// Template describes one pre-approved analytics query: the SQL to run
// against Postgres, and the sensitivity bound needed to calibrate
// noise correctly for it.
type Template struct {
	Name        string
	Description string
	SQL         string
	// Sensitivity is the maximum amount a single row (person) could
	// change this query's result by. This must be re-derived by hand
	// any time the SQL changes — it is not something the engine can
	// infer automatically.
	Sensitivity float64
}

// Registry is the fixed set of queries callers may run. Nothing else
// is reachable through the API.
var Registry = map[string]Template{
	"count_users_by_age": {
		Name:        "count_users_by_age",
		Description: "Count of users grouped by age group",
		SQL: `SELECT age_group, COUNT(*) AS value
		      FROM users
		      GROUP BY age_group
		      ORDER BY age_group`,
		// One person can appear in exactly one age-group row, and can
		// change that row's count by at most 1.
		Sensitivity: 1,
	},
	"avg_salary_by_department": {
		Name:        "avg_salary_by_department",
		Description: "Average salary grouped by department",
		SQL: `SELECT department, AVG(salary) AS value
		      FROM users
		      GROUP BY department
		      ORDER BY department`,
		// Bounded by the maximum plausible individual salary. If one
		// person could earn arbitrarily much, this query would have
		// unbounded sensitivity and could not be safely answered.
		Sensitivity: 50000,
	},
	"daily_event_counts": {
		Name:        "daily_event_counts",
		Description: "Count of events grouped by day",
		SQL: `SELECT event_date, COUNT(*) AS value
		      FROM events
		      GROUP BY event_date
		      ORDER BY event_date`,
		// One event row = one person's action; adding/removing one
		// event changes a single day's count by at most 1.
		Sensitivity: 1,
	},
	"purchase_totals_by_category": {
		Name:        "purchase_totals_by_category",
		Description: "Sum of purchase amounts grouped by category",
		SQL: `SELECT category, SUM(amount) AS value
		      FROM purchases
		      GROUP BY category
		      ORDER BY category`,
		// Bounded by the maximum plausible single purchase amount.
		Sensitivity: 1000,
	},
	"user_cohort_retention": {
		Name:        "user_cohort_retention",
		Description: "Count of retained users grouped by signup cohort",
		SQL: `SELECT date_trunc('month', signup_date) AS cohort,
		             COUNT(*) FILTER (WHERE last_active_date >= signup_date + INTERVAL '30 days') AS value
		      FROM users
		      GROUP BY cohort
		      ORDER BY cohort`,
		// One user can only ever count as one retained/not-retained
		// row in one cohort.
		Sensitivity: 1,
	},
}

// Lookup returns the template for a given name, or an error if the
// name isn't in the whitelist. This is the only path by which a query
// name from an HTTP request is turned into SQL — there is no
// alternative code path that accepts raw SQL from a caller.
func Lookup(name string) (Template, error) {
	t, ok := Registry[name]
	if !ok {
		return Template{}, fmt.Errorf("unknown query template %q: only whitelisted templates are allowed", name)
	}
	return t, nil
}