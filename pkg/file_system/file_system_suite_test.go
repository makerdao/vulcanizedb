package file_system_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestFileSystem(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "FileSystem Suite")
}
