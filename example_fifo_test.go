package non_blocking_io_test

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
	"time"

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

	start := time.Now()
	buf := make([]byte, 100)
	n, err := output.Read(buf)

	if err != nil {
		panic(err)
	}

	if time.Now().Sub(start).Microseconds() > 100 {
		panic(fmt.Sprintf("Took more than 100 microseconds to read %d bytes", n))
	}

	fmt.Printf("Took less than 100 microseconds to read %d bytes: \"%s\"\n", n, strings.ReplaceAll(string(buf[:n]), "\n", "\\n"))

	// Second read will fail because no data is available.
	start = time.Now()
	buf = make([]byte, 100)
	n, err = output.Read(buf)

	if time.Now().Sub(start).Microseconds() > 100 {
		panic(fmt.Sprintf("Took more than 100 microseconds to read %d bytes", n))
	}

	fmt.Printf("Took less than 100 microseconds to read %d bytes\n", n)
	fmt.Printf("Expected timeout error - %v\n", err)

	// Output: Took less than 100 microseconds to read 4 bytes: "foo\n"
	// Took less than 100 microseconds to read 0 bytes
	// Expected timeout error - resource temporarily unavailable
}

// To write to the stdout of a sub-process, use the ``NewFifo`` function and pass the ``Fd`` pointer to the ``Stdin``
// parameter of the command.
func ExampleNewFifo_stdin() {
	var err error
	input, err := nbio.NewFifo()
	if err != nil {
		panic(err)
	}

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

	if time.Now().Sub(start).Microseconds() > 100 {
		panic(fmt.Sprintf("Took more than 100 microseconds to write %d bytes", n))
	}

	fmt.Printf("Took less than 100 microseconds to write %d bytes\n", n)

	// Second read will fail because no data is available.
	start = time.Now()
	n, err = input.Write(data)

	if time.Now().Sub(start).Microseconds() > 100 {
		panic(fmt.Sprintf("Took more than 100 microseconds to write %d bytes", n))
	}

	fmt.Printf("Took less than 100 microseconds to write %d bytes\n", n)
	fmt.Printf("Expected timeout error - %v\n", err)

	// Output: Took less than 100 microseconds to write 8192 bytes
	// Took less than 100 microseconds to write 0 bytes
	// Expected timeout error - resource temporarily unavailable
}
