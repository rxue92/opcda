package opcda

import (
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	ole "github.com/go-ole/go-ole"
)

var logger *log.Logger

// Default is no logger
func init() {
	logger = newLogger(io.Discard)
}

// Debug will set the logger to print to stderr
func Debug() {
	logger = newLogger(os.Stderr)
}

// SetLogWriter sets a user-defined writer for logger
func SetLogWriter(w io.Writer) {
	logger = newLogger(w)
}

// newLogger creats a log.Logger with standard settings
func newLogger(w io.Writer) *log.Logger {
	return log.New(w, "OPC ", log.LstdFlags)
}

func refineOleError(err error) error {
	if err == nil {
		return nil
	}
	oleError, ok := err.(*ole.OleError)
	if !ok {
		return err
	}
	return fmt.Errorf("code=%v, desc=%q, sub=[%s]", oleError.Code(), strings.TrimRight(oleError.Description(), "\r\n"), refineOleError(oleError.SubError()))
}
