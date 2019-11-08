package store

import (
	"io"
	"log"
)

type quotaProvider interface {
	registerSpace(userID int64, adr *Address, used int64) (int64, error)
	getUserQuota(userID int64, adr *Address) (int64, error)
	getAppQuota(appID string) (int64, error)
}

type quotaKeeper interface {
	registerUserSpace(userID int64, appID string, workspaceID int64, space int64) (int64, error)
	getUserQuota(userID int64, appID string, workspaceID int64) (int64, error)
	registerAppSpace(app string, space int64) (int64, error)
	getAppQuota(app string) (int64, error)
}

type storeQuotaProvider struct {
	keeper quotaKeeper
}

type quotaLimitedReader struct {
	appQuota  int64
	userQuota int64
	reader    io.Reader
	qp        *quotaProvider
	userID    int64
	adr       *Address
}

func (r *quotaLimitedReader) Read(p []byte) (n int, err error) {
	bytes := r.reader.Read(p)
	log.Println("Read ", len(p), "bytes")
	return bytes
}

func createProvider(quotaKeeper quotaKeeper) storeQuotaProvider {
	return storeQuotaProvider{
		keeper: quotaKeeper,
	}
}

func (p *storeQuotaProvider) getAppQuota(appID string) (int64, error) {
	return p.keeper.getAppQuota(appID)
}

func (p *storeQuotaProvider) getUserQuota(userID int64, adr *Address) (int64, error) {
	return p.keeper.getUserQuota(userID, adr.App, adr.WorkspaceID)
}

func (p *storeQuotaProvider) registerSpace(userID int64, adr *Address, used int64) (int64, error) {
	q, err := p.keeper.registerAppSpace(adr.App, used)
	if err == nil {
		q, err = p.keeper.registerUserSpace(userID, adr.App, adr.WorkspaceID, used)
	}
	return q, err
}

// QuotaReader returns the reader which requests for quota on every "Read" operation.
// Recommended max size of a buffer to read is up to 1 Mb
func QuotaReader(r io.Reader, q *quotaProvider, userID int64, adr *Address) io.Reader {
	return &quotaLimitedReader{r, q, userID, adr, int64(-1), int64(-1)}
}
