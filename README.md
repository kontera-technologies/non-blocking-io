# Non Blocking IO
[![GoDoc](https://godoc.org/github.com/kontera-technologies/non-blocking-io?status.svg)](https://godoc.org/github.com/kontera-technologies/non-blocking-io)
[![codecov](https://codecov.io/gh/kontera-technologies/non-blocking-io/branch/master/graph/badge.svg)](https://codecov.io/gh/kontera-technologies/non-blocking-io)
[![Build Status](https://travis-ci.org/kontera-technologies/non-blocking-io.svg?branch=master)](https://travis-ci.org/kontera-technologies/non-blocking-io)

Provides a reader and writer that will return an error if the file descriptor is not ready for reading / writing
(respectfully).

## Installation
    go get github.com/kontera-technologies/non-blocking-io

## Usage
```go

package main
import nbio "github.com/kontera-technologies/non-blocking-io"

```