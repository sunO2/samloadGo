package main

/*
#cgo CFLAGS: -I.
#include <stdlib.h> // For C.free
#include <stdbool.h> // For bool

// Dart C API declarations (from dart_api.h)
// This is a simplified but more complete version for CGO.
typedef int64_t Dart_Port;

typedef enum {
  Dart_CObject_kNull = 0,
  Dart_CObject_kBool,
  Dart_CObject_kInt32,
  Dart_CObject_kInt64, // Value 3
  Dart_CObject_kDouble,
  Dart_CObject_kString,
  Dart_CObject_kArray, // Value 6
  Dart_CObject_kTypedData,
  Dart_CObject_kExternalTypedData,
  Dart_CObject_kSendPort,
  Dart_CObject_kCapability,
  Dart_CObject_kNativePointer,
  Dart_CObject_kUnsupported,
  Dart_CObject_kNumberOfTypes
} Dart_CObject_Type;

typedef struct _Dart_CObject {
  Dart_CObject_Type type;
  union {
    bool as_bool;
    int32_t as_int32;
    int64_t as_int64;
    double as_double;
    char* as_string;
    struct {
      intptr_t length;
      struct _Dart_CObject** values;
    } as_array;
    struct {
      Dart_Port id;
      Dart_Port origin_id;
    } as_send_port;
    void* as_native_pointer;
  } value;
} Dart_CObject;

typedef bool (*Dart_PostCObject_Type)(Dart_Port port, Dart_CObject* message);

// Global variable to store the Dart_PostCObject function pointer
static Dart_PostCObject_Type Dart_PostCObject_Fn = NULL;

// Global variable to store the Dart SendPort ID
static Dart_Port global_dart_send_port_id = 0;

// A C function to set the Dart_PostCObject function pointer
static void set_dart_post_c_object(Dart_PostCObject_Type func) {
    Dart_PostCObject_Fn = func;
}

// A C function to set the global Dart SendPort ID
static void set_global_dart_send_port_id(Dart_Port port_id) {
    global_dart_send_port_id = port_id;
}

// A C function to post a message to Dart
// type: 0 for progress update
static void post_dart_message_from_c(int type, long current, long max, long bps) {
    if (Dart_PostCObject_Fn == NULL || global_dart_send_port_id == 0) {
        return; // NativeApi not initialized or SendPort not set
    }

    // Allocate Dart_CObject for the array and its elements
    Dart_CObject* message = (Dart_CObject*)malloc(sizeof(Dart_CObject));
    if (message == NULL) return; // Handle allocation failure
    message->type = Dart_CObject_kArray; // Should be 6
    message->value.as_array.length = 4;
    message->value.as_array.values = (Dart_CObject**)malloc(sizeof(Dart_CObject*) * 4);
    if (message->value.as_array.values == NULL) {
        free(message);
        return; // Handle allocation failure
    }

    Dart_CObject* type_obj = (Dart_CObject*)malloc(sizeof(Dart_CObject));
    if (type_obj == NULL) goto cleanup_array_values;
    type_obj->type = Dart_CObject_kInt64; // Should be 3
    type_obj->value.as_int64 = type;
    message->value.as_array.values[0] = type_obj;

    Dart_CObject* current_obj = (Dart_CObject*)malloc(sizeof(Dart_CObject));
    if (current_obj == NULL) goto cleanup_array_values;
    current_obj->type = Dart_CObject_kInt64; // Should be 3
    current_obj->value.as_int64 = current;
    message->value.as_array.values[1] = current_obj;

    Dart_CObject* max_obj = (Dart_CObject*)malloc(sizeof(Dart_CObject));
    if (max_obj == NULL) goto cleanup_array_values;
    max_obj->type = Dart_CObject_kInt64; // Should be 3
    max_obj->value.as_int64 = max;
    message->value.as_array.values[2] = max_obj;

    Dart_CObject* bps_obj = (Dart_CObject*)malloc(sizeof(Dart_CObject));
    if (bps_obj == NULL) goto cleanup_array_values;
    bps_obj->type = Dart_CObject_kInt64; // Should be 3
    bps_obj->value.as_int64 = bps;
    message->value.as_array.values[3] = bps_obj;

    Dart_PostCObject_Fn(global_dart_send_port_id, message);

cleanup_array_values:
    // Free the allocated memory for the array values and the message itself
    for (int i = 0; i < message->value.as_array.length; ++i) {
        if (message->value.as_array.values[i] != NULL) {
            free(message->value.as_array.values[i]);
        }
    }
    free(message->value.as_array.values);
    free(message);
}

// A C function to call the Go-provided C callback
typedef void (*progressCallback)(long current, long max, long bps);

static inline void callProgressCallback(progressCallback cb, long current, long max, long bps) {
    if (cb != NULL) {
        cb(current, max, bps);
    }
}
*/
import "C"
import (
	"encoding/json"
	"fmt"
	"os"
	"unsafe"

	"samsung-firmware-tool/internal/cryptutils"
	"samsung-firmware-tool/internal/fusclient"
	"samsung-firmware-tool/internal/request"
	"samsung-firmware-tool/internal/util"
	"samsung-firmware-tool/internal/versionfetch"
)

// Result struct for JSON output
type Result struct {
	Success bool        `json:"success"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

//export SetDartPostCObject
func SetDartPostCObject(ptr unsafe.Pointer) {
	C.set_dart_post_c_object((C.Dart_PostCObject_Type)(ptr))
	fmt.Printf("Dart_PostCObject function pointer set.\n")
}

//export SetDartSendPortID
func SetDartSendPortID(portID C.longlong) {
	C.set_global_dart_send_port_id(C.Dart_Port(portID))
	fmt.Printf("Global Dart SendPort ID set to: %d\n", portID)
}

//export CheckFirmwareVersion
func CheckFirmwareVersion(modelC *C.char, regionC *C.char) *C.char {
	model := C.GoString(modelC)
	region := C.GoString(regionC)

	if model == "" || region == "" {
		res := Result{Success: false, Message: "错误: model 和 region 是必需的。"}
		jsonRes, _ := json.Marshal(res)
		return C.CString(string(jsonRes))
	}

	fmt.Printf("Checking latest version for Model: %s, Region: %s\n", model, region)
	result := versionfetch.GetLatestVersion(model, region)

	if result.Error != nil {
		res := Result{Success: false, Message: fmt.Sprintf("Error checking version: %v, Raw output: %s", result.Error, result.RawOutput)}
		jsonRes, _ := json.Marshal(res)
		return C.CString(string(jsonRes))
	}

	data := map[string]string{
		"versionCode":    result.VersionCode,
		"androidVersion": result.AndroidVersion,
	}
	res := Result{Success: true, Message: "版本检查成功", Data: data}
	jsonRes, _ := json.Marshal(res)
	return C.CString(string(jsonRes))
}

//export DownloadFirmware
func DownloadFirmware(modelC *C.char, regionC *C.char, fwVersionC *C.char, imeiSerialC *C.char, outputPathC *C.char) *C.char {
	model := C.GoString(modelC)
	region := C.GoString(regionC)
	fwVersion := C.GoString(fwVersionC)
	imeiSerial := C.GoString(imeiSerialC)
	outputPath := C.GoString(outputPathC)

	if model == "" || region == "" || fwVersion == "" || imeiSerial == "" || outputPath == "" {
		res := Result{Success: false, Message: "错误: model, region, fwVersion, imeiSerial, 和 outputPath 是下载固件所必需的。"}
		jsonRes, _ := json.Marshal(res)
		return C.CString(string(jsonRes))
	}

	fmt.Printf("Downloading firmware %s for Model: %s, Region: %s to %s\n", fwVersion, model, region, outputPath)

	client := fusclient.NewFusClient()

	onFinish := func(msg string) {
		fmt.Println(msg)
	}
	onVersionException := func(err error, info *request.BinaryFileInfo) {
		fmt.Printf("Version exception: %v\n", err)
		if info != nil {
			fmt.Println("Attempting to proceed with download despite version exception...")
			performDownloadGo(info, client, outputPath) // No sendPortID here
		}
	}
	shouldReportError := func(err error) bool {
		return true // For now, always report
	}

	binaryInfo := request.RetrieveBinaryFileInfo(fwVersion, model, region, imeiSerial, client, onFinish, onVersionException, shouldReportError)
	if binaryInfo == nil {
		res := Result{Success: false, Message: "Failed to retrieve binary file information."}
		jsonRes, _ := json.Marshal(res)
		return C.CString(string(jsonRes))
	}

	err := performDownloadGo(binaryInfo, client, outputPath) // No sendPortID here
	if err != nil {
		res := Result{Success: false, Message: fmt.Sprintf("Error during download: %v", err)}
		jsonRes, _ := json.Marshal(res)
		return C.CString(string(jsonRes))
	}

	res := Result{Success: true, Message: "固件下载成功", Data: map[string]string{"filePath": outputPath + "/" + binaryInfo.FileName}}
	jsonRes, _ := json.Marshal(res)
	return C.CString(string(jsonRes))
}

func performDownloadGo(binaryInfo *request.BinaryFileInfo, client *fusclient.FusClient, outputPath string) error {
	outputFile, err := os.Create(outputPath + "/" + binaryInfo.FileName)
	if err != nil {
		return fmt.Errorf("Error creating output file: %v", err)
	}
	defer outputFile.Close()

	// Perform BINARY_INIT request as in Kotlin version
	nonce, err := client.GetNonce()
	if err != nil {
		return fmt.Errorf("Error getting nonce for BINARY_INIT: %v", err)
	}
	binaryInitRequest := request.CreateBinaryInit(binaryInfo.FileName, nonce)
	_, err = client.MakeReq(fusclient.BinaryInit, binaryInitRequest, true)
	if err != nil {
		return fmt.Errorf("Error performing BINARY_INIT request: %v", err)
	}

	progressCallback := func(current, max, bps int64) {
		// Call the C function to post messages to Dart
		C.post_dart_message_from_c(0, C.long(current), C.long(max), C.long(bps))
	}

	md5Sum, err := client.DownloadFile(binaryInfo.Path+binaryInfo.FileName, 0, binaryInfo.Size, outputFile, 0, progressCallback)
	if err != nil {
		return fmt.Errorf("\nError downloading file: %v", err)
	}
	fmt.Printf("\nDownload complete. MD5: %s\n", md5Sum)
	return nil
}

//export DecryptFirmware
func DecryptFirmware(inputPathC *C.char, outputPathC *C.char, fwVersionC *C.char, modelC *C.char, regionC *C.char, imeiSerialC *C.char) *C.char {
	inputPath := C.GoString(inputPathC)
	outputPath := C.GoString(outputPathC)
	fwVersion := C.GoString(fwVersionC)
	model := C.GoString(modelC)
	region := C.GoString(regionC)
	imeiSerial := C.GoString(imeiSerialC)

	if inputPath == "" || outputPath == "" || fwVersion == "" || model == "" || region == "" || imeiSerial == "" {
		res := Result{Success: false, Message: "错误: inputPath, outputPath, fwVersion, model, region, 和 imeiSerial 是解码固件所必需的。"}
		jsonRes, _ := json.Marshal(res)
		return C.CString(string(jsonRes))
	}

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
		res := Result{Success: false, Message: "Failed to retrieve binary file information for decryption key."}
		jsonRes, _ := json.Marshal(res)
		return C.CString(string(jsonRes))
	}

	var decryptionKey []byte
	var decryptionKeyStr string

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
		res := Result{Success: false, Message: fmt.Sprintf("Error opening input file: %v", err)}
		jsonRes, _ := json.Marshal(res)
		return C.CString(string(jsonRes))
	}
	defer inputFile.Close()

	outputFile, err := os.Create(outputPath)
	if err != nil {
		res := Result{Success: false, Message: fmt.Sprintf("Error creating output file: %v", err)}
		jsonRes, _ := json.Marshal(res)
		return C.CString(string(jsonRes))
	}
	defer outputFile.Close()

	inputStat, err := inputFile.Stat()
	if err != nil {
		res := Result{Success: false, Message: fmt.Sprintf("Error getting input file info: %v", err)}
		jsonRes, _ := json.Marshal(res)
		return C.CString(string(jsonRes))
	}
	fileSize := inputStat.Size()

	progressCallback := func(current, max, bps int64) {
		// Call the C function to post messages to Dart
		C.post_dart_message_from_c(0, C.long(current), C.long(max), C.long(bps))
	}

	err = cryptutils.DecryptProgress(inputFile, outputFile, decryptionKey, fileSize, util.DEFAULT_CHUNK_SIZE, progressCallback)
	if err != nil {
		res := Result{Success: false, Message: fmt.Sprintf("\nError decrypting file: %v", err)}
		jsonRes, _ := json.Marshal(res)
		return C.CString(string(jsonRes))
	}
	fmt.Println("\nDecryption complete.")

	res := Result{Success: true, Message: "固件解密成功", Data: map[string]string{"outputPath": outputPath}}
	jsonRes, _ := json.Marshal(res)
	return C.CString(string(jsonRes))
}

// FreeString is a C-callable function to free memory allocated by C.CString
// This is important to prevent memory leaks when C code calls Go functions
// that return C strings.
//
//export FreeString
func FreeString(ptr *C.char) {
	C.free(unsafe.Pointer(ptr))
}

func main() {
	// Required for c-shared build mode, but can be empty.
}
