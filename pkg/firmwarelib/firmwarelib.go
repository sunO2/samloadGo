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

// A C struct to hold the Dart callback information
typedef struct {
    Dart_Port send_port_id;
    Dart_PostCObject_Type post_c_object_fn;
} Dart_Callback_Handle;

// A C function to post a message to Dart using the provided handle
// type: 0 for progress update
static void post_dart_message_from_c(Dart_Callback_Handle* handle, int type, long current, long max, long bps) {
    if (handle == NULL || handle->post_c_object_fn == NULL || handle->send_port_id == 0) {
        return; // Callback handle not initialized or SendPort not set
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

    handle->post_c_object_fn(handle->send_port_id, message);

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
	"unsafe"

	"samsung-firmware-tool/cmd"
	"samsung-firmware-tool/internal/versionfetch"
)

// Result struct for JSON output
type Result struct {
	Success bool        `json:"success"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

var downloadManagerMap = make(map[string]*cmd.DownloadTask)

//export NewDartCallbackHandle
func NewDartCallbackHandle(sendPortID C.longlong, postCObjectPtr unsafe.Pointer) *C.Dart_Callback_Handle {
	handle := (*C.Dart_Callback_Handle)(C.malloc(C.sizeof_Dart_Callback_Handle))
	if handle == nil {
		return nil // Handle allocation failure
	}
	handle.send_port_id = C.Dart_Port(sendPortID)
	handle.post_c_object_fn = (C.Dart_PostCObject_Type)(postCObjectPtr)
	fmt.Printf("New Dart_Callback_Handle created with SendPortID: %d\n", sendPortID)
	return handle
}

//export FreeDartCallbackHandle
func FreeDartCallbackHandle(handle *C.Dart_Callback_Handle) {
	if handle != nil {
		C.free(unsafe.Pointer(handle))
		fmt.Println("Dart_Callback_Handle freed.")
	}
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
func DownloadFirmware(modelC *C.char, regionC *C.char, fwVersionC *C.char, imeiSerialC *C.char, outputPathC *C.char, callbackHandle *C.Dart_Callback_Handle) *C.char {
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

	/// 相同id不进入
	taskId := model + region + fwVersion + imeiSerial + outputPath
	if task, exists := downloadManagerMap[taskId]; exists {
		if task.Status >= cmd.StatusInitializing && task.Status < cmd.StatusFailed {
			res := Result{Success: false, Message: "下载中..."}
			jsonRes, _ := json.Marshal(res)
			return C.CString(string(jsonRes))
		}
	}

	fmt.Printf("Downloading firmware %s for Model: %s, Region: %s to %s\n", fwVersion, model, region, outputPath)
	progressCallback := func(current, max, bps int64) {
		C.post_dart_message_from_c(callbackHandle, 0, C.long(current), C.long(max), C.long(bps))
	}

	task := cmd.NewDownloadTask(model, region, fwVersion, imeiSerial, outputPath, progressCallback)
	downloadManagerMap[taskId] = task
	err := task.Start()
	if nil != err {
		res := Result{Success: false, Message: err.Error()}
		jsonRes, _ := json.Marshal(res)
		return C.CString(string(jsonRes))
	}

	res := Result{Success: true, Message: "固件下载成功", Data: map[string]string{"filePath": outputPath + "/" + task.FileName}}
	jsonRes, _ := json.Marshal(res)
	return C.CString(string(jsonRes))
}

//export DecryptFirmware
func DecryptFirmware(inputPathC *C.char, outputPathC *C.char, fwVersionC *C.char, modelC *C.char, regionC *C.char, imeiSerialC *C.char, callbackHandle *C.Dart_Callback_Handle) *C.char {
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

	progressCallback := func(current, max, bps int64) {

	}
	err := cmd.DecryptFirmware(inputPath, outputPath, fwVersion, model, region, imeiSerial, progressCallback)
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
