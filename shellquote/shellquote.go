// Copyright 2021 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Command shellquote echoes its arguments with minimal quoting to reproduce
// those arguments in a shell command.
package main

import (
	"fmt"
	"os"

	"github.com/kballard/go-shellquote"
)

func main() {
	fmt.Println(shellquote.Join(os.Args[1:]...))
}
