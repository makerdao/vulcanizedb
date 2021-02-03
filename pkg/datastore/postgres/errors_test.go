package postgres_test

import (
	"fmt"

	"github.com/lib/pq"
	"github.com/makerdao/vulcanizedb/pkg/datastore/postgres"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)


var _ = Describe("Postgres errors", func() {
	var FKViolationErr = &pq.Error{
		Severity:         "ERROR",
		Code:             "23503",
	}

	Describe("UnwrapErrorRecursively", func() {
		It("unwraps a wrapped error", func() {
			e1 := TestError{}
			e2 := fmt.Errorf("error 2: %w", e1)
			e3 := fmt.Errorf("error 3: %w", e2)

			unwrappedE1 := postgres.UnwrapErrorRecursively(e1)
			Expect(unwrappedE1).To(Equal(e1))

			unwrappedE2 := postgres.UnwrapErrorRecursively(e2)
			Expect(unwrappedE2).To(Equal(e1))

			unwrappedE3 := postgres.UnwrapErrorRecursively(e3)
			Expect(unwrappedE3).To(Equal(e1))
		})
	})

	Describe("IsForeignKeyViolationErr", func() {
		It("returns true if the error is a pg.Error and a foreign_key_violation", func() {
			isFkViolation := postgres.IsForeignKeyViolationErr(FKViolationErr)
			Expect(isFkViolation).To(BeTrue())
		})

		It("returns false if the error is a pg.Error but not a foreign_key_violation", func() {
			var PqError = pq.Error{
				Severity: "ERROR",
				Code:     "02000",
			}

			isFkViolation := postgres.IsForeignKeyViolationErr(&PqError)
			Expect(isFkViolation).To(BeFalse())
		})

		It("returns false if the error is not a pg.Error", func() {
			testError := TestError{}
			isFkViolation := postgres.IsForeignKeyViolationErr(testError)
			Expect(isFkViolation).To(BeFalse())
		})

		It("determines if the error is a foreign_key_violation error when it's a wrapped error", func() {
			wrappedFKErr := fmt.Errorf("fk wrapped err: %w", FKViolationErr)
			isFkViolation := postgres.IsForeignKeyViolationErr(wrappedFKErr)
			Expect(isFkViolation).To(BeTrue())
		})
	})
})

type TestError struct{}
func (e TestError) Error() string {
	return "Test Error"
}

