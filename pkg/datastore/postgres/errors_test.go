package postgres_test

import (
	"fmt"

	"github.com/makerdao/vulcanizedb/pkg/datastore/postgres"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)


var _ = Describe("Postgres errors", func() {
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
})

type TestError struct{}
func (e TestError) Error() string {
	return "Test Error"
}

