package non_blocking_io_test

import (
	"bytes"
	"fmt"
	"log"
	"os/exec"
	"strings"
	"time"

	"golang.org/x/sys/unix"

	nbio "github.com/kontera-technologies/non-blocking-io"
)

// To read from the stdout of a sub-process, use the ``NewFifo`` function and pass the ``Fd`` pointer to the ``Stdout``
// parameter of the command.
func ExampleNewFifo_stdout() {
	var err error
	output, err := nbio.NewFifo()
	if err != nil {
		panic(err)
	}
	defer output.Close()

	// testdata/endless-foo.sh will output "foo" every 100 milliseconds.
	cmd := exec.Command("testdata/endless-foo.sh")
	cmd.Stdout = output
	err = cmd.Start()
	if err != nil {
		panic(err)
	}
	defer cmd.Process.Kill()

	// Wait for a few milliseconds to give the process time to start, to make sure first read will succeed.
	time.Sleep(time.Millisecond * 100)

	buf := make([]byte, 100)
	start := time.Now()
	n, err := output.Read(buf)

	if err != nil {
		panic(err)
	}

	if dur := time.Since(start); dur.Microseconds() > 500 {
		panic(fmt.Sprintf("Took %d microseconds to read %d bytes", dur.Microseconds(), n))
	}

	fmt.Printf("Took less than 500 microseconds to read %d bytes: \"%s\".\n", n, strings.ReplaceAll(string(buf[:n]), "\n", "\\n"))

	// Second read will fail because no data is available.
	buf = make([]byte, 100)
	start = time.Now()
	n, err = output.Read(buf)

	if dur := time.Since(start); dur.Microseconds() > 500 {
		panic(fmt.Sprintf("Took %d microseconds to read %d bytes", dur.Microseconds(), n))
	}

	fmt.Printf("Took less than 500 microseconds to read %d bytes.\n", n)
	fmt.Printf("Expected timeout error - %v.\n", err)

	// third read will wait until data is available.
	buf = make([]byte, 100)
	start = time.Now()
	n, err = output.SelectRead(buf, unix.Timeval{Usec: 100000}) // 100ms = 100000μs
	if err != nil {
		log.Println(err)
	}
	if dur := time.Since(start); !(dur.Milliseconds() < 110 && dur.Milliseconds() > 90) {
		panic(fmt.Sprintf("Took %s milliseconds to read %d bytes.", dur, n))
	}

	fmt.Printf("Took about 100 milliseconds to read %d bytes: \"%s\".\n", n, strings.ReplaceAll(string(buf[:n]), "\n", "\\n"))

	// Output: Took less than 500 microseconds to read 4 bytes: "foo\n".
	// Took less than 500 microseconds to read 0 bytes.
	// Expected timeout error - resource temporarily unavailable.
	// Took about 100 milliseconds to read 4 bytes: "foo\n".
}

// To write to the stdout of a sub-process, use the ``NewFifo`` function and pass the ``Fd`` pointer to the ``Stdin``
// parameter of the command.
func ExampleNewFifo_stdin() {
	var err error
	input, err := nbio.NewFifo()
	if err != nil {
		panic(err)
	}
	defer input.Close()

	cmd := exec.Command("testdata/endless-read.sh")
	cmd.Stdin = input
	err = cmd.Start()
	if err != nil {
		panic(err)
	}
	defer cmd.Process.Kill()

	// Wait for a few milliseconds to give the process time to start, to make sure first write will succeed.
	time.Sleep(time.Millisecond * 100)

	data := append(bytes.Repeat([]byte("foo"), 100000), '\n')

	start := time.Now()
	n, err := input.Write(data)

	if err != nil {
		panic(err)
	}

	if dur := time.Since(start); dur.Microseconds() > 500 {
		panic(fmt.Sprintf("Took %d microseconds to write %d bytes", dur.Microseconds(), n))
	}
	if n >= len(data) {
		panic(fmt.Sprintf("Took less than 500 microseconds to write %d bytes", n))
	}

	fmt.Printf("Took less than 500 microseconds to write less than %d bytes\n", len(data))

	// Second read will fail because no data is available.
	start = time.Now()
	n, err = input.Write(data)

	if dur := time.Since(start); dur.Microseconds() > 500 {
		panic(fmt.Sprintf("Took %d microseconds to write %d bytes", dur.Microseconds(), n))
	}
	if n > 0 {
		panic(fmt.Sprintf("Took less than 500 microseconds to write %d bytes", n))
	}

	fmt.Printf("Took less than 500 microseconds to write 0 bytes\n")
	fmt.Printf("Expected timeout error - %v\n", err)

	// Output: Took less than 500 microseconds to write less than 300001 bytes
	// Took less than 500 microseconds to write 0 bytes
	// Expected timeout error - resource temporarily unavailable
}
