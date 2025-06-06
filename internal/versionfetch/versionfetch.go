package versionfetch

import (
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"strings"

	"samsung-firmware-tool/internal/request" // For FetchResult.VersionFetchResult
	"samsung-firmware-tool/internal/util"
)

// GetLatestVersion gets the latest firmware version for a given model and region.
func GetLatestVersion(model, region string) *request.VersionFetchResult {
	url := fmt.Sprintf("https://fota-cloud-dn.ospserver.net:443/firmware/%s/%s/version.xml", region, model)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return &request.VersionFetchResult{
			Error: err,
		}
	}
	req.Header.Set("User-Agent", "Kies2.0_FUS")

	resp, err := util.GlobalHttpClient.Do(req)
	if err != nil {
		return &request.VersionFetchResult{
			Error: err,
		}
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return &request.VersionFetchResult{
			Error: err,
		}
	}
	body := string(bodyBytes)

	var responseXML util.XMLNode
	err = xml.Unmarshal(bodyBytes, &responseXML)
	if err != nil {
		return &request.VersionFetchResult{
			Error:     fmt.Errorf("failed to parse XML response: %w", err),
			RawOutput: body,
		}
	}

	if strings.EqualFold(responseXML.XMLName.Local, "Error") {
		codeNode := util.FirstElementByTagName(&responseXML, "Code")
		messageNode := util.FirstElementByTagName(&responseXML, "Message")

		code := ""
		if codeNode != nil {
			code = codeNode.Text()
		}
		message := ""
		if messageNode != nil {
			message = messageNode.Text()
		}

		return &request.VersionFetchResult{
			Error:     fmt.Errorf("code: %s, message: %s", code, message),
			RawOutput: body,
		}
	}

	firmwareNode := util.FirstElementByTagName(&responseXML, "firmware")
	if firmwareNode == nil {
		return &request.VersionFetchResult{
			Error:     fmt.Errorf("firmware tag not found in response"),
			RawOutput: body,
		}
	}

	versionNode := util.FirstElementByTagName(firmwareNode, "version")
	if versionNode == nil {
		return &request.VersionFetchResult{
			Error:     fmt.Errorf("version tag not found in firmware"),
			RawOutput: body,
		}
	}

	latestNode := util.FirstElementByTagName(versionNode, "latest")
	if latestNode == nil {
		return &request.VersionFetchResult{
			Error:     fmt.Errorf("latest tag not found in version"),
			RawOutput: body,
		}
	}

	vc := strings.Split(latestNode.Text(), "/")
	if len(vc) == 3 {
		vc = append(vc, vc[0]) // Add pda to the end if missing
	}
	if len(vc) > 2 && vc[2] == "" {
		vc[2] = vc[0] // If phone is empty, use pda
	}

	androidVersion := ""
	for _, attr := range latestNode.Attr { // Access Attr directly from latestNode
		if attr.Name.Local == "o" {
			androidVersion = attr.Value
			break
		}
	}

	return &request.VersionFetchResult{
		VersionCode:    strings.Join(vc, "/"),
		AndroidVersion: androidVersion,
		RawOutput:      body,
	}
}
