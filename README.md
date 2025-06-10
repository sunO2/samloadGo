# samloadGo

## 项目简介

`samloadGo` 是一个用 Go 语言编写的命令行工具，旨在为三星设备用户和开发者提供便捷的固件查询、下载和解密能力。通过简单的参数配置，即可获取最新的官方固件，支持线下刷机和固件分析。

## 项目用途

- 查询三星设备最新官方固件版本
- 下载固件包（支持断点续传）
- 解密官方加密固件，便于刷机或二次分析

## 安装方式

### 环境要求

- Go 1.18 及以上版本
- 能正常访问外网

### 编译安装

1. 克隆项目源码

    ```bash
    git clone https://github.com/sunO2/samloadGo.git
    cd samloadGo
    ```

2. 编译生成可执行文件

    ```bash
    go build -o samloadGo main.go
    ```

3. 现在你可以在当前目录下找到 `samloadGo` 可执行文件。

## 作为动态库使用

除了作为命令行工具，`samloadGo` 的核心功能（查询、下载、解密）也可以编译为动态库，方便其他语言和应用集成。

### 构建动态库

在 `samsung-firmware-tool` 项目根目录下执行以下命令：

```bash
# Linux (生成 .so 文件)
GOOS=linux GOARCH=amd64 go build -buildmode=c-shared -o build/libfirmwarelib.so pkg/firmwarelib/firmwarelib.go

# macOS (生成 .dylib 文件)
GOOS=darwin GOARCH=amd64 go build -buildmode=c-shared -o build/libfirmwarelib.dylib pkg/firmwarelib/firmwarelib.go

# Windows (生成 .dll 文件)
GOOS=windows GOARCH=amd64 go build -buildmode=c-shared -o build/firmwarelib.dll pkg/firmwarelib/firmwarelib.go

# Android (生成 .so 文件，需要 Android NDK)
# 确保已设置 ANDROID_NDK_HOME 环境变量，例如：export ANDROID_NDK_HOME=/path/to/android-ndk-r25b
# arm64-v8a 架构
GOOS=android GOARCH=arm64 CGO_ENABLED=1 CC=$ANDROID_NDK_HOME/toolchains/llvm/prebuilt/linux-x86_64/bin/aarch64-linux-android21-clang go build -buildmode=c-shared -o build/libfirmwarelib_arm64.so pkg/firmwarelib/firmwarelib.go
# armeabi-v7a 架构
GOOS=android GOARCH=arm CGO_ENABLED=1 CC=$ANDROID_NDK_HOME/toolchains/llvm/prebuilt/linux-x86_64/bin/armv7a-linux-androideabi21-clang go build -buildmode=c-shared -o build/libfirmwarelib_arm.so pkg/firmwarelib/firmwarelib.go
```

编译成功后，动态库文件 (`.so`, `.dylib`, `.dll`) 和对应的 C 头文件 (`libfirmwarelib.h`) 将生成在 `samsung-firmware-tool/build/` 目录下。

### C/C++ 语言集成示例

以下是一个简单的 C 语言示例，展示如何加载动态库并调用 `CheckFirmwareVersion`、`DownloadFirmware` 和 `DecryptFirmware` 函数，并接收进度回调。

**`example.c`:**
```c
#include "libfirmwarelib.h" // 包含生成的头文件
#include <stdio.h>
#include <stdlib.h> // For free

// C 语言实现的进度回调函数
void myProgressCallback(long current, long max, long bps) {
    printf("\rProgress: %ld/%ld bytes (%.2f%%) @ %ld B/s", current, max, (float)current / max * 100, bps);
    fflush(stdout); // 确保立即刷新输出
}

int main() {
    // 示例：查询最新固件版本
    char* model_check = "SM-G998B";
    char* region_check = "EUX";
    char* check_result_json = CheckFirmwareVersion(model_check, region_check);
    printf("CheckFirmwareVersion Result: %s\n", check_result_json);
    FreeString(check_result_json); // 释放 Go 分配的内存

    // 示例：下载固件 (需要替换为实际参数)
    // char* model_download = "SM-G998B";
    // char* region_download = "EUX";
    // char* fwVersion_download = "G998BXXU1AUAE"; // 替换为实际固件版本
    // char* imeiSerial_download = "123456789012345"; // 替换为实际 IMEI/序列号
    // char* outputPath_download = "/tmp"; // 替换为有效的输出目录

    // char* download_result_json = DownloadFirmware(
    //     model_download,
    //     region_download,
    //     fwVersion_download,
    //     imeiSerial_download,
    //     outputPath_download,
    //     myProgressCallback // 传入 C 函数指针作为回调
    // );
    // printf("\nDownloadFirmware Result: %s\n", download_result_json);
    // FreeString(download_result_json); // 释放 Go 分配的内存

    // 示例：解密固件 (需要替换为实际参数)
    // char* inputPath_decrypt = "/path/to/input.enc4"; // 替换为实际输入文件路径
    // char* outputPath_decrypt = "/path/to/output.tar.md5"; // 替换为实际输出文件路径
    // char* fwVersion_decrypt = "G998BXXU1AUAE"; // 替换为实际固件版本
    // char* model_decrypt = "SM-G998B";
    // char* region_decrypt = "EUX";
    // char* imeiSerial_decrypt = "123456789012345";

    // char* decrypt_result_json = DecryptFirmware(
    //     inputPath_decrypt,
    //     outputPath_decrypt,
    //     fwVersion_decrypt,
    //     model_decrypt,
    //     region_decrypt,
    //     imeiSerial_decrypt,
    //     myProgressCallback // 传入 C 函数指针作为回调
    // );
    // printf("\nDecryptFirmware Result: %s\n", decrypt_result_json);
    // FreeString(decrypt_result_json); // 释放 Go 分配的内存

    return 0;
}
```

**编译 C/C++ 示例：**

在 Linux 或 macOS 上，使用 `gcc` 编译：
```bash
# 假设 libfirmwarelib.so/.dylib 和 libfirmwarelib.h 在当前目录或系统库路径中
gcc example.c -L./build -lfirmwarelib -o example
```
在 Windows 上，使用 `MinGW-w64` 或 `MSVC` 编译：
```bash
# MinGW-w64
gcc example.c -L./build -lfirmwarelib -o example.exe

# MSVC (需要配置环境)
# cl example.c /link /LIBPATH:./build firmwarelib.lib
```

### 其他语言集成

- **Python**: 可以使用 `ctypes` 模块加载动态库并调用 C 接口。
- **Java**: 可以使用 Java Native Interface (JNI) 或 Java Native Access (JNA) 来加载动态库并调用 C 接口。
- **Node.js**: 可以使用 `node-ffi-napi` 或 `N-API` 来加载动态库并调用 C 接口。

## 使用说明

### 基本参数

| 参数         | 说明                    | 示例                |
| ------------ | ----------------------- | ------------------- |
| --model      | 设备型号                | SM-G998U            |
| --region     | 地区代码                | XAA                 |
| --imei       | 设备 IMEI（部分必需）   | 123456789012345     |
| --fw         | 固件版本（部分必需）    | G998USQU4AUF5       |
| --output     | 输出文件路径            | ./firmware.zip      |
| --input      | 输入文件路径（解密用）  | ./firmware.zip.enc4 |

### 操作参数

- `--check-version` 查询最新固件版本
- `--download` 下载固件
- `--decrypt` 解密固件

### 快速示例

1. **查询最新固件版本**

    ```bash
    ./samloadGo --model SM-G998U --region XAA --check-version
    ```

2. **下载固件**

    ```bash
    ./samloadGo --model SM-G998U --region XAA --fw G998USQU4AUF5 --imei 123456789012345 --output ./firmware.zip --download
    ```

3. **解密固件**

    ```bash
    ./samloadGo --model SM-G998U --region XAA --fw G998USQU4AUF5 --imei 123456789012345 --input ./firmware.zip.enc4 --output ./firmware.zip --decrypt
    ```

4. **交互模式启动**

    ```bash
    ./samloadGo
    ```

    未传递参数时，程序会进入交互模式，按提示输入相关信息即可。

### 高级说明

- 所有网络请求均直连三星官方固件服务器，数据安全可靠。
- 下载和解密均支持大文件分块处理，稳健高效。
- 如果遇到参数缺失，程序会自动提示补充。

## 常见问题

- 若遇到网络连接问题，请检查本地网络环境及代理设置。
- 如有固件解密失败，建议核查输入文件及参数完整性。

## 贡献与许可

欢迎提交 issue 和 PR 共同完善本项目！

本项目采用 MIT 许可证，详见 [LICENSE](LICENSE)。

---

**作者**：[sunO2](https://github.com/sunO2)
