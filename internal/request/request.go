package request

import (
	"encoding/xml"
	"fmt"
	"strings"
	"time"

	"samsung-firmware-tool/internal/cryptutils"
	"samsung-firmware-tool/internal/fusclient"
	"samsung-firmware-tool/internal/util"
)

// BinaryFileInfo represents information about a firmware binary file.
type BinaryFileInfo struct {
	Path     string
	FileName string
	Size     int64
	CRC32    uint32
	V4Key    []byte
	V4KeyStr string
}

// FetchResultGetBinaryFileResult represents the result of fetching binary file information.
type FetchResultGetBinaryFileResult struct {
	Info         *BinaryFileInfo
	Error        error
	RawOutput    string
	RequestBody  string
	ResponseCode string
}

// VersionFetchResult represents the result of fetching version information.
type VersionFetchResult struct {
	VersionCode    string
	AndroidVersion string
	Error          error
	RawOutput      string
}

// Custom error types
type VersionException struct {
	Message string
}

func (e *VersionException) Error() string {
	return e.Message
}

type VersionCheckException struct {
	Message string
}

func (e *VersionCheckException) Error() string {
	return e.Message
}

type VersionMismatchException struct {
	Message string
}

func (e *VersionMismatchException) Error() string {
	return e.Message
}

type NoBinaryFileError struct {
	Model  string
	Region string
}

func (e *NoBinaryFileError) Error() string {
	return fmt.Sprintf("No binary file found for model %s, region %s", e.Model, e.Region)
}

// GetLogicCheck generates a logic-check for a given input.
func GetLogicCheck(input, nonce string) string {
	if len(input) < 16 {
		return ""
	}

	var result strings.Builder
	for _, char := range nonce {
		idx := int(char) & 0xf // Equivalent to char.code % 16
		if idx >= len(input) {
			// This should ideally not happen if input is long enough and nonce chars are within expected range
			return "" // Or handle error appropriately
		}
		result.WriteByte(input[idx])
	}
	return result.String()
}

// performBinaryInformRetry performs a binary inform request with retries for multiple IMEI/Serial numbers.
func PerformBinaryInformRetry(
	fw, model, region, imeiSerial string,
	includeNonce bool,
	client *fusclient.FusClient,
) (string, *util.XMLNode, error) {
	splitImeiSerial := strings.FieldsFunc(imeiSerial, func(r rune) bool {
		return r == '\n' || r == ';'
	})

	var latestRequest string
	var latestResult *util.XMLNode
	var latestError error

	for i, imei := range splitImeiSerial {
		imei = strings.TrimSpace(imei)
		if imei == "" {
			continue
		}

		nonce, err := client.GetNonce()
		if err != nil {
			latestError = err
			continue
		}

		latestRequest = CreateBinaryInform(fw, model, region, nonce, imei)

		if i%10 == 0 {
			time.Sleep(1 * time.Second) // Delay as in Kotlin code
		}

		response, err := client.MakeReq(fusclient.BinaryInform, latestRequest, includeNonce)
		// fmt.Println(response)
		if err != nil {
			latestError = err
			fmt.Printf("Error making request for IMEI %s: %v\n", imei, err)
			continue
		}

		var fusMsg util.XMLNode
		err = xml.Unmarshal([]byte(response), &fusMsg)
		if err != nil {
			latestError = err
			fmt.Printf("Error unmarshalling XML for IMEI %s: %v\n", imei, err)
			continue
		}
		latestResult = &fusMsg

		statusNode := util.FirstElementByTagName(
			util.FirstElementByTagName(
				util.FirstElementByTagName(latestResult, "FUSBody"),
				"Results",
			),
			"Status",
		)
		status := ""
		if statusNode != nil {
			status = statusNode.Text()
		}

		if status != "408" {
			return latestRequest, latestResult, nil
		}
	}

	if latestError != nil {
		return latestRequest, latestResult, latestError
	}

	return latestRequest, latestResult, fmt.Errorf("all IMEI/Serial attempts failed with status 408")
}

var dataNode = func(node string, data string) string {
	return fmt.Sprintf("<%s><Data>%s</Data></%s>", node, strings.TrimSpace(data), node)
}

// CreateBinaryInform generates the XML needed to perform a binary inform.
func CreateBinaryInform(
	fw, model, region, nonce, imeiSerial string,
) string {
	split := strings.Split(fw, "/")
	pda, csc, phone, data := "", "", "", ""
	if len(split) > 0 {
		pda = split[0]
	}
	if len(split) > 1 {
		csc = split[1]
	}
	if len(split) > 2 {
		phone = split[2]
	}
	if len(split) > 3 {
		data = split[3]
	}

	logicCheck := GetLogicCheck(fw, nonce)

	// Manually construct XML string
	var xmlBuilder strings.Builder
	xmlBuilder.WriteString("<FUSMsg>")
	xmlBuilder.WriteString("<FUSHdr>")
	xmlBuilder.WriteString("<ProtoVer>1.0</ProtoVer>")
	xmlBuilder.WriteString("<SessionID>0</SessionID>")
	xmlBuilder.WriteString("<MsgID>1</MsgID>")
	xmlBuilder.WriteString("</FUSHdr>")
	xmlBuilder.WriteString("<FUSBody>")
	xmlBuilder.WriteString("<Put>")
	xmlBuilder.WriteString(dataNode("ACCESS_MODE", "2"))
	xmlBuilder.WriteString(dataNode("BINARY_NATURE", "1"))
	xmlBuilder.WriteString(dataNode("CLIENT_PRODUCT", "Smart Switch"))
	xmlBuilder.WriteString(dataNode("CLIENT_VERSION", "4.3.23123_1"))
	xmlBuilder.WriteString(dataNode("DEVICE_IMEI_PUSH", imeiSerial))
	xmlBuilder.WriteString(dataNode("DEVICE_FW_VERSION", fw))
	xmlBuilder.WriteString(dataNode("DEVICE_LOCAL_CODE", region))
	xmlBuilder.WriteString(dataNode("DEVICE_AID_CODE", region))
	xmlBuilder.WriteString(dataNode("DEVICE_MODEL_NAME", model))
	xmlBuilder.WriteString(dataNode("LOGIC_CHECK", logicCheck))
	xmlBuilder.WriteString(dataNode("DEVICE_CONTENTS_DATA_VERSION", data))
	xmlBuilder.WriteString(dataNode("DEVICE_CSC_CODE2_VERSION", csc))
	xmlBuilder.WriteString(dataNode("DEVICE_PDA_CODE1_VERSION", pda))
	xmlBuilder.WriteString(dataNode("DEVICE_PHONE_FONT_VERSION", phone))
	xmlBuilder.WriteString("<CLIENT_LANGUAGE>")
	xmlBuilder.WriteString("<Type>String</Type>")
	xmlBuilder.WriteString("<Type>ISO 3166-1-alpha-3</Type>")
	xmlBuilder.WriteString("<Data>1033</Data>")
	xmlBuilder.WriteString("</CLIENT_LANGUAGE>")

	// Some regions need extra properties specified.
	// TODO: Make these settable in the UI?
	var cc, mcc, mnc *string
	switch region {
	case "EUX":
		c := "DE"
		mccVal := "262"
		mncVal := "01"
		cc = &c
		mcc = &mccVal
		mnc = &mncVal
	case "EUY":
		c := "RS"
		mccVal := "220"
		mncVal := "01"
		cc = &c
		mcc = &mccVal
		mnc = &mncVal
	}

	if cc != nil {
		xmlBuilder.WriteString(dataNode("DEVICE_CC_CODE", *cc))
	}
	if mcc != nil {
		xmlBuilder.WriteString(dataNode("MCC_NUM", *mcc))
	}
	if mnc != nil {
		xmlBuilder.WriteString(dataNode("MNC_NUM", *mnc))
	}

	xmlBuilder.WriteString("</Put>")
	xmlBuilder.WriteString("<Get>")
	xmlBuilder.WriteString("<CmdID>2</CmdID>")
	xmlBuilder.WriteString("<LATEST_FW_VERSION/>")
	xmlBuilder.WriteString("</Get>")
	xmlBuilder.WriteString("</FUSBody>")
	xmlBuilder.WriteString("</FUSMsg>")

	return xmlBuilder.String()
}

// CreateBinaryInit generates the XML needed to perform a binary init.
func CreateBinaryInit(fileName, nonce string) string {
	special := fileName
	if len(fileName) > 0 {
		// Equivalent to slice(this.length - (16 % this.length)..this.lastIndex)
		// This logic seems a bit off in Kotlin, 16 % this.length would be 0 if length is a multiple of 16,
		// resulting in slice(length..lastIndex) which is empty.
		// Assuming it means the last 16 characters if length >= 16, otherwise the whole string.
		if len(fileName) >= 16 {
			split0 := strings.Split(fileName, ".")[0]
			special = split0[len(split0)-16:]
		}
	}

	logicCheck := GetLogicCheck(special, nonce)

	var xmlBuilder strings.Builder
	xmlBuilder.WriteString("<FUSMsg>")
	xmlBuilder.WriteString("<FUSHdr>")
	xmlBuilder.WriteString("<ProtoVer>1.0</ProtoVer>")
	xmlBuilder.WriteString("</FUSHdr>")
	xmlBuilder.WriteString("<FUSBody>")
	xmlBuilder.WriteString("<Put>")
	xmlBuilder.WriteString(dataNode("BINARY_FILE_NAME", fileName))
	xmlBuilder.WriteString(dataNode("LOGIC_CHECK", logicCheck))
	xmlBuilder.WriteString("</Put>")
	xmlBuilder.WriteString("</FUSBody>")
	xmlBuilder.WriteString("</FUSMsg>")

	return xmlBuilder.String()
}

// RetrieveBinaryFileInfo retrieves the file information for a given firmware.
func RetrieveBinaryFileInfo(
	fw, model, region, imeiSerial string,
	client *fusclient.FusClient,
	onFinish func(string),
	onVersionException func(error, *BinaryFileInfo),
	shouldReportError func(error) bool,
) *BinaryFileInfo {
	result := GetBinaryFile(fw, model, region, imeiSerial, client)

	info := result.Info
	err := result.Error
	output := result.RawOutput
	// requestBody := result.RequestBody // Commented out as it's unused without Bugsnag

	if err != nil {
		if _, ok := err.(*VersionException); ok && onVersionException != nil {
			onVersionException(err, info)
			return nil
		} else {
			onFinish(fmt.Sprintf("%s\n\n%s", err.Error(), output))
			// TODO: Implement isReportableCode and CrossPlatformBugsnag equivalent
			// if result.isReportableCode() && !strings.Contains(output, "Incapsula") &&
			// 	!model.isAccessoryModel && shouldReportError(err) {
			// 	CrossPlatformBugsnag.notify(DownloadError(requestBody, output, err))
			// }
		}
	}

	return info
}

// GetBinaryFile retrieves the file information for a given firmware.
func GetBinaryFile(
	fw, model, region, imeiSerial string,
	client *fusclient.FusClient,
) *FetchResultGetBinaryFileResult {
	requestBody, responseXMLNode, err := PerformBinaryInformRetry(fw, model, region, imeiSerial, false, client)
	if err != nil {
		return &FetchResultGetBinaryFileResult{
			Error:       err,
			RawOutput:   fmt.Sprintf("firmware: %s, model: %s, region: %s", fw, model, region),
			RequestBody: requestBody,
		}
	}

	statusNode := util.FirstElementByTagName(
		util.FirstElementByTagName(
			util.FirstElementByTagName(responseXMLNode, "FUSBody"),
			"Results",
		),
		"Status",
	)
	status := ""
	if statusNode != nil {
		status = statusNode.Text()
	}

	if status == "F01" {
		return &FetchResultGetBinaryFileResult{
			Error:        fmt.Errorf("invalid firmware error"), // TODO: Use proper error message from resources
			RawOutput:    responseXMLNode.Text(),
			RequestBody:  requestBody,
			ResponseCode: status,
		}
	}

	if status == "408" {
		return &FetchResultGetBinaryFileResult{
			Error:        fmt.Errorf("invalid IMEI or serial"), // TODO: Use proper error message from resources
			RawOutput:    responseXMLNode.Text(),
			RequestBody:  requestBody,
			ResponseCode: status,
		}
	}

	if status != "200" {
		return &FetchResultGetBinaryFileResult{
			Error:        fmt.Errorf("bad return status: %s", status), // TODO: Use proper error message from resources
			RawOutput:    responseXMLNode.Text(),
			RequestBody:  requestBody,
			ResponseCode: status,
		}
	}

	noBinaryError := func() *FetchResultGetBinaryFileResult {
		return &FetchResultGetBinaryFileResult{
			Error:        &NoBinaryFileError{Model: model, Region: region},
			RawOutput:    responseXMLNode.Text(),
			RequestBody:  requestBody,
			ResponseCode: status,
		}
	}

	sizeStr := util.FirstElementByTagName(util.FirstElementByTagName(
		util.FirstElementByTagName(
			util.FirstElementByTagName(responseXMLNode, "FUSBody"),
			"Put",
		),
		"BINARY_BYTE_SIZE",
	), "Data")
	size := int64(0)
	if sizeStr == nil || sizeStr.Text() == "" {
		return noBinaryError()
	}
	fmt.Sscanf(sizeStr.Text(), "%d", &size)

	fileNameNode := util.FirstElementByTagName(util.FirstElementByTagName(
		util.FirstElementByTagName(
			util.FirstElementByTagName(responseXMLNode, "FUSBody"),
			"Put",
		),
		"BINARY_NAME",
	), "Data")
	fileName := ""
	if fileNameNode == nil || fileNameNode.Text() == "" {
		return noBinaryError()
	}
	fileName = fileNameNode.Text()

	generateInfo := func() *BinaryFileInfo {
		pathNode := util.FirstElementByTagName(util.FirstElementByTagName(
			util.FirstElementByTagName(
				util.FirstElementByTagName(responseXMLNode, "FUSBody"),
				"Put",
			),
			"MODEL_PATH",
		), "Data")
		path := ""
		if pathNode != nil {
			path = pathNode.Text()
		}

		crc32Node := util.FirstElementByTagName(util.FirstElementByTagName(
			util.FirstElementByTagName(
				util.FirstElementByTagName(responseXMLNode, "FUSBody"),
				"Put",
			),
			"BINARY_CRC",
		), "Data")
		crc32Val := uint32(0)
		if crc32Node != nil && crc32Node.Text() != "" {
			fmt.Sscanf(crc32Node.Text(), "%d", &crc32Val)
		}

		// Kotlin code calls CryptUtils.getV4Key here if extractV4Key returns null.
		// This would create a circular dependency if CryptUtils also calls Request.
		// For now, we'll just use extractV4Key. If it's null, V4Key will be nil.
		v4Key, v4KeyStr := ExtractV4Key(responseXMLNode)
		if v4Key == nil {
			// TODO: Potentially call CryptUtils.GetV4Key here, but need to break circular dependency.
			// For now, leave it nil.
		}

		return &BinaryFileInfo{
			Path:     path,
			FileName: fileName,
			Size:     size,
			CRC32:    crc32Val,
			V4Key:    v4Key,
			V4KeyStr: v4KeyStr,
		}
	}

	dataKeys := []string{
		"DEVICE_USER_DATA_FILE",
		"DEVICE_BOOT_FILE",
		"DEVICE_PDA_CODE1_FILE",
	}

	var dataFile string
	for _, key := range dataKeys {
		node := util.FirstElementByTagName(
			util.FirstElementByTagName(
				util.FirstElementByTagName(
					util.FirstElementByTagName(responseXMLNode, "FUSBody"),
					"Put",
				),
				key,
			), "Data")

		if node != nil && node.Text() != "" {
			dataFile = node.Text()
			break
		}
	}

	if dataFile == "" {
		return &FetchResultGetBinaryFileResult{
			Info:         generateInfo(),
			Error:        &VersionCheckException{Message: "version check error"}, // TODO: Use proper error message
			RequestBody:  requestBody,
			ResponseCode: status,
		}
	}

	getIndex := func(file string) *int {
		if file == "" {
			return nil
		}
		fileSplit := strings.Split(file, "_")
		modelSuffix := model
		if strings.Contains(model, "-") {
			modelSuffix = strings.Split(model, "-")[1]
		}

		for i, part := range fileSplit {
			if strings.HasPrefix(part, modelSuffix) || strings.HasPrefix(part, strings.ReplaceAll(model, "-", "")) {
				return &i
			}
		}
		return nil
	}

	getSuffix := func(str string) string {
		split := strings.Split(str, "_")
		if len(split) > 1 {
			return split[1]
		}
		return ""
	}

	dataIndex := getIndex(dataFile)
	if dataIndex == nil {
		return &FetchResultGetBinaryFileResult{
			Info:         generateInfo(),
			Error:        &VersionCheckException{Message: "version check error: data file index not found"},
			RequestBody:  requestBody,
			ResponseCode: status,
		}
	}

	fwSplit := strings.Split(fw, "/")
	fwVersion, fwCsc, fwCp, fwPda := "", "", "", ""
	if len(fwSplit) > 0 {
		fwVersion = fwSplit[0]
	}
	if len(fwSplit) > 1 {
		fwCsc = fwSplit[1]
	}
	if len(fwSplit) > 2 {
		fwCp = fwSplit[2]
	}
	if len(fwSplit) > 3 {
		fwPda = fwSplit[3]
	}

	dataFileSplit := strings.Split(dataFile, "_")
	version := dataFileSplit[*dataIndex]
	versionSuffix := ""
	if *dataIndex+1 < len(dataFileSplit) {
		versionSuffix = dataFileSplit[*dataIndex+1]
	}

	cscFileNode := util.FirstElementByTagName(
		util.FirstElementByTagName(
			util.FirstElementByTagName(responseXMLNode, "FUSBody"),
			"Put",
		),
		"DEVICE_CSC_HOME_FILE",
	)
	cscFile := ""
	if cscFileNode != nil {
		cscFile = cscFileNode.Text()
	}
	if cscFile == "" {
		cscFileNode = util.FirstElementByTagName(
			util.FirstElementByTagName(
				util.FirstElementByTagName(responseXMLNode, "FUSBody"),
				"Put",
			),
			"DEVICE_CSC_FILE",
		)
		if cscFileNode != nil {
			cscFile = cscFileNode.Text()
		}
	}

	cscIndex := getIndex(cscFile)
	var servedCsc, cscSuffix string
	if cscIndex != nil {
		cscFileSplit := strings.Split(cscFile, "_")
		if *cscIndex < len(cscFileSplit) {
			servedCsc = cscFileSplit[*cscIndex]
		}
		if *cscIndex+1 < len(cscFileSplit) {
			cscSuffix = cscFileSplit[*cscIndex+1]
		}
	}

	cpFileNode := util.FirstElementByTagName(
		util.FirstElementByTagName(
			util.FirstElementByTagName(responseXMLNode, "FUSBody"),
			"Put",
		),
		"DEVICE_PHONE_FONT_FILE",
	)
	cpFile := ""
	if cpFileNode != nil {
		cpFile = cpFileNode.Text()
	}

	cpIndex := getIndex(cpFile)
	var servedCp, cpSuffix string
	if cpIndex != nil {
		cpFileSplit := strings.Split(cpFile, "_")
		if *cpIndex < len(cpFileSplit) {
			servedCp = cpFileSplit[*cpIndex]
		}
		if *cpIndex+1 < len(cpFileSplit) {
			cpSuffix = cpFileSplit[*cpIndex+1]
		}
	}

	pdaFileNode := util.FirstElementByTagName(
		util.FirstElementByTagName(
			util.FirstElementByTagName(responseXMLNode, "FUSBody"),
			"Put",
		),
		"DEVICE_PDA_CODE1_FILE",
	)
	pdaFile := ""
	if pdaFileNode != nil {
		pdaFile = pdaFileNode.Text()
	}

	pdaIndex := getIndex(pdaFile)
	var servedPda string
	if pdaIndex != nil {
		pdaFileSplit := strings.Split(pdaFile, "_")
		if *pdaIndex < len(pdaFileSplit) {
			servedPda = pdaFileSplit[*pdaIndex]
		}
	}

	finalServedCsc := servedCsc
	if finalServedCsc == "" {
		finalServedCsc = versionSuffix
	}
	finalServedCp := servedCp
	if finalServedCp == "" {
		finalServedCp = version
	}
	finalServedPda := servedPda
	if finalServedPda == "" {
		finalServedPda = version
	}

	served := fmt.Sprintf("%s/%s/%s/%s", version, finalServedCsc, finalServedCp, finalServedPda)

	cscMatch := fwCsc == finalServedCsc
	cpMatch := fwCp == finalServedCp
	fwVersionMatch := fwVersion == version
	fwPdaMatch := fwPda == finalServedPda

	cscSuffixMatch := true
	if fwCsc != "" { // Only check suffix if fwCsc is not empty
		fwCscSuffix := getSuffix(fwCsc)
		if fwCscSuffix != "" {
			cscSuffixMatch = fwCscSuffix == cscSuffix
		}
	}

	cpSuffixMatch := true
	if fwCp != "" { // Only check suffix if fwCp is not empty
		fwCpSuffix := getSuffix(fwCp)
		if fwCpSuffix != "" {
			cpSuffixMatch = fwCpSuffix == cpSuffix
		}
	}

	if served != fw || !cscMatch || !cpMatch || !fwVersionMatch ||
		!fwPdaMatch || !cscSuffixMatch || !cpSuffixMatch {
		return &FetchResultGetBinaryFileResult{
			Info:         generateInfo(),
			Error:        &VersionMismatchException{Message: fmt.Sprintf("version mismatch: expected %s, got %s", fw, served)}, // TODO: Use proper error message
			RequestBody:  requestBody,
			ResponseCode: status,
		}
	}

	return &FetchResultGetBinaryFileResult{
		Info:         generateInfo(),
		RequestBody:  requestBody,
		ResponseCode: status,
	}
}

// ExtractV4Key extracts the V4 decryption key from the XML response.
func ExtractV4Key(doc *util.XMLNode) ([]byte, string) {
	fwVerNode := util.FirstElementByTagName(util.FirstElementByTagName(
		util.FirstElementByTagName(
			util.FirstElementByTagName(doc, "FUSBody"),
			"Results",
		),
		"LATEST_FW_VERSION",
	), "Data")
	fwVer := ""
	if fwVerNode != nil {
		fwVer = fwVerNode.Text()
	}

	putNode := util.FirstElementByTagName(util.FirstElementByTagName(doc, "FUSBody"), "Put")
	if putNode == nil {
		return nil, ""
	}

	logicValNode := util.FirstElementByTagName(util.FirstElementByTagName(putNode, "LOGIC_VALUE_FACTORY"), "Data")
	if logicValNode == nil {
		logicValNode = util.FirstElementByTagName(util.FirstElementByTagName(putNode, "LOGIC_VALUE_HOME"), "Data")
	}
	logicVal := ""
	if logicValNode != nil {
		logicVal = logicValNode.Text()
	}

	if fwVer != "" && logicVal != "" {
		decKeyStr := GetLogicCheck(fwVer, logicVal)
		hasher := cryptutils.MD5Hasher() // Assuming MD5Hasher is a function in cryptutils that returns a new MD5 hash.
		hasher.Write([]byte(decKeyStr))
		return hasher.Sum(nil), decKeyStr
	}

	return nil, ""
}
