package non_blocking_io_test

import (
	"fmt"
	"log"
	"os"

	"golang.org/x/sys/unix"

	nbio "github.com/kontera-technologies/non-blocking-io"
)

func Example() {
	fd, err := nbio.Open("testdata/foo.txt", unix.O_RDONLY, 0)
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}

	buf := make([]byte, 8192)
	n, err := fd.Read(buf)
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}
	fmt.Printf("Read %d bytes.\n", n)
	// Output: Read 8192 bytes.
}
