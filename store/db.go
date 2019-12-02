package store

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"

	"github.com/google/uuid"
)

const (
	dbFolder = ".db"
)

// JSONQuotaKeeper keeps quotas in JSON file
type JSONQuotaKeeper struct {
	filename string
	address  AppWorkspaceID
	//mux      sync.Mutex
}

type dbUserID struct {
	id int64
}

type dbAppWorkspace struct {
	appID       string
	workspaceID string
}

type dbKeeperSettings struct {
}

func fileExists(clusterFile string) (bool, error) {
	_, err := os.Stat(clusterFile)
	if err != nil && os.IsNotExist(err) {
		return false, nil
	} else if err != nil {
		return false, err
	}
	return true, nil
}

func copy(src, dst string) (int64, error) {
	sourceFileStat, err := os.Stat(src)
	if err != nil {
		return 0, err
	}

	if !sourceFileStat.Mode().IsRegular() {
		return 0, fmt.Errorf("%s is not a regular file", src)
	}

	source, err := os.Open(src)
	if err != nil {
		return 0, err
	}
	defer source.Close()

	destination, err := os.Create(dst)
	if err != nil {
		return 0, err
	}
	defer destination.Close()
	nBytes, err := io.Copy(destination, source)
	return nBytes, err
}

func getJSONPath(appID string, filename string) (string, error) {
	pwd, err := getHome()
	if err != nil {
		return "", err
	}

	path := filepath.Join(pwd, dbFolder, appID)
	err = os.MkdirAll(path, os.ModePerm)
	if err != nil {
		return "", err
	}

	path = filepath.Join(path, filename)
	exists, err := fileExists(path)
	if err != nil {
		return "", err
	}

	if !exists {
		_, err := copy(filepath.Join(pwd, "resources", "templates", filename), path)
		if err != nil {
			return "", err
		}
	}

	return path, nil
}

// ReadBin reads binary
func ReadBin(adr AppWorkspaceID, guid string) ([]byte, error) {
	pwd, err := getHome()
	if err != nil {
		return nil, err
	}

	path := filepath.Join(pwd, dbFolder, adr.AppID(), adr.WorkspaceID(), guid)
	f, err := os.OpenFile(path, os.O_RDONLY, 0600)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	bytes, err := ioutil.ReadAll(f)
	if err != nil {
		return nil, err
	}
	return bytes, nil
}

func getBinaryWriter(adr AppWorkspaceID, filename string) (io.WriteCloser, string, error) {

	guid, err := uuid.NewUUID()
	guidStr := guid.String()
	if err != nil {
		return nil, guidStr, err
	}

	pwd, err := getHome()
	if err != nil {
		return nil, guidStr, err
	}

	path := filepath.Join(pwd, dbFolder, adr.AppID(), adr.WorkspaceID())

	err = os.MkdirAll(path, os.ModePerm)
	if err != nil {
		return nil, guidStr, err
	}

	fName, err := os.OpenFile(filepath.Join(path, guid.String()+".name"), os.O_RDWR|os.O_CREATE, 0600)
	defer fName.Close()
	_, err = fName.WriteString(filename)
	if err != nil && err != io.EOF {
		return nil, guidStr, err
	}

	f, err := os.OpenFile(filepath.Join(path, guid.String()), os.O_RDWR|os.O_CREATE, 0600)
	if err != nil {
		return nil, guidStr, err
	}

	return f, guidStr, nil
}

func (k *JSONQuotaKeeper) readQuotas(appID string) (map[string]int64, error) {
	filename, err := getJSONPath(appID, k.filename)
	if err != nil {
		return nil, err
	}

	f, err := os.OpenFile(filename, os.O_RDWR|os.O_CREATE, 0600)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	bytes, err := ioutil.ReadAll(f)
	if err != nil {
		return nil, err
	}

	m := make(map[string]int64)
	err = json.Unmarshal(bytes, &m)
	return m, nil
}

func (k *JSONQuotaKeeper) saveQuotas(appID string, qq map[string]int64) error {
	jsonBytes, err := json.Marshal(qq)
	if err != nil {
		return err
	}
	filename, err := getJSONPath(appID, k.filename)
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(filename, jsonBytes, 0600)

	if err != nil {
		return err
	}
	return nil
}

func (k *JSONQuotaKeeper) register(appID string, id string, space int64) error {
	qq, error := k.readQuotas(appID)
	if error != nil {
		return error
	}
	qq[id] = qq[id] - space
	return k.saveQuotas(appID, qq)
}

func (k *JSONQuotaKeeper) registerAppSpace(a AppID, space int64) error {
	return k.register(a.AppID(), "app", space)
}

func (k *JSONQuotaKeeper) registerUserSpace(u UserID, w AppWorkspaceID, space int64) error {
	return k.register(w.AppID(), strconv.FormatInt(u.UserID(), 10), space)
}

func (k *JSONQuotaKeeper) getUserQuota(u UserID, w AppWorkspaceID) (int64, error) {
	qq, error := k.readQuotas(w.AppID())
	if error != nil {
		return 0, error
	}
	return qq[strconv.FormatInt(u.UserID(), 10)], nil
}

func (k *JSONQuotaKeeper) getAppQuota(a AppID) (int64, error) {
	qq, error := k.readQuotas(a.AppID())
	if error != nil {
		return 0, error
	}
	return qq["app"], nil
}

func isAuthorized(u UserID) bool {
	return u.UserID() == 1 || u.UserID() == 2
}

func newJSONQuotaKeeper(filename string, address AppWorkspaceID) *JSONQuotaKeeper {
	return &JSONQuotaKeeper{filename, address}
}

func (u *dbUserID) UserID() int64 {
	return u.id
}

func (a *dbAppWorkspace) AppID() string {
	return a.appID
}

func (a *dbAppWorkspace) WorkspaceID() string {
	return a.workspaceID
}

func (s *dbKeeperSettings) MaxUploadSize(dest AppWorkspaceID) int64 {
	return 100 << 20 // 100Mb
}
func (s *dbKeeperSettings) QuotaCacheSize() int64 {
	return 1 << 20 // 1 Mb
}
