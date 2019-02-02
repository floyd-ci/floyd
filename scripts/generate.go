package main

import (
	"archive/tar"
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
)

func main() {
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
		_, err = io.Copy(w, file)
		if err != nil {
			log.Fatal(err)
		}
	}
	if err := w.Close(); err != nil {
		log.Fatal(err)
	}

	out, err := os.Create("rc.go")
	if err != nil {
		log.Fatal(err)
	}

	out.Write([]byte("package main\n\n"))
	out.Write([]byte("func tarRC() []byte {\n"))
	out.Write([]byte("\treturn []byte{"))

	for i, c := range buf.Bytes()[:buf.Len()-2<<9] {
		if i%12 == 0 {
			out.Write([]byte("\n\t\t"))
		} else {
			out.Write([]byte(" "))
		}
		fmt.Fprintf(out, "0x%02x,", c)
	}

	out.Write([]byte("\n\t}\n"))
	out.Write([]byte("}\n"))
}
