package cmd

import (
	"fmt"
	"os"

	"samsung-firmware-tool/internal/fusclient"
	"samsung-firmware-tool/internal/request"

	"github.com/spf13/cobra"
)

// DownloadCmd represents the download command
var DownloadCmd = &cobra.Command{
	Use:   "download",
	Short: "Download firmware",
	Long:  `This command downloads firmware for a given device model, region, firmware version, and IMEI/Serial number.`,
	Run: func(cmd *cobra.Command, args []string) {
		if model == "" || region == "" || fwVersion == "" || imeiSerial == "" || outputFile == "" {
			fmt.Println("错误: --model, --region, --fw, --imei, 和 --output 是下载固件所必需的。")
			os.Exit(1)
		}
		downloadFirmware(model, region, fwVersion, imeiSerial, outputFile)
	},
}

func init() {
	rootCmd.AddCommand(DownloadCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// downloadCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// downloadCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
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
}

func downloadFirmware(model, region, fwVersion, imeiSerial, outputPath string) {
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
