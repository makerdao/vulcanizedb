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

type StatusAppender struct {
	file      string
	message   []byte
	perm      os.FileMode
	fileFlags int
}

func NewStatusAppender(file string, message []byte) StatusAppender {
	return StatusAppender{
		file:      file,
		message:   message,
		perm:      0644,
		fileFlags: os.O_APPEND | os.O_CREATE | os.O_WRONLY,
	}
}
func (a StatusAppender) Write() error {
	file, openErr := os.OpenFile(a.file, a.fileFlags, a.perm)
	if openErr != nil {
		return openErr
	}

	_, writeErr := file.Write(a.message)
	if writeErr != nil {
		closeErr := file.Close()
		if closeErr != nil {
			return closeErr
		}
		return writeErr
	}
	return file.Close()
}
