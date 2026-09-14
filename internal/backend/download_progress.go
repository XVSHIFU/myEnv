package backend

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"path"
	"strings"
	"sync/atomic"
)

// DownloadProgress describes bytes read from an artifact response. Total is zero
// when response headers cannot describe those bytes reliably. Done clears the
// transfer even when the download fails; it does not claim successful validation.
type DownloadProgress struct {
	ID        uint64
	Label     string
	Completed int64
	Total     int64
	Done      bool
}

type downloadObserverKey struct{}
type downloadObserver struct {
	next   atomic.Uint64
	notify func(DownloadProgress)
}

// WithDownloadProgress observes downloads made with this context. The callback
// runs on the download goroutine and must only store/coalesce bounded state.
func WithDownloadProgress(ctx context.Context, notify func(DownloadProgress)) context.Context {
	if notify == nil {
		return ctx
	}
	return context.WithValue(ctx, downloadObserverKey{}, &downloadObserver{notify: notify})
}

type downloadReporter struct {
	observer *downloadObserver
	progress DownloadProgress
	started  bool
}

func beginDownload(ctx context.Context, address string) *downloadReporter {
	observer, _ := ctx.Value(downloadObserverKey{}).(*downloadObserver)
	if observer == nil {
		return nil
	}
	label := "archive"
	if parsed, err := url.Parse(address); err == nil {
		if name := path.Base(parsed.Path); name != "" && name != "." && name != "/" {
			label = strings.Map(func(r rune) rune {
				if r < 32 || r == 127 {
					return -1
				}
				return r
			}, name)
			if len(label) > 256 {
				label = strings.ToValidUTF8(label[:256], "")
			}
		}
	}
	return &downloadReporter{observer: observer, progress: DownloadProgress{ID: observer.next.Add(1), Label: label}}
}

func responseDownloadTotal(response *http.Response) int64 {
	encoding := response.Header.Get("Content-Encoding")
	if response.ContentLength <= 0 || response.Uncompressed || (encoding != "" && !strings.EqualFold(encoding, "identity")) {
		return 0
	}
	return response.ContentLength
}

func (d *downloadReporter) reader(source io.Reader, total int64) io.Reader {
	if d == nil {
		return source
	}
	d.started = true
	d.progress.Total = total
	d.observer.notify(d.progress)
	return &downloadProgressReader{Reader: source, reporter: d}
}

func (d *downloadReporter) finish() {
	if d == nil || !d.started || d.progress.Done {
		return
	}
	d.progress.Done = true
	d.observer.notify(d.progress)
}

type downloadProgressReader struct {
	io.Reader
	reporter *downloadReporter
}

func (r *downloadProgressReader) Read(p []byte) (int, error) {
	n, err := r.Reader.Read(p)
	if n > 0 {
		d := r.reporter
		d.progress.Completed += int64(n)
		if d.progress.Completed > d.progress.Total {
			d.progress.Total = 0
		}
		d.observer.notify(d.progress)
	}
	return n, err
}
