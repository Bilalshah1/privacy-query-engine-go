package query

import (
	"context"
	"database/sql"
)

// Row is one grouped result row: a label (e.g. an age group or
// department name) and the true numeric value before any noise is
// applied. The true value never leaves this package on its own — a
// caller must add DP noise before returning it over the API.
type Row struct {
	Label string
	Value float64
}

// Execute runs the named whitelisted template against the database and
// returns the true (un-noised) result rows. Callers must add Laplace
// or Gaussian noise before this data is serialized into an HTTP
// response — this function intentionally does not do that itself, to
// keep "fetch the truth" and "protect the truth" as separate,
// independently testable steps.
func Execute(ctx context.Context, db *sql.DB, templateName string) ([]Row, error) {
	tmpl, err := Lookup(templateName)
	if err != nil {
		return nil, err
	}

	rows, err := db.QueryContext(ctx, tmpl.SQL)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Row
	for rows.Next() {
		var label string
		var value sql.NullFloat64
		if err := rows.Scan(&label, &value); err != nil {
			return nil, err
		}
		out = append(out, Row{Label: label, Value: value.Float64})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}