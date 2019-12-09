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
	"time"
)

func getAddress(r *http.Request) (AppWorkspaceID, error) {
	parts := strings.Split(r.URL.Path[1:], "/")
	if len(parts) != 3 {
		return nil, fmt.Errorf("Unexpected path: %s", r.URL.Path[1:])
	}
	return &dbAppWorkspace{
		appID:       parts[1],
		workspaceID: parts[2],
	}, nil
}

func readUserID(reader *multipart.Reader) (UserID, string, int) {
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

	return &dbUserID{userID}, "", http.StatusOK
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

	//qkeep := newJSONQuotaKeeper("quotas.json", address)
	qkeep := newTCPQuotaKeeper()
	settings := &dbKeeperSettings{}

	r.Body = http.MaxBytesReader(w, r.Body, settings.MaxUploadSize(address))
	reader, err := r.MultipartReader()

	userID, errstr, status := readUserID(reader)

	if len(errstr) > 0 {
		fail(w, errstr, status)
		return
	}

	if !isAuthorized(userID) {
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
		t1 := time.Now()
		buf := bufio.NewReader(p)
		sniff, _ := buf.Peek(chunkSize)
		contentType := http.DetectContentType(sniff)
		log.Println(fmt.Sprintf("File: %s; Content type: %s", p.FileName(), contentType))
		if !isContentTypeAllowed(contentType) {
			fail(w, "file type not allowed: "+contentType, http.StatusBadRequest)
			return
		}

		outputWriter, guid, err := getBinaryWriter(address, p.FileName())
		if err != nil {
			fail(w, err.Error(), http.StatusInternalServerError)
			return
		}
		provider := createQuotaManager(qkeep, settings)

		wr := QuotaCounterWriter(outputWriter, provider, userID, address, false)
		defer wr.Close()

		written, err := io.Copy(wr, buf)

		if err != nil && err == errOutOfQuota {
			fail(w, "Out of quota", http.StatusInsufficientStorage)
			return
		}
		if err != nil && err != io.EOF {
			fail(w, err.Error(), http.StatusInternalServerError)
			return
		}

		millis := time.Since(t1).Nanoseconds() / 1000000
		sec := float64(millis) / 1000.0
		mbs := float64(written) / 1000000
		rate := mbs / sec
		log.Println(fmt.Sprintf("Written: %.2f Mb (%d bytes) in %.2f seconds (%d ms). Speed: %.2f Mb/s", mbs, written, sec, millis, rate))

		resp.Result = append(resp.Result, PostResponseItem{
			FileName: p.FileName(),
			GUID:     guid,
		})
		resp.DiskQuota, err = qkeep.getUserQuota(userID, address)
		if err != nil {
			fail(w, "Internal server error: "+err.Error(), http.StatusInternalServerError)
			return
		}

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
	addr := dbAppWorkspace{
		appID:       parts[1],
		workspaceID: parts[2],
	}

	bytes, err := ReadBin(&addr, parts[3])
	if err != nil {
		fail(w, err.Error(), http.StatusInternalServerError)
	}
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
