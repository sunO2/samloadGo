package fusclient

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"os" // Used for splitting cookies
	"sync"

	"samsung-firmware-tool/internal/cryptutils"
	"samsung-firmware-tool/internal/util"
)

// RequestType defines the type of request to make to Samsung's server.
type RequestType string

const (
	GenerateNonce RequestType = "NF_DownloadGenerateNonce.do"
	BinaryInform  RequestType = "NF_DownloadBinaryInform.do"
	BinaryInit    RequestType = "NF_DownloadBinaryInitForMass.do"
)

// FusClient manages communications with Samsung's server.
type FusClient struct {
	encNonce  string
	nonce     string
	auth      string
	sessionID string
	mu        sync.Mutex // Mutex to protect client state during concurrent access
}

// NewFusClient creates and returns a new FusClient instance.
func NewFusClient() *FusClient {
	return &FusClient{}
}

// GetNonce retrieves the current nonce, generating it if necessary.
func (f *FusClient) GetNonce() (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.nonce == "" {
		err := f.generateNonce()
		if err != nil {
			return "", err
		}
	}
	return f.nonce, nil
}

// generateNonce generates a new nonce by making a request to the server.
func (f *FusClient) generateNonce() error {
	fmt.Println("Generating nonce.")
	_, err := f.MakeReq(GenerateNonce, "", true)
	if err != nil {
		return err
	}
	fmt.Printf("Nonce: %s\n", f.nonce)
	fmt.Printf("Auth: %s\n", f.auth)
	return nil
}

// getAuthV constructs the Authorization header value.
func (f *FusClient) getAuthV(includeNonce bool) string {
	nonceVal := ""
	if includeNonce {
		nonceVal = f.encNonce
	}
	return fmt.Sprintf("FUS nonce=\"%s\", signature=\"%s\", nc=\"\", type=\"\", realm=\"\", newauth=\"1\"", nonceVal, f.auth)
}

// getDownloadUrl constructs the download URL for a given file path.
func (f *FusClient) getDownloadUrl(path string) string {
	return fmt.Sprintf("http://cloud-neofussvr.samsungmobile.com/NF_DownloadBinaryForMass.do?file=%s", path)
}

// makeReq makes a request to Samsung, automatically inserting authorization data.
func (f *FusClient) MakeReq(requestType RequestType, data string, includeNonce bool) (string, error) {
	if f.nonce == "" && requestType != GenerateNonce {
		err := f.generateNonce()
		if err != nil {
			return "", err
		}
	}
	authV := f.getAuthV(includeNonce)

	req, err := http.NewRequest("POST", fmt.Sprintf("https://neofussvr.sslcs.cdngc.net/%s", requestType), bytes.NewBufferString(data))
	if err != nil {
		return "", err
	}

	req.Header.Set("Authorization", authV)
	req.Header.Set("User-Agent", "Kiss2.0_FUS")
	req.Header.Set("Cookie", "JSESSIONID="+f.sessionID)
	req.Header.Set("Set-Cookie", "JSESSIONID="+f.sessionID)
	req.Header.Set("Content-Length", fmt.Sprintf("%d", len(data)))

	resp, err := util.GlobalHttpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	body := string(bodyBytes)

	if requestType != GenerateNonce && f.is401(resp, body) {
		err = f.generateNonce()
		if err != nil {
			return "", err
		}
		return f.MakeReq(requestType, data, includeNonce) // Retry with new nonce
	}

	// Update nonce and session ID from response headers
	if nonceHeader := resp.Header.Get("NONCE"); nonceHeader != "" {
		f.encNonce = nonceHeader
		decryptedNonce, err := cryptutils.DecryptNonce(f.encNonce)
		if err != nil {
			return "", fmt.Errorf("failed to decrypt nonce: %w", err)
		}
		f.nonce = decryptedNonce
		auth, err := cryptutils.GetAuth(f.nonce)
		if err != nil {
			return "", fmt.Errorf("failed to get auth: %w", err)
		}
		f.auth = auth
	} else if nonceHeader := resp.Header.Get("nonce"); nonceHeader != "" {
		f.encNonce = nonceHeader
		decryptedNonce, err := cryptutils.DecryptNonce(f.encNonce)
		if err != nil {
			return "", fmt.Errorf("failed to decrypt nonce: %w", err)
		}
		f.nonce = decryptedNonce
		auth, err := cryptutils.GetAuth(f.nonce)
		if err != nil {
			return "", fmt.Errorf("failed to get auth: %w", err)
		}
		f.auth = auth
	}

	for _, cookie := range resp.Cookies() {
		if cookie.Name == "JSESSIONID" {
			// The Kotlin code removes everything after the first semicolon.
			// Go's cookie.Value already gives the clean value.
			// However, the Kotlin code also has `.replace(Regex(";.*$"), "")`
			// which implies the value itself might contain semicolons.
			// Let's try to replicate that behavior if needed.
			// For now, assume cookie.Value is clean.
			f.sessionID = cookie.Value
			break
		}
	}

	return body, nil
}

// DownloadFile downloads a file from Samsung's server.
func (f *FusClient) DownloadFile(
	fileName string,
	start int64,
	size int64,
	output *os.File,
	outputSize int64,
	progressCallback func(current, max, bps int64),
) (string, error) {
	authV := f.getAuthV(true)
	url := f.getDownloadUrl(fileName)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}

	req.Header.Set("Authorization", authV)
	req.Header.Set("User-Agent", "Kies2.0_FUS")
	if start > 0 {
		req.Header.Set("Range", fmt.Sprintf("bytes=%d-", start))
	}

	resp, err := util.GlobalHttpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	md5 := resp.Header.Get("Content-MD5")

	err = util.TrackOperationProgress(
		size,
		progressCallback,
		func() (int64, error) {
			n, err := io.CopyN(output, resp.Body, util.DEFAULT_CHUNK_SIZE)
			return n, err
		},
		outputSize,
		func() bool {
			return true // Continue until io.CopyN returns EOF or error
		},
		false,
	)

	if err != nil && err != io.EOF {
		return "", err
	}

	return md5, nil
}

// is401 checks if the response indicates a 401 Unauthorized status.
func (f *FusClient) is401(resp *http.Response, body string) bool {
	if resp.StatusCode == 401 {
		return true
	}

	var fusMsg util.XMLNode
	err := xml.Unmarshal([]byte(body), &fusMsg)
	if err != nil {
		return false // Not XML or parsing error, assume not 401 from body
	}

	fusBody := util.FirstElementByTagName(&fusMsg, "FUSBody")
	if fusBody == nil {
		return false
	}
	results := util.FirstElementByTagName(fusBody, "Results")
	if results == nil {
		return false
	}
	status := util.FirstElementByTagName(results, "Status")
	if status == nil {
		return false
	}

	return status.Text() == "401"
}
