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
	address  *Address
	//mux      sync.Mutex
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
	pwd, err := os.Getwd()
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
		_, err := copy(filepath.Join(pwd, "store", filename), path)
		if err != nil {
			return "", err
		}
	}

	return path, nil
}

// ReadBin reads binary
func ReadBin(adr *Address, guid string) ([]byte, error) {
	pwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	path := filepath.Join(pwd, dbFolder, adr.App, strconv.FormatInt(adr.WorkspaceID, 16), guid)
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

func getBinaryWriter(adr *Address, filename string) (io.WriteCloser, string, error) {

	guid, err := uuid.NewUUID()
	guidStr := guid.String()
	if err != nil {
		return nil, guidStr, err
	}

	pwd, err := os.Getwd()
	if err != nil {
		return nil, guidStr, err
	}

	path := filepath.Join(pwd, dbFolder, adr.App, strconv.FormatInt(adr.WorkspaceID, 16))

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

func (k *JSONQuotaKeeper) registerAppSpace(appID string, space int64) error {
	return k.register(appID, "app", space)
}

func (k *JSONQuotaKeeper) registerUserSpace(userID int64, appID string, workspaceID int64, space int64) error {
	return k.register(appID, strconv.FormatInt(userID, 10), space)
}

func (k *JSONQuotaKeeper) getUserQuota(userID int64, appID string, workspaceID int64) (int64, error) {
	qq, error := k.readQuotas(appID)
	if error != nil {
		return 0, error
	}
	return qq[strconv.FormatInt(userID, 10)], nil
}

func (k *JSONQuotaKeeper) getAppQuota(appID string) (int64, error) {
	qq, error := k.readQuotas(appID)
	if error != nil {
		return 0, error
	}
	return qq["app"], nil
}

func newJSONQuotaKeeper(filename string, address *Address) *JSONQuotaKeeper {
	return &JSONQuotaKeeper{filename, address}
}
