package non_blocking_io_test

import (
	"fmt"
	"log"
	"os/exec"
	"time"

	nbio "github.com/kontera-technologies/non-blocking-io"
)

func ExampleNonBlockingStdout() {
	var err error
	rw, err := nbio.NewFifo()
	if err != nil {
		log.Fatal(err)
	}

	// testdata/foo.sh will output "foo" every 100 milliseconds.
	cmd := exec.Command("testdata/foo.sh")
	cmd.Stdout = rw
	err = cmd.Start()
	if err != nil {
		log.Fatal(err)
	}
	defer cmd.Process.Kill()

	// Wait for a few milliseconds to give the process time to start, to make sure first read will succeed.
	time.Sleep(time.Millisecond * 100)

	start := time.Now()
	buf := make([]byte, 100)
	n, err := rw.Read(buf)

	log.Printf("Took %s to read %d bytes", time.Now().Sub(start), n)
	log.Printf("Error - %v", err)
	fmt.Println(string(buf[:n]))

	// Second read will fail because no data is available.
	start = time.Now()
	buf = make([]byte, 100)
	n, err = rw.Read(buf)

	log.Printf("Took %s to read %d bytes", time.Now().Sub(start), n)
	log.Printf("Expected Error - %v", err)
}