// Copyright 2023 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Command watchnet diagnoses open Cloud NAT connections.
package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"log"
	"maps"
	"net"
	"os/exec"
	"sort"
	"strings"
	"time"
)

type natPort struct {
	host    string
	port    string
	state   string
	program string
}

func main() {
	maxSeen := 0

	connsByRemote := map[string]map[string]bool{} // remote → local → true
	totalConns := 0

	for {
		cmd := exec.Command("ss", "--process", "--oneline", "--no-header", "--resolve", "--tcp", "--udp", "state", "connected", "! ( dst = localhost || dst = metadata )")
		cmd.Stderr = new(strings.Builder)
		out, err := cmd.StdoutPipe()
		if err != nil {
			log.Fatal(err)
		}
		if err := cmd.Start(); err != nil {
			log.Fatal(err)
		}

		nonNats := 0
		var nats []natPort

		prevUnique := totalConns
		br := bufio.NewReader(out)
		for {
			line, err := br.ReadSlice('\n')
			if err == io.EOF {
				break
			}
			if err != nil {
				log.Fatalf("%v:\nprocessing output: %v", cmd, err)
			}

			f := bytes.Fields(line)
			if len(f) < 6 {
				log.Fatalf("%v:\nunexpected short line: %q", cmd, line)
			}
			state := f[1]
			localAddr := f[4]
			remoteAddr := f[5]
			var program []byte
			if len(f) > 6 {
				program = f[6]
			}

			host, port, err := net.SplitHostPort(string(remoteAddr))
			if err != nil {
				log.Fatalf("%v:\nunexpected remoteAddr: %q", cmd, remoteAddr)
			}

			if strings.HasSuffix(host, ".1e100.net") {
				nonNats++
				continue
			}

			locals := connsByRemote[string(remoteAddr)]
			if locals == nil {
				locals = make(map[string]bool)
				connsByRemote[string(remoteAddr)] = locals
			}
			if !locals[string(localAddr)] {
				locals[string(localAddr)] = true
				totalConns++
			}

			nats = append(nats, natPort{
				host:    host,
				port:    port,
				state:   string(state),
				program: string(program),
			})
		}

		if err := cmd.Wait(); err != nil {
			log.Print(cmd.Stderr)
			log.Fatalf("%v: %v", cmd, err)
		}

		if len(nats) > maxSeen {
			fmt.Printf("%s\treached %d connected ports:\n", time.Now().Format(time.RFC3339), len(nats))
			for _, p := range nats {
				fmt.Printf("%s\t%s:%s\t%s\n", p.state, p.host, p.port, p.program)
			}
			fmt.Println()
			maxSeen = len(nats)
		}

		if totalConns > prevUnique {
			fmt.Printf("%s\nsaw %d connections with %d unique remote addrs", time.Now().Format(time.RFC3339), totalConns, len(connsByRemote))
			remotes := maps.Keys(connsByRemote)
			sort.Strings(remotes)
			for _, r := range remotes {
				fmt.Printf("%s\t%d\n", r, len(connsByRemote[r]))
			}
			fmt.Println()
		}
	}
}
