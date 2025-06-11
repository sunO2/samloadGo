package cmd

import (
	"errors"
	"fmt"
	"os"

	"samsung-firmware-tool/internal/cryptutils"
	"samsung-firmware-tool/internal/fusclient"
	"samsung-firmware-tool/internal/request"
	"samsung-firmware-tool/internal/util"

	"github.com/spf13/cobra"
)

// DecryptCmd represents the decrypt command
var DecryptCmd = &cobra.Command{
	Use:   "decrypt",
	Short: "Decrypt firmware",
	Long:  `This command decrypts a firmware file using the provided firmware version, model, region, and IMEI/Serial number.`,
	Run: func(cmd *cobra.Command, args []string) {
		if inputFile == "" || outputFile == "" || fwVersion == "" || model == "" || region == "" || imeiSerial == "" {
			fmt.Println("错误: --input, --output, --fw, --model, --region, 和 --imei 是解码固件所必需的。")
			os.Exit(1)
		}
		progressCallback := func(current, max, bps int64) {
			fmt.Printf("\rDecrypting: %d/%d bytes (%.2f%%) @ %d B/s", current, max, float64(current)/float64(max)*100, bps)
		}
		DecryptFirmware(inputFile, outputFile, fwVersion, model, region, imeiSerial, progressCallback)
	},
}

func init() {
	rootCmd.AddCommand(DecryptCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// decryptCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// decryptCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

func DecryptFirmware(inputPath, outputPath, fwVersion, model, region, imeiSerial string, progressCallback ProgressCallback) error {
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
		return errors.New("failed to retrieve binary file information for decryption key")
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
		return fmt.Errorf("error opening input file: %v", err)
	}
	defer inputFile.Close()

	outputFile, err := os.Create(outputPath)
	if err != nil {
		fmt.Printf("Error creating output file: %v\n", err)
		return fmt.Errorf("error creating output file: %v", err)
	}
	defer outputFile.Close()

	inputStat, err := inputFile.Stat()
	if err != nil {
		fmt.Printf("Error getting input file info: %v\n", err)
		return fmt.Errorf("error getting input file info: %v", err)
	}
	fileSize := inputStat.Size()
	err = cryptutils.DecryptProgress(inputFile, outputFile, decryptionKey, fileSize, util.DEFAULT_CHUNK_SIZE, progressCallback)
	if err != nil {
		fmt.Printf("\nError decrypting file: %v\n", err)
		return fmt.Errorf("error decrypting file: %v", err)
	}
	fmt.Println("\nDecryption complete.")
	return nil
}
