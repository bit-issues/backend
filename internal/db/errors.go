package db

import (
	"github.com/go-sql-driver/mysql"
	"github.com/samber/lo"
)

const ErrCodeUniqueViolation = 1062

func IsUniqueViolation(err error) bool {
	if err == nil {
		return false
	}

	if mysqlErr, ok := lo.ErrorsAs[*mysql.MySQLError](err); ok {
		return mysqlErr.Number == ErrCodeUniqueViolation
	}
	return false
}
