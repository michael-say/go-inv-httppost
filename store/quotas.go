package store

import (
	"errors"
	"fmt"
	"io"
	"log"
)

var errOutOfQuota = errors.New("Out of quota")

// KeeperSettings defines the way StoreKeeper reads its settings
type KeeperSettings = interface {
	MaxUploadSize(dest AppWorkspaceID) int64
	QuotaCacheSize() int64
}

// AppID identifies app
type AppID interface {
	AppID() string
}

// UserID identifies user
type UserID interface {
	UserID() int64
}

// WorkspaceID identifies workspace
type WorkspaceID interface {
	WorkspaceID() string
}

// AppWorkspaceID identifies app and workspace
type AppWorkspaceID interface {
	AppID
	WorkspaceID
}

type quotaManager interface {
	registerSpace(u UserID, w AppWorkspaceID, used int64) error
	getUserQuota(u UserID, w AppWorkspaceID) (int64, error)
	getAppQuota(a AppID) (int64, error)
	getSettings() KeeperSettings
}

type quotaProvider interface {
	registerUserSpace(u UserID, w AppWorkspaceID, space int64) error
	getUserQuota(u UserID, w AppWorkspaceID) (int64, error)
	registerAppSpace(a AppID, space int64) error
	getAppQuota(a AppID) (int64, error)
}

type storeQuotaManager struct {
	prov     quotaProvider
	settings KeeperSettings
}

// QuotaManagedWriter is a writer which checks app and user quotas and registers content in QuotaManager
// when quota is over, Write returns errOutOfQuota
type QuotaManagedWriter struct {
	writer  io.Writer
	qp      quotaManager
	user    UserID
	adr     AppWorkspaceID
	written int64
	quota   int64
	verbose bool
	total   int64
}

func (w *QuotaManagedWriter) count() (err error) {
	if w.written > 0 {
		if w.verbose {
			log.Println(fmt.Sprintf("Registering used space: %d bytes", w.written))
		}
		err = w.qp.registerSpace(w.user, w.adr, w.written)
		if err != nil {
			return err
		}
		w.written = 0
	}

	aquota, err := w.qp.getAppQuota(w.adr)
	if err != nil {
		return err
	}
	uquota, err := w.qp.getUserQuota(w.user, w.adr)
	if err != nil {
		return err
	}
	w.quota = min(min(aquota, uquota), w.qp.getSettings().QuotaCacheSize())
	if w.verbose {
		log.Println(fmt.Sprintf("Quota updated: %d (uq: %d, aq: %d) ", w.quota, uquota, aquota))
	}

	return nil
}

func (w *QuotaManagedWriter) Write(p []byte) (n int, err error) {

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

// Close validates that all written data is registered in QuotaManager
func (w *QuotaManagedWriter) Close() (err error) {
	if w.written > 0 {
		err = w.count()
	}
	return
}

// QuotaCounterWriter returns the writer which registers used space while sending data to some reader
func QuotaCounterWriter(w io.Writer, q quotaManager, user UserID, adr AppWorkspaceID, verbose bool) io.WriteCloser {
	return &QuotaManagedWriter{w, q, user, adr, 0, 0, verbose, 0}
}

func min(a int64, b int64) int64 {
	if a <= b {
		return a
	}
	return b
}

func createQuotaManager(provider quotaProvider, settings KeeperSettings) *storeQuotaManager {
	return &storeQuotaManager{provider, settings}
}

func (p *storeQuotaManager) getAppQuota(app AppID) (int64, error) {
	return p.prov.getAppQuota(app)
}

func (p *storeQuotaManager) getUserQuota(u UserID, w AppWorkspaceID) (int64, error) {
	return p.prov.getUserQuota(u, w)
}

func (p *storeQuotaManager) registerSpace(u UserID, w AppWorkspaceID, used int64) error {
	err := p.prov.registerAppSpace(w, used)
	if err == nil {
		err = p.prov.registerUserSpace(u, w, used)
	}
	return err
}

func (p *storeQuotaManager) getSettings() KeeperSettings {
	return p.settings
}
