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