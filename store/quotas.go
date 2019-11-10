package store

import (
	"errors"
	"fmt"
	"io"
	"log"
)

const (
	quotaCacheSize = 1 << 20 // 1 Mb
)

var errOutOfQuota = errors.New("Out of quota")

type quotaProvider interface {
	registerSpace(userID int64, adr *Address, used int64) error
	getUserQuota(userID int64, adr *Address) (int64, error)
	getAppQuota(appID string) (int64, error)
}

type quotaKeeper interface {
	registerUserSpace(userID int64, appID string, workspaceID int64, space int64) error
	getUserQuota(userID int64, appID string, workspaceID int64) (int64, error)
	registerAppSpace(appID string, space int64) error
	getAppQuota(appID string) (int64, error)
}

type storeQuotaProvider struct {
	keeper quotaKeeper
}

type quotaLimitedReader struct {
	reader io.Reader
	qp     quotaProvider
	userID int64
	adr    *Address
	quota  int64
	read   int64
}

type quotaCounterWriter struct {
	writer  io.Writer
	qp      quotaProvider
	userID  int64
	adr     *Address
	written int64
	quota   int64
	verbose bool
	total   int64
}

func (w *quotaCounterWriter) count() (err error) {
	if w.written > 0 {
		if w.verbose {
			log.Println(fmt.Sprintf("Registering used space: %d bytes", w.written))
		}
		err = w.qp.registerSpace(w.userID, w.adr, w.written)
		if err != nil {
			return err
		}
		w.written = 0
	}

	aquota, err := w.qp.getAppQuota(w.adr.App)
	if err != nil {
		return err
	}
	uquota, err := w.qp.getUserQuota(w.userID, w.adr)
	if err != nil {
		return err
	}
	w.quota = min(min(aquota, uquota), quotaCacheSize)
	if w.verbose {
		log.Println(fmt.Sprintf("Quota updated: %d (uq: %d, aq: %d) ", w.quota, uquota, aquota))
	}

	return nil
}

func (w *quotaCounterWriter) Write(p []byte) (n int, err error) {

	if w.written >= w.quota {
		err = w.count()
		if err != nil {
			return 0, err
		}
	}

	if w.quota <= 0 {
		return 0, errOutOfQuota
	}

	maxWrite := w.quota - w.written
	if maxWrite < int64(len(p)) {
		var p2 []byte = p[0:maxWrite]
		n, err = w.writer.Write(p2)
	} else {
		n, err = w.writer.Write(p)
	}

	w.written += int64(n)
	w.total += int64(n)
	if w.verbose {
		log.Println(fmt.Sprintf("Part written %d/%d bytes (quota: %d/%d), total read: %d bytes", n, len(p), w.written, w.quota, w.total))
	}
	return
}

func (w *quotaCounterWriter) Close() (err error) {
	if w.written > 0 {
		err = w.count()
	}
	return
}

// QuotaCounterWriter returns the writer which registers used space while sending data to some reader
func QuotaCounterWriter(w io.Writer, q quotaProvider, userID int64, adr *Address, verbose bool) io.WriteCloser {
	return &quotaCounterWriter{w, q, userID, adr, 0, 0, verbose, 0}
}

func min(a int64, b int64) int64 {
	if a <= b {
		return a
	}
	return b
}

func (r *quotaLimitedReader) Read(p []byte) (n int, err error) {
	if r.read >= r.quota {
		aquota, err := r.qp.getAppQuota(r.adr.App)
		if err != nil {
			return 0, err
		}
		uquota, err := r.qp.getUserQuota(r.userID, r.adr)
		if err != nil {
			return 0, err
		}
		r.quota += min(min(aquota, uquota), quotaCacheSize)
		log.Println(fmt.Sprintf("Quota: %d (uq: %d, aq: %d) ", r.quota, uquota, aquota))
	}

	if r.read >= r.quota {
		return 0, io.EOF
	}

	maxRead := r.quota - r.read
	if int64(len(p)) > maxRead {
		p = p[0:maxRead]
	}

	n, err = r.reader.Read(p)
	r.read += int64(n)

	/*	err = r.qp.registerSpace(r.userID, r.adr, n)
		if err != nil {
			return n, err
		} */

	//log.Println(fmt.Sprintf("Read %d/%d bytes (q: %d), total read: %d Mb/s", n, len(p), r.quota, r.read))
	return
}

func createProvider(quotaKeeper quotaKeeper) *storeQuotaProvider {
	return &storeQuotaProvider{
		keeper: quotaKeeper,
	}
}

func (p *storeQuotaProvider) getAppQuota(appID string) (int64, error) {
	return p.keeper.getAppQuota(appID)
}

func (p *storeQuotaProvider) getUserQuota(userID int64, adr *Address) (int64, error) {
	return p.keeper.getUserQuota(userID, adr.App, adr.WorkspaceID)
}

func (p *storeQuotaProvider) registerSpace(userID int64, adr *Address, used int64) error {
	err := p.keeper.registerAppSpace(adr.App, used)
	if err == nil {
		err = p.keeper.registerUserSpace(userID, adr.App, adr.WorkspaceID, used)
	}
	return err
}

// QuotaLimitedReader returns the reader which requests for quota on every "Read" operation.
// Recommended max size of a buffer to read is up to 1 Mb
func QuotaLimitedReader(r io.Reader, q quotaProvider, userID int64, adr *Address) io.Reader {
	return &quotaLimitedReader{r, q, userID, adr, 0, 0}
}
