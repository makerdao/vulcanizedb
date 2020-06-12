package file_system

import (
	"io/ioutil"
	"os"
)

type IStatusWriter interface {
	Write() error
}

type StatusWriter struct {
	file    string
	message []byte
	perm    os.FileMode
}

func NewStatusWriter(file string, message []byte) StatusWriter {
	return StatusWriter{
		file:    file,
		message: message,
		perm:    0644,
	}
}
func (w StatusWriter) Write() error {
	return ioutil.WriteFile(w.file, w.message, w.perm)
}
