package ostent
import (
	"io"
	"os"
	"log"
	"bufio"
	"strings"
)

func init() {
	lf := logFiltered{}
	var reader io.Reader
	reader, lf.writer = io.Pipe()
	lf.scanner = bufio.NewScanner(reader)
	go lf.read()
	log.SetOutput(&lf)
}

type logFiltered struct{
	writer  io.Writer
	scanner *bufio.Scanner
	ping chan bool
}

func (lf *logFiltered) Write(p []byte) (int, error) {
	return lf.writer.Write(p)
}

func (lf *logFiltered) read() {
	for {
		if !lf.scanner.Scan() {
			if err := lf.scanner.Err(); err != nil {
				log.New(os.Stderr, "", log.LstdFlags).Printf("bufio.Scanner.Scan Err: %s", err)
			}
			continue
		}
		text := lf.scanner.Text()
		if strings.Contains(text, " handling ") {
			continue
		}
		os.Stderr.WriteString(text +"\n")
	}
}










