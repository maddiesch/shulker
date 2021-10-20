package cargo

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"runtime"
	"strconv"

	"github.com/cheggaaa/pb/v3"
)

var HTTPClient = http.DefaultClient
var Log = log.New(os.Stderr, "[cargo] ", log.LstdFlags|log.Lmsgprefix)

func Download(ctx context.Context, source string, dest string) error {
	done := make(chan error, 1)

	go func() {
		defer close(done)

		fail := func(err error) {
			done <- err
			runtime.Goexit()
		}

		tempFile, err := os.CreateTemp("", "cargo-*")
		if err != nil {
			fail(err)
		}
		defer func() {
			tempFile.Close()
			os.Remove(tempFile.Name())
		}()

		req, err := http.NewRequest("GET", source, nil)
		if err != nil {
			fail(err)
		}

		Log.Printf("Download %s", req.URL.String())

		resp, err := HTTPClient.Do(req)
		if err != nil {
			fail(err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			fail(&ResponseError{resp.StatusCode})
		}

		contentLen, _ := strconv.ParseInt(resp.Header.Get("Content-Length"), 10, 64)

		progress := &ProgressWriter{
			bar: pb.Start64(contentLen),
		}
		defer progress.bar.Finish()

		if _, err := io.Copy(tempFile, io.TeeReader(resp.Body, progress)); err != nil {
			fail(err)
		}

		tempFile.Close()

		if err := os.Rename(tempFile.Name(), dest); err != nil {
			fail(err)
		}
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-done:
		return err
	}
}

type ProgressWriter struct {
	bar *pb.ProgressBar
}

func (p *ProgressWriter) Write(b []byte) (int, error) {
	n := len(b)

	p.bar.Add(n)

	return n, nil
}

type ResponseError struct {
	StatusCode int
}

func (e ResponseError) Error() string {
	return fmt.Sprintf("http response error (%s)", http.StatusText(e.StatusCode))
}
