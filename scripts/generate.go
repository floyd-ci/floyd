package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
)

func main() {
	data := createTar()
	data = compress(data)
	writeRC(data)
}

func createTar() []byte {
	var buf bytes.Buffer
	w := tar.NewWriter(&buf)
	fs, err := ioutil.ReadDir("rc")
	if err != nil {
		log.Fatal(err)
	}
	for _, f := range fs {
		err := w.WriteHeader(&tar.Header{
			Name:     f.Name(),
			Mode:     int64(f.Mode()),
			Size:     f.Size(),
			Typeflag: '0',
		})
		if err != nil {
			log.Fatal(err)
		}
		file, err := os.Open("rc/" + f.Name())
		if err != nil {
			log.Fatal(err)
		}
		defer file.Close()
		if _, err := io.Copy(w, file); err != nil {
			log.Fatal(err)
		}
	}
	if err := w.Close(); err != nil {
		log.Fatal(err)
	}
	return buf.Bytes()[:buf.Len()-2<<9]
}

func compress(data []byte) []byte {
	var buf bytes.Buffer
	w := gzip.NewWriter(&buf)
	if _, err := w.Write(data); err != nil {
		log.Fatal(err)
	}
	if err := w.Close(); err != nil {
		log.Fatal(err)
	}
	return buf.Bytes()
}

func writeRC(data []byte) {
	out, err := os.Create("rc.go")
	if err != nil {
		log.Fatal(err)
	}

	out.Write([]byte(`package main

import (
	"bytes"
	"compress/gzip"
	"io"
)

func tarRC(dst io.Writer) error {
	buf := bytes.NewBuffer([]byte{`))

	for i, c := range data {
		if i%12 == 0 {
			out.Write([]byte("\n\t\t"))
		} else {
			out.Write([]byte(" "))
		}
		fmt.Fprintf(out, "0x%02x,", c)
	}

	out.Write([]byte(`
	})
	r, err := gzip.NewReader(buf)
	if err != nil {
		return err
	}
	if _, err := io.Copy(dst, r); err != nil {
		return err
	}
	return r.Close()
}
`))
}
