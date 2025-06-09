package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

var (
	model       string
	region      string
	imeiSerial  string
	fwVersion   string
	outputFile  string
	inputFile   string
	currentLang string
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "samsung-firmware-tool",
	Short: "", // Will be set in init()
	Long:  "", // Will be set in init()
}

var translations = map[string]map[string]string{
	"en": {
		"root_short": "A tool to check, download, and decrypt Samsung firmware.",
		"root_long": `samsung-firmware-tool is a command-line application that allows you to:

- Check for the latest firmware version for a given device model and region.
- Download firmware.
- Decrypt downloaded firmware files.`,
		"model_desc":                          "Device model (e.g., SM-G998U)",
		"region_desc":                         "Device region (e.g., XAA)",
		"imei_desc":                           "Device IMEI or Serial number (can be multiple separated by newline or semicolon)",
		"fw_desc":                             "Firmware version (e.g., G998USQU4AUF5/G998UOYN4AUF5/G998USQU4AUF5/G998USQU4AUF5)",
		"output_desc":                         "Output file path for download or decryption",
		"input_desc":                          "Input file path for decryption",
		"check_short":                         "Check for the latest firmware version",
		"check_long":                          "This command checks for the latest firmware version for a given device model and region.",
		"download_short":                      "Download firmware",
		"download_long":                       "This command downloads firmware for a given device model, region, firmware version, and IMEI/Serial number.",
		"decrypt_short":                       "Decrypt firmware",
		"decrypt_long":                        "This command decrypts a firmware file using the provided firmware version, model, region, and IMEI/Serial number.",
		"err_model_region_required":           "Error: --model and --region are required.",
		"err_download_required":               "Error: --model, --region, --fw, --imei, and --output are required for downloading firmware.",
		"err_decrypt_required":                "Error: --input, --output, --fw, --model, --region, and --imei are required for decrypting firmware.",
		"checking_version":                    "Checking latest version for Model: %s, Region: %s\n",
		"error_checking_version":              "Error checking version: %v\n",
		"raw_output":                          "Raw output: %s\n",
		"latest_fw_version":                   "Latest Firmware Version: %s\n",
		"android_version":                     "Android Version: %s\n",
		"downloading_fw":                      "Downloading firmware %s for Model: %s, Region: %s to %s\n",
		"error_creating_output_file":          "Error creating output file: %v\n",
		"error_getting_nonce":                 "Error getting nonce for BINARY_INIT: %v\n",
		"error_binary_init_request":           "Error performing BINARY_INIT request: %v\n",
		"downloading_progress":                "\rDownloading: %d/%d bytes (%.2f%%) @ %d B/s",
		"error_downloading_file":              "\nError downloading file: %v\n",
		"download_complete_md5":               "\nDownload complete. MD5: %s\n",
		"failed_retrieve_binary_info":         "Failed to retrieve binary file information.",
		"version_exception":                   "Version exception: %v\n",
		"attempting_proceed_download":         "Attempting to proceed with download despite version exception...",
		"decrypting_file":                     "Decrypting %s to %s\n",
		"failed_retrieve_binary_info_decrypt": "Failed to retrieve binary file information for decryption key.",
		"using_v4_key":                        "Using V4 decryption key.",
		"using_v2_key":                        "Using V2 decryption key.",
		"decryption_key_md5":                  "Decryption Key (MD5): %x\n",
		"decryption_key_string":               "Decryption Key (String): %s\n",
		"error_opening_input_file":            "Error opening input file: %v\n",
		"error_creating_output_file_decrypt":  "Error creating output file: %v\n",
		"error_getting_input_file_info":       "Error getting input file info: %v\n",
		"decrypting_progress":                 "\rDecrypting: %d/%d bytes (%.2f%%) @ %d B/s",
		"error_decrypting_file":               "\nError decrypting file: %v\n",
		"decryption_complete":                 "\nDecryption complete.",
	},
	"zh": {
		"root_short": "一个用于检查、下载和解密三星固件的工具。",
		"root_long": `samsung-firmware-tool 是一个命令行应用程序，允许您：

- 检查给定设备型号和地区的最新固件版本。
- 下载固件。
- 解密已下载的固件文件。`,
		"model_desc":                          "设备型号 (例如: SM-G998U)",
		"region_desc":                         "设备地区 (例如: XAA)",
		"imei_desc":                           "设备IMEI或序列号 (多个请用分号或换行符分隔)",
		"fw_desc":                             "固件版本 (例如: G998USQU4AUF5/G998UOYN4AUF5/G998USQU4AUF5/G998USQU4AUF5)",
		"output_desc":                         "下载或解密的输出文件路径",
		"input_desc":                          "解密的输入文件路径",
		"check_short":                         "查询最新固件版本",
		"check_long":                          "此命令用于检查给定设备型号和地区的最新固件版本。",
		"download_short":                      "下载固件",
		"download_long":                       "此命令用于下载给定设备型号、地区、固件版本和IMEI/序列号的固件。",
		"decrypt_short":                       "解密固件",
		"decrypt_long":                        "此命令使用提供的固件版本、型号、地区和IMEI/序列号解密固件文件。",
		"err_model_region_required":           "错误: --model 和 --region 是必需的。",
		"err_download_required":               "错误: --model, --region, --fw, --imei, 和 --output 是下载固件所必需的。",
		"err_decrypt_required":                "错误: --input, --output, --fw, --model, --region, 和 --imei 是解码固件所必需的。",
		"checking_version":                    "正在检查型号: %s, 地区: %s 的最新版本\n",
		"error_checking_version":              "检查版本时出错: %v\n",
		"raw_output":                          "原始输出: %s\n",
		"latest_fw_version":                   "最新固件版本: %s\n",
		"android_version":                     "安卓版本: %s\n",
		"downloading_fw":                      "正在下载型号: %s, 地区: %s 的固件 %s 到 %s\n",
		"error_creating_output_file":          "创建输出文件时出错: %v\n",
		"error_getting_nonce":                 "获取Nonce用于BINARY_INIT时出错: %v\n",
		"error_binary_init_request":           "执行BINARY_INIT请求时出错: %v\n",
		"downloading_progress":                "\r正在下载: %d/%d 字节 (%.2f%%) @ %d B/s",
		"error_downloading_file":              "\n下载文件时出错: %v\n",
		"download_complete_md5":               "\n下载完成。MD5: %s\n",
		"failed_retrieve_binary_info":         "未能检索二进制文件信息。",
		"version_exception":                   "版本异常: %v\n",
		"attempting_proceed_download":         "尝试继续下载，尽管存在版本异常...",
		"decrypting_file":                     "正在解密 %s 到 %s\n",
		"failed_retrieve_binary_info_decrypt": "未能检索用于解密密钥的二进制文件信息。",
		"using_v4_key":                        "正在使用 V4 解密密钥。",
		"using_v2_key":                        "正在使用 V2 解密密钥。",
		"decryption_key_md5":                  "解密密钥 (MD5): %x\n",
		"decryption_key_string":               "解密密钥 (字符串): %s\n",
		"error_opening_input_file":            "打开输入文件时出错: %v\n",
		"error_creating_output_file_decrypt":  "创建输出文件时出错: %v\n",
		"error_getting_input_file_info":       "获取输入文件信息时出错: %v\n",
		"decrypting_progress":                 "\r正在解密: %d/%d 字节 (%.2f%%) @ %d B/s",
		"error_decrypting_file":               "\n解密文件时出错: %v\n",
		"decryption_complete":                 "\n解密完成。",
	},
}

func T(key string) string {
	if _, ok := translations[currentLang]; !ok {
		currentLang = "en" // Default to English if language not found
	}
	if val, ok := translations[currentLang][key]; ok {
		return val
	}
	return key // Return key if translation not found
}

func init() {
	// Determine language from environment
	lang := os.Getenv("LANG")
	if lang == "" {
		lang = os.Getenv("LC_ALL")
	}
	if strings.HasPrefix(lang, "zh") {
		currentLang = "zh"
	} else {
		currentLang = "en" // Default to English
	}

	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be available to all subcommands in this application.
	rootCmd.Short = T("root_short")
	rootCmd.Long = T("root_long")

	rootCmd.PersistentFlags().StringVarP(&model, "model", "m", "", T("model_desc"))
	rootCmd.PersistentFlags().StringVarP(&region, "region", "r", "", T("region_desc"))
	rootCmd.PersistentFlags().StringVarP(&imeiSerial, "imei", "i", "", T("imei_desc"))
	rootCmd.PersistentFlags().StringVarP(&fwVersion, "fw", "f", "", T("fw_desc"))
	rootCmd.PersistentFlags().StringVarP(&outputFile, "output", "o", "", T("output_desc"))
	rootCmd.PersistentFlags().StringVarP(&inputFile, "input", "p", "", T("input_desc")) // Changed from -i to -p to avoid conflict with imei

	// Cobra also supports local flags, which will only run when this command
	// is called directly.
	// rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	// Always execute rootCmd to parse flags, but handle errors
	err := rootCmd.Execute()
	if err != nil {
		// If the error is due to a subcommand not found or similar,
		// we might want to proceed to interactive mode.
		// For now, if any error occurs, we exit.
		// A more robust solution would check the error type.
		if err.Error() != "unknown command" && !strings.Contains(err.Error(), "unknown flag") {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	}

	// If no subcommand was executed and no specific command was given, default to the 'run' command.
	if len(os.Args) == 1 {
		runCmd.Run(rootCmd, []string{}) // Pass an empty string slice instead of 'args'
	}
}
