package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
)

func main() {
	fs, err := ioutil.ReadDir("rc")
	if err != nil {
		log.Fatal(err)
	}

	out, err := os.Create("rc.go")
	if err != nil {
		log.Fatal(err)
	}

	out.Write([]byte("package main \n\n"))
	out.Write([]byte("import \"archive/tar\"\n\n"))

	fmt.Fprintln(out, "func writeTarRC(w *tar.Writer) error {")

	for _, f := range fs {
		fmt.Fprintf(out, "\tif err := writeTarRecord(w, \"%s\", `", f.Name())
		f, _ := os.Open("rc/" + f.Name())
		io.Copy(out, f)
		out.Write([]byte("`); err != nil {\n\t\treturn err\n\t}\n"))
	}
	out.Write([]byte("\treturn nil\n}\n"))
}
