package main

import (
	"bufio"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"vkarmani-browser-helper/internal/app"
)

const allowedOrigin = "chrome-extension://bfcnangfjclakpliejpaamnalnjbmlgh/"

func readMessage(r *bufio.Reader) (app.Request, error) {
	var req app.Request
	var n uint32
	if err := binary.Read(r, binary.LittleEndian, &n); err != nil {
		return req, err
	}
	if n == 0 || n > 4<<20 {
		return req, fmt.Errorf("invalid native message size %d", n)
	}
	b := make([]byte, n)
	if _, err := io.ReadFull(r, b); err != nil {
		return req, err
	}
	if err := json.Unmarshal(b, &req); err != nil {
		return req, err
	}
	return req, nil
}

func writeMessage(w *bufio.Writer, resp app.Response) error {
	b, err := json.Marshal(resp)
	if err != nil {
		return err
	}
	if len(b) > 1<<20 {
		return fmt.Errorf("native response too large")
	}
	if err := binary.Write(w, binary.LittleEndian, uint32(len(b))); err != nil {
		return err
	}
	if _, err := w.Write(b); err != nil {
		return err
	}
	return w.Flush()
}

func callerOrigin() string {
	for _, arg := range os.Args[1:] {
		if strings.HasPrefix(arg, "chrome-extension://") {
			return arg
		}
	}
	return ""
}

func main() {
	origin := callerOrigin()
	if origin != "" && origin != allowedOrigin {
		fmt.Fprintln(os.Stderr, "native host rejected caller origin")
		os.Exit(3)
	}
	a, err := app.New()
	if err != nil {
		fmt.Fprintln(os.Stderr, "native host initialization failed")
		os.Exit(2)
	}
	defer a.Close()
	r := bufio.NewReader(os.Stdin)
	w := bufio.NewWriter(os.Stdout)
	for {
		req, err := readMessage(r)
		if err == io.EOF {
			return
		}
		if err != nil {
			fmt.Fprintln(os.Stderr, "native message read failed")
			return
		}
		if err := writeMessage(w, a.Handle(req)); err != nil {
			fmt.Fprintln(os.Stderr, "native message write failed")
			return
		}
	}
}
