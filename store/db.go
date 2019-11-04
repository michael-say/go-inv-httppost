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

// UserContext is a user Test Context
type UserContext struct {
	ID            int    `json:"id"`
	Name          string `json:"name"`
	Authorized    bool   `json:"authorized"`
	UserDiskQuota int64  `json:"userDiskQuota"`
}

// AppContext is an app test context
type AppContext struct {
	MaxUploadFileSize int64         `json:"maxUploadFileSize"`
	Users             []UserContext `json:"users"`
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

func getUserContext(appCtx *AppContext, userID int) (*UserContext, error) {
	for i := range appCtx.Users {
		if appCtx.Users[i].ID == userID {
			return &appCtx.Users[i], nil
		}
	}
	return nil, fmt.Errorf("Unknown user %d", userID)
}

func saveContext(adr *Address, ctx *AppContext) error {
	pwd, err := os.Getwd()
	if err != nil {
		return err
	}

	path := filepath.Join(pwd, dbFolder, adr.App)
	err = os.MkdirAll(path, os.ModePerm)
	if err != nil {
		return err
	}

	path = filepath.Join(path, "context.json")
	/*	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0600)
		if err != nil {
			return nil, err
		}
		defer f.Close()*/

	jsonBytes, err := json.Marshal(ctx)
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(path, jsonBytes, 0600)

	if err != nil {
		return err
	}

	return nil

}

func getContext(adr *Address) (*AppContext, error) {

	pwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	path := filepath.Join(pwd, dbFolder, adr.App)
	err = os.MkdirAll(path, os.ModePerm)
	if err != nil {
		return nil, err
	}

	path = filepath.Join(path, "context.json")
	exists, err := fileExists(path)
	if err != nil {
		return nil, err
	}

	if !exists {
		_, err := copy(filepath.Join(pwd, "store", "context.json"), path)
		if err != nil {
			return nil, err
		}
	}

	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0600)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	bytes, err := ioutil.ReadAll(f)
	if err != nil {
		return nil, err
	}

	c := AppContext{}
	err = json.Unmarshal(bytes, &c)
	return &c, nil
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

// SaveBin saves binary to database and returns it's guid
func SaveBin(adr *Address, r *io.Reader, filename string) (int64, string, error) {

	guid, err := uuid.NewUUID()
	if err != nil {
		return 0, "", err
	}

	pwd, err := os.Getwd()
	if err != nil {
		return 0, "", err
	}

	path := filepath.Join(pwd, dbFolder, adr.App, strconv.FormatInt(adr.WorkspaceID, 16))

	err = os.MkdirAll(path, os.ModePerm)
	if err != nil {
		return 0, "", err
	}

	f, err := os.OpenFile(filepath.Join(path, guid.String()), os.O_RDWR|os.O_CREATE, 0600)
	if err != nil {
		return 0, "", err
	}
	defer f.Close()

	fName, err := os.OpenFile(filepath.Join(path, guid.String()+".name"), os.O_RDWR|os.O_CREATE, 0600)
	defer fName.Close()
	_, err = fName.WriteString(filename)
	if err != nil && err != io.EOF {
		return 0, "", err
	}

	written, err := io.Copy(f, *r)

	if err != nil && err != io.EOF {
		return 0, "", err
	}

	return written, guid.String(), nil
}
