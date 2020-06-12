package file_system_test

import (
	"os"

	"github.com/makerdao/vulcanizedb/pkg/file_system"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("StatusWriter", func() {
	var (
		testFileName = "test-file"
		testFilePath = "/tmp/" + testFileName
		testFileContents = []byte("test contents")
	)
	AfterEach(func() {
		err := os.Remove(testFilePath)
		Expect(err).NotTo(HaveOccurred())
	})

	It("It writes the file with the given file name and content", func() {
		writer := file_system.NewStatusWriter(testFilePath, testFileContents)
		err := writer.Write()

		Expect(err).NotTo(HaveOccurred())
		info, err := os.Stat(testFilePath)
		Expect(err).NotTo(HaveOccurred())
		Expect(info.IsDir()).To(BeFalse())
		Expect(info.Name()).To(Equal(testFileName))
	})
})
