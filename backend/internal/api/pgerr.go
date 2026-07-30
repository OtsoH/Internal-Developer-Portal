package api

import (
	"errors"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
)

// isUniqueViolation reports whether err is a unique-constraint violation raised
// by one specific constraint.
//
// The constraint name is what makes this useful. SQLSTATE 23505 alone is not
// enough: a mutation writes to services, tags and service_tags in one
// transaction, so a concurrent insert of the same tag raises 23505 on
// tags_name_key inside a request that has nothing wrong with its slug. Reporting
// that as a slug conflict would send the caller off to rename a field that was
// never the problem.
func isUniqueViolation(err error, constraint string) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) &&
		pgErr.Code == pgerrcode.UniqueViolation &&
		pgErr.ConstraintName == constraint
}
