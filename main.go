package main

import (
	"flag"
	"fmt"
	"os"

	"samsung-firmware-tool/internal/cryptutils"
	"samsung-firmware-tool/internal/fusclient"
	"samsung-firmware-tool/internal/request"
	"samsung-firmware-tool/internal/util"
	"samsung-firmware-tool/internal/versionfetch"
)

func main() {
	// Define command-line flags
	model := flag.String("model", "", "Device model (e.g., SM-G998U)")
	region := flag.String("region", "", "Device region (e.g., XAA)")
	imeiSerial := flag.String("imei", "", "Device IMEI or Serial number (can be multiple separated by newline or semicolon)")
	fwVersion := flag.String("fw", "", "Firmware version (e.g., G998USQU4AUF5/G998UOYN4AUF5/G998USQU4AUF5/G998USQU4AUF5)")
	outputFile := flag.String("output", "", "Output file path for download or decryption")
	inputFile := flag.String("input", "", "Input file path for decryption")

	checkVersion := flag.Bool("check-version", false, "Check for the latest firmware version")
	downloadFirmware := flag.Bool("download", false, "Download firmware")
	decryptFirmware := flag.Bool("decrypt", false, "Decrypt firmware")

	flag.Parse()

	// Check for required parameters in command-line mode, and prompt if missing
	// These values will also be used as defaults for interactive mode
	if *model == "" {
		fmt.Print("请输入设备型号 (例如: SM-G998U): ")
		fmt.Scanln(model)
		if *model == "" {
			fmt.Println("错误: 设备型号是必需的。")
			os.Exit(1)
		}
	}
	if *region == "" {
		fmt.Print("请输入设备地区 (例如: XAA): ")
		fmt.Scanln(region)
		if *region == "" {
			fmt.Println("错误: 设备地区是必需的。")
			os.Exit(1)
		}
	}

	if *imeiSerial == "" {
		fmt.Print("请输设备imei: ")
		fmt.Scanln(imeiSerial)
		if *imeiSerial == "" {
			fmt.Println("错误: 设备imei是必需的。")
			os.Exit(1)
		}
	}

	// imeiSerial is only required for download and decrypt operations
	if (*downloadFirmware || *decryptFirmware) && *imeiSerial == "" {
		fmt.Print("请输入设备IMEI或序列号 (多个请用分号或换行符分隔): ")
		fmt.Scanln(imeiSerial)
		if *imeiSerial == "" {
			fmt.Println("错误: 设备IMEI或序列号是必需的。")
			os.Exit(1)
		}
	}

	// Initialize interactive variables with command-line flag values if provided
	interactiveModel := *model
	interactiveRegion := *region
	interactiveImeiSerial := *imeiSerial

	// Check if any action flags were provided. If not, enter interactive mode.
	if !*checkVersion && !*downloadFirmware && !*decryptFirmware {
		for {
			var choice string
			fmt.Println("\n请输入下面代码执行操作:")
			fmt.Println("A - 查询最新固件版本")
			fmt.Println("B - 下载固件")
			fmt.Println("C - 解码固件")
			fmt.Println("!q - 退出")
			fmt.Print("请输入您的选择 (A/B/C/!q): ")
			fmt.Scanln(&choice)

			switch choice {
			case "A", "a":
				// Model and Region are already prompted and validated above
				checkLatestVersion(interactiveModel, interactiveRegion)
			case "B", "b":
				var interactiveFwVersion, interactiveOutputFile string
				// Model, Region, IMEI/Serial are already prompted and validated above
				fmt.Print("请输入固件版本 (例如: G998USQU4AUF5/G998UOYN4AUF5/G998USQU4AUF5/G998USQU4AUF5): ")
				fmt.Scanln(&interactiveFwVersion)
				fmt.Print("请输入输出文件路径 (例如: firmware.zip): ")
				fmt.Scanln(&interactiveOutputFile)
				if interactiveModel == "" || interactiveRegion == "" || interactiveFwVersion == "" || interactiveImeiSerial == "" || interactiveOutputFile == "" {
					fmt.Println("错误: 型号、地区、固件版本、IMEI/序列号和输出文件是下载固件所必需的。")
					continue
				}
				download(interactiveModel, interactiveRegion, interactiveFwVersion, interactiveImeiSerial, interactiveOutputFile)
			case "C", "c":
				var interactiveInputFile, interactiveOutputFile, interactiveFwVersion string
				fmt.Print("请输入输入文件路径 (例如: firmware.zip.enc4): ")
				fmt.Scanln(&interactiveInputFile)
				fmt.Print("请输入输出文件路径 (例如: firmware.zip): ")
				fmt.Scanln(&interactiveOutputFile)
				fmt.Print("请输入固件版本 (例如: G998USQU4AUF5/G998UOYN4AUF5/G998USQU4AUF5/G998USQU4AUF5): ")
				fmt.Scanln(&interactiveFwVersion)
				// Model, Region, IMEI/Serial are already prompted and validated above
				if interactiveInputFile == "" || interactiveOutputFile == "" || interactiveFwVersion == "" || interactiveModel == "" || interactiveRegion == "" || interactiveImeiSerial == "" {
					fmt.Println("错误: 输入文件、输出文件、固件版本、型号、地区和IMEI/序列号是解码固件所必需的。")
					continue
				}
				decrypt(interactiveInputFile, interactiveOutputFile, interactiveFwVersion, interactiveModel, interactiveRegion, interactiveImeiSerial)
			case "!q":
				fmt.Println("退出程序。")
				return
			default:
				fmt.Println("无效的选择。请选择 A、B、C 或 !q。")
			}
		}
	} else { // Command-line flags were provided
		// The model, region, and imeiSerial are already prompted and validated above if they were missing.
		// Now, proceed with the original command-line flag logic.
		if *checkVersion {
			if *model == "" || *region == "" { // This check is redundant now but kept for clarity
				fmt.Println("Error: --model and --region are required for checking version.")
				flag.Usage()
				os.Exit(1)
			}
			checkLatestVersion(*model, *region)
		} else if *downloadFirmware {
			if *model == "" || *region == "" || *fwVersion == "" || *imeiSerial == "" || *outputFile == "" { // This check is partially redundant now
				fmt.Println("Error: --model, --region, --fw, --imei, and --output are required for downloading firmware.")
				flag.Usage()
				os.Exit(1)
			}
			download(*model, *region, *fwVersion, *imeiSerial, *outputFile)
		} else if *decryptFirmware {
			if *inputFile == "" || *outputFile == "" || *fwVersion == "" || *model == "" || *region == "" || *imeiSerial == "" { // This check is partially redundant now
				fmt.Println("Error: --input, --output, --fw, --model, --region, and --imei are required for decrypting firmware.")
				flag.Usage()
				os.Exit(1)
			}
			decrypt(*inputFile, *outputFile, *fwVersion, *model, *region, *imeiSerial)
		} else {
			fmt.Println("Please specify an action: --check-version, --download, or --decrypt.")
			flag.Usage()
			os.Exit(1)
		}
	}
}

func checkLatestVersion(model, region string) {
	fmt.Printf("Checking latest version for Model: %s, Region: %s\n", model, region)
	result := versionfetch.GetLatestVersion(model, region)

	if result.Error != nil {
		fmt.Printf("Error checking version: %v\n", result.Error)
		fmt.Printf("Raw output: %s\n", result.RawOutput)
		return
	}

	fmt.Printf("Latest Firmware Version: %s\n", result.VersionCode)
	fmt.Printf("Android Version: %s\n", result.AndroidVersion)
}

func performDownloadGo(binaryInfo *request.BinaryFileInfo, client *fusclient.FusClient, outputPath string) {
	outputFile, err := os.Create(outputPath + "/" + binaryInfo.FileName)
	if err != nil {
		fmt.Printf("Error creating output file: %v\n", err)
		return
	}
	defer outputFile.Close()

	// Perform BINARY_INIT request as in Kotlin version
	nonce, err := client.GetNonce()
	if err != nil {
		fmt.Printf("Error getting nonce for BINARY_INIT: %v\n", err)
		return
	}
	binaryInitRequest := request.CreateBinaryInit(binaryInfo.FileName, nonce)
	_, err = client.MakeReq(fusclient.BinaryInit, binaryInitRequest, true)
	if err != nil {
		fmt.Printf("Error performing BINARY_INIT request: %v\n", err)
		return
	}

	progressCallback := func(current, max, bps int64) {
		fmt.Printf("\rDownloading: %d/%d bytes (%.2f%%) @ %d B/s", current, max, float64(current)/float64(max)*100, bps)
	}

	md5Sum, err := client.DownloadFile(binaryInfo.Path+binaryInfo.FileName, 0, binaryInfo.Size, outputFile, 0, progressCallback)
	if err != nil {
		fmt.Printf("\nError downloading file: %v\n", err)
		return
	}
	fmt.Printf("\nDownload complete. MD5: %s\n", md5Sum)

	// Optional: Verify MD5
	// if md5Sum != "" {
	// 	file, err := os.Open(outputPath)
	// 	if err != nil {
	// 		fmt.Printf("Error opening downloaded file for MD5 check: %v\n", err)
	// 		return
	// 	}
	// 	defer file.Close()
	// 	match, err := cryptutils.CheckMD5(md5Sum, file)
	// 	if err != nil {
	// 		fmt.Printf("Error checking MD5: %v\n", err)
	// 	} else if match {
	// 		fmt.Println("MD5 check successful.")
	// 	} else {
	// 		fmt.Println("MD5 check failed.")
	// 	}
	// }
}

func download(model, region, fwVersion, imeiSerial, outputPath string) {
	fmt.Printf("Downloading firmware %s for Model: %s, Region: %s to %s\n", fwVersion, model, region, outputPath)

	client := fusclient.NewFusClient()

	onFinish := func(msg string) {
		fmt.Println(msg)
	}
	onVersionException := func(err error, info *request.BinaryFileInfo) {
		fmt.Printf("Version exception: %v\n", err)
		if info != nil {
			fmt.Println("Attempting to proceed with download despite version exception...")
			performDownloadGo(info, client, outputPath)
		}
	}
	shouldReportError := func(err error) bool {
		return true // For now, always report
	}

	binaryInfo := request.RetrieveBinaryFileInfo(fwVersion, model, region, imeiSerial, client, onFinish, onVersionException, shouldReportError)
	if binaryInfo == nil {
		fmt.Println("Failed to retrieve binary file information.")
		return
	}

	performDownloadGo(binaryInfo, client, outputPath)
}

func decrypt(inputPath, outputPath, fwVersion, model, region, imeiSerial string) {
	fmt.Printf("Decrypting %s to %s\n", inputPath, outputPath)

	client := fusclient.NewFusClient()

	onFinish := func(msg string) {
		fmt.Println(msg)
	}
	onVersionException := func(err error, info *request.BinaryFileInfo) {
		fmt.Printf("Version exception: %v\n", err)
		if info != nil {
			fmt.Printf("Binary File Info: %+v\n", *info)
		}
	}
	shouldReportError := func(err error) bool {
		return true // For now, always report
	}

	binaryInfo := request.RetrieveBinaryFileInfo(fwVersion, model, region, imeiSerial, client, onFinish, onVersionException, shouldReportError)
	if binaryInfo == nil {
		fmt.Println("Failed to retrieve binary file information for decryption key.")
		return
	}

	var decryptionKey []byte
	var decryptionKeyStr string

	// Determine decryption key based on file extension or other info
	// Kotlin code uses .enc4 and .enc2. We need to infer this.
	// For simplicity, let's assume if V4Key is present, use it, otherwise use V2Key.
	if binaryInfo.V4Key != nil {
		decryptionKey = binaryInfo.V4Key
		decryptionKeyStr = binaryInfo.V4KeyStr
		fmt.Println("Using V4 decryption key.")
	} else {
		decryptionKey, decryptionKeyStr = cryptutils.GetV2Key(fwVersion, model, region)
		fmt.Println("Using V2 decryption key.")
	}

	fmt.Printf("Decryption Key (MD5): %x\n", decryptionKey)
	fmt.Printf("Decryption Key (String): %s\n", decryptionKeyStr)

	inputFile, err := os.Open(inputPath)
	if err != nil {
		fmt.Printf("Error opening input file: %v\n", err)
		return
	}
	defer inputFile.Close()

	outputFile, err := os.Create(outputPath)
	if err != nil {
		fmt.Printf("Error creating output file: %v\n", err)
		return
	}
	defer outputFile.Close()

	inputStat, err := inputFile.Stat()
	if err != nil {
		fmt.Printf("Error getting input file info: %v\n", err)
		return
	}
	fileSize := inputStat.Size()

	progressCallback := func(current, max, bps int64) {
		fmt.Printf("\rDecrypting: %d/%d bytes (%.2f%%) @ %d B/s", current, max, float64(current)/float64(max)*100, bps)
	}

	err = cryptutils.DecryptProgress(inputFile, outputFile, decryptionKey, fileSize, util.DEFAULT_CHUNK_SIZE, progressCallback)
	if err != nil {
		fmt.Printf("\nError decrypting file: %v\n", err)
		return
	}
	fmt.Println("\nDecryption complete.")

	// Optional: Verify CRC32
	// if binaryInfo.CRC32 != 0 {
	// 	fmt.Println("Checking CRC32...")
	// 	// Reopen decrypted file for CRC32 check
	// 	decryptedFile, err := os.Open(outputPath)
	// 	if err != nil {
	// 		fmt.Printf("Error opening decrypted file for CRC32 check: %v\n", err)
	// 		return
	// 	}
	// 	defer decryptedFile.Close()
	//
	// 	decryptedStat, err := decryptedFile.Stat()
	// 	if err != nil {
	// 		fmt.Printf("Error getting decrypted file info for CRC32 check: %v\n", err)
	// 		return
	// 	}
	// 	decryptedFileSize := decryptedStat.Size()
	//
	// 	crcProgressCallback := func(current, max, bps int64) {
	// 		fmt.Printf("\rCRC32 Check: %d/%d bytes (%.2f%%)", current, max, float64(current)/float64(max)*100)
	// 	}
	//
	// 	match, err := cryptutils.CheckCrc32(decryptedFile, decryptedFileSize, binaryInfo.CRC32, crcProgressCallback)
	// 	if err != nil {
	// 		fmt.Printf("\nError checking CRC32: %v\n", err)
	// 	} else if match {
	// 		fmt.Println("\nCRC32 check successful.")
	// 	} else {
	// 		fmt.Println("\nCRC32 check failed.")
	// 	}
	// }
}
