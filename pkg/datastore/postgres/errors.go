package postgres

import (
	"errors"
	"fmt"

	"github.com/lib/pq"
)

const (
	DbConnectionFailedMsg        = "db connection failed"
	SettingNodeFailedMsg         = "unable to set db node"
    ForeignKeyViolationErrorCode = "23503"
)

var ErrHeaderDoesNotExist = errors.New("header does not exist")

func ErrDBConnectionFailed(connectErr error) error {
	return formatError(DbConnectionFailedMsg, connectErr.Error())
}

func ErrUnableToSetNode(setErr error) error {
	return formatError(SettingNodeFailedMsg, setErr.Error())
}

func formatError(msg, err string) error {
	return errors.New(fmt.Sprintf("%s: %s", msg, err))
}

func UnwrapErrorRecursively(err error) error {
	unwrapped := errors.Unwrap(err)
	if unwrapped != nil {
		return UnwrapErrorRecursively(unwrapped)
	}
	return err
}

func IsForeignKeyViolationErr(err error) bool {
	unwrappedErr := UnwrapErrorRecursively(err)
	pgErr, ok := unwrappedErr.(*pq.Error)
	if !ok {
		return false
	}

	return pgErr.Code == ForeignKeyViolationErrorCode
}
