package file_system_test

import (
	"io/ioutil"
	"os"

	"github.com/makerdao/vulcanizedb/pkg/file_system"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("StatusWriter", func() {
	var (
		testFileName     = "test-file"
		testFilePath     = "/tmp/" + testFileName
		testFileContents = []byte("test contents\n")
	)
	AfterEach(func() {
		err := os.Remove(testFilePath)
		Expect(err).NotTo(HaveOccurred())
	})

	Describe("StatusWriter", func() {
		var writer = file_system.NewStatusWriter(testFilePath, testFileContents)

		It("It writes the contents to the given file path", func() {
			err := writer.Write()
			Expect(err).NotTo(HaveOccurred())

			info, err := os.Stat(testFilePath)
			Expect(err).NotTo(HaveOccurred())
			Expect(info.IsDir()).To(BeFalse())
			Expect(info.Name()).To(Equal(testFileName))

			contents, err := ioutil.ReadFile(testFilePath)
			Expect(err).NotTo(HaveOccurred())
			Expect(contents).To(Equal(testFileContents))
		})

		It("truncates the file contents", func() {
			err := writer.Write()
			Expect(err).NotTo(HaveOccurred())

			newFileContents := []byte("new file contents")
			writer2 := file_system.NewStatusWriter(testFilePath, newFileContents)
			err2 := writer2.Write()
			Expect(err2).NotTo(HaveOccurred())

			contents, err := ioutil.ReadFile(testFilePath)
			Expect(err).NotTo(HaveOccurred())
			Expect(contents).To(Equal(newFileContents))
		})
	})

	Describe("StatusAppender", func() {
		var writer = file_system.NewStatusAppender(testFilePath, testFileContents)

		It("It writes the contents to the given file path", func() {
			err := writer.Write()
			Expect(err).NotTo(HaveOccurred())

			info, err := os.Stat(testFilePath)
			Expect(err).NotTo(HaveOccurred())
			Expect(info.IsDir()).To(BeFalse())
			Expect(info.Name()).To(Equal(testFileName))

			contents, err := ioutil.ReadFile(testFilePath)
			Expect(err).NotTo(HaveOccurred())
			Expect(contents).To(Equal(testFileContents))
		})

		It("appends the content to the end of the given file", func() {
			err := writer.Write()
			Expect(err).NotTo(HaveOccurred())

			newFileContents := []byte("new file contents")
			writer2 := file_system.NewStatusAppender(testFilePath, newFileContents)
			err2 := writer2.Write()
			Expect(err2).NotTo(HaveOccurred())

			contents, err := ioutil.ReadFile(testFilePath)
			expectedContents := []byte(testFileContents)
			expectedContents = append(expectedContents, []byte(newFileContents)...)

			Expect(err).NotTo(HaveOccurred())
			Expect(contents).To(Equal(expectedContents))
		})
	})
})
