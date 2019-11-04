package store

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"strconv"
	"strings"
)

func getAddress(r *http.Request) (*Address, error) {
	parts := strings.Split(r.URL.Path[1:], "/")
	if len(parts) != 3 {
		return nil, fmt.Errorf("Unexpected path: %s", r.URL.Path[1:])
	}
	wid, err := strconv.ParseInt(parts[2], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("Unexpected value for workspaceId: %s", parts[2])
	}
	return &Address{
		App:         parts[1],
		WorkspaceID: wid,
	}, nil
}

func readUserCtx(reader *multipart.Reader, appCtx *AppContext) (*UserContext, string, int) {
	userIDBuf := make([]byte, 512)
	p, err := reader.NextPart()
	if err != nil {
		return nil, "Expected form field", http.StatusBadRequest
	}
	if p.FormName() != userIDFileFieldName {
		return nil, fmt.Sprintf("\"%s\" field is expected", userIDFileFieldName), http.StatusBadRequest
	}
	_, err = p.Read(userIDBuf)
	if err != nil && err != io.EOF {
		return nil, err.Error(), http.StatusInternalServerError
	}
	userID, err := strconv.ParseInt(strings.TrimRight(string(userIDBuf), "\x00"), 10, 32)
	if err != nil {
		return nil, err.Error(), http.StatusInternalServerError
	}
	userCtx, err := getUserContext(appCtx, int(userID))
	if err != nil {
		return nil, err.Error(), http.StatusForbidden
	}
	return userCtx, "", http.StatusOK
}

func fail(w http.ResponseWriter, err string, status int) {
	http.Error(w, err, status)
	log.Println("Error", status, err)
}

func postHandler(w http.ResponseWriter, r *http.Request) {
	address, err := getAddress(r)
	if err != nil {
		fail(w, "Request parse error: "+err.Error(), http.StatusBadRequest)
		return
	}

	appCtx, err := getContext(address)
	if err != nil {
		fail(w, "Internal server error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, appCtx.MaxUploadFileSize)
	reader, err := r.MultipartReader()

	userCtx, errstr, status := readUserCtx(reader, appCtx)

	if len(errstr) > 0 {
		fail(w, errstr, status)
		return
	}

	if !userCtx.Authorized {
		fail(w, "Not authorized", http.StatusForbidden)
		return
	}

	resp := PostResponse{
		DiskQuota: 0,
		Result:    make([]PostResponseItem, 0),
	}

	p, err := reader.NextPart()
	if err != nil {
		fail(w, fmt.Sprintf("Form field is expected %s", postFileFieldName), http.StatusBadRequest)
		return
	}

	for {

		if p.FormName() != postFileFieldName {
			fail(w, fmt.Sprintf("\"%s\" field is expected", postFileFieldName), http.StatusBadRequest)
			return
		}

		if userCtx.UserDiskQuota < chunkSize {
			fail(w, "User disk quota overlimit", http.StatusRequestEntityTooLarge)
			return
		}

		buf := bufio.NewReader(p)
		sniff, _ := buf.Peek(chunkSize)
		contentType := http.DetectContentType(sniff)
		log.Println(fmt.Sprintf("File: %s; Content type: %s", p.FileName(), contentType))
		if !isContentTypeAllowed(contentType) {
			fail(w, "file type not allowed: "+contentType, http.StatusBadRequest)
			return
		}

		log.Println("Limiting reader to " + strconv.Itoa(int(userCtx.UserDiskQuota+1)) + " bytes")
		lmt := io.LimitReader(buf, userCtx.UserDiskQuota+1)
		written, itemGUID, err := SaveBin(address, &lmt, p.FileName())
		log.Println("written: " + strconv.Itoa(int(written)) + " bytes")

		if err != nil {
			fail(w, "Internal server error: "+err.Error(), http.StatusInternalServerError)
			return
		}

		userCtx.UserDiskQuota = userCtx.UserDiskQuota - written
		err = saveContext(address, appCtx)
		if err != nil {
			fail(w, "Internal server error: "+err.Error(), http.StatusInternalServerError)
			return
		}
		if userCtx.UserDiskQuota < 0 {
			fail(w, "User disk quota overlimit", http.StatusRequestEntityTooLarge)
			return
		}

		resp.Result = append(resp.Result, PostResponseItem{
			FileName: p.FileName(),
			GUID:     itemGUID,
		})
		resp.DiskQuota = userCtx.UserDiskQuota

		p, err = reader.NextPart()
		if err != nil {
			break
		}
	}
	jsonBytes, err := json.Marshal(resp)
	if err != nil {
		fail(w, "Internal server error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	//
	fmt.Fprint(w, string(jsonBytes))
}

func getHandler(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(r.URL.Path[1:], "/")
	if len(parts) != 4 {
		fail(w, fmt.Sprintf("Unexpected path: %s", r.URL.Path[1:]), http.StatusBadRequest)
		return
	}
	wid, err := strconv.ParseInt(parts[2], 10, 64)
	if err != nil {
		fail(w, fmt.Sprintf("Unexpected value for workspaceId: %s", parts[2]), http.StatusBadRequest)
		return
	}
	addr := Address{
		App:         parts[1],
		WorkspaceID: wid,
	}

	bytes, err := ReadBin(&addr, parts[3])
	w.Write(bytes)
}

// BinHandler handles POST requests to /store path
// http://localhost/bin/app/workspace/
func BinHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		log.Println("HTTP POST", r.UserAgent())
		postHandler(w, r)
	} else if r.Method == "GET" {
		log.Println("HTTP GET", r.UserAgent())
		getHandler(w, r)
	} else {
		fail(w, "Unsupported method", http.StatusMethodNotAllowed)
	}
}
