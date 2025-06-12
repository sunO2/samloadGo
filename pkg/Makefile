# Makefile for building firmwarelib as a shared library for different platforms

# Go package to build
PACKAGE = ./firmwarelib

# Output directory for shared libraries
BUILD_DIR = ../build

ANDROID_NDK_HOME = /Users/ex-hezhihu870/develop/android/sdk/ndk/22.1.7171670
ANDROID_LIB_DIR = ./../../samsung-firmware-tool-client/firmware_client/android/app/src/main/jniLibs
# Target shared library name
LIB_NAME = firmwarelib

.PHONY: all clean linux-amd64 linux-arm64 windows-amd64 darwin-amd64 darwin-arm64 android-arm android-arm64

all: linux-amd64 linux-arm64 windows-amd64 darwin-amd64 darwin-arm64 android-arm android-arm64

$(BUILD_DIR):
	@mkdir -p $(BUILD_DIR)

# Linux AMD64
linux-amd64: $(BUILD_DIR)
	@echo "Building for Linux AMD64..."
	GOOS=linux GOARCH=amd64 go build -buildmode=c-shared -o $(BUILD_DIR)/lib$(LIB_NAME).so $(PACKAGE)
	@echo "Linux AMD64 build complete: $(BUILD_DIR)/lib$(LIB_NAME).so"

# Linux ARM64
linux-arm64: $(BUILD_DIR)
	@echo "Building for Linux ARM64..."
	GOOS=linux GOARCH=arm64 go build -buildmode=c-shared -o $(BUILD_DIR)/lib$(LIB_NAME)_arm64.so $(PACKAGE)
	@echo "Linux ARM64 build complete: $(BUILD_DIR)/lib$(LIB_NAME)_arm64.so"

# Windows AMD64
windows-amd64: $(BUILD_DIR)
	@echo "Building for Windows AMD64..."
	GOOS=windows GOARCH=amd64 go build -buildmode=c-shared -o $(BUILD_DIR)/$(LIB_NAME).dll $(PACKAGE)
	@echo "Windows AMD64 build complete: $(BUILD_DIR)/$(LIB_NAME).dll"

# macOS AMD64
darwin-amd64: $(BUILD_DIR)
	@echo "Building for macOS AMD64..."
	GOOS=darwin GOARCH=amd64 go build -buildmode=c-shared -o $(BUILD_DIR)/lib$(LIB_NAME)_amd64.dylib $(PACKAGE)
	@echo "macOS AMD64 build complete: $(BUILD_DIR)/lib$(LIB_NAME)_amd64.dylib"

# macOS ARM64 (Apple Silicon)
darwin-arm64: $(BUILD_DIR)
	@echo "Building for macOS ARM64..."
	GOOS=darwin GOARCH=arm64 go build -buildmode=c-shared -o $(BUILD_DIR)/lib$(LIB_NAME)_arm64.dylib $(PACKAGE)
	@echo "macOS ARM64 build complete: $(BUILD_DIR)/lib$(LIB_NAME)_arm64.dylib"

# Android ARM
android-arm: $(BUILD_DIR)
	@echo "Building for Android ARM..."
	GOOS=android GOARCH=arm CGO_ENABLED=1 CC=$(ANDROID_NDK_HOME)/toolchains/llvm/prebuilt/darwin-x86_64/bin/armv7a-linux-androideabi21-clang \
	go build -buildmode=c-shared -ldflags="-s -w" -o $(ANDROID_LIB_DIR)/armeabi-v7a/lib$(LIB_NAME).so $(PACKAGE)
	@echo "Android ARM build complete: $(ANDROID_LIB_DIR)/armeabi-v7a/lib$(LIB_NAME).so"
	file $(ANDROID_LIB_DIR)/armeabi-v7a/lib$(LIB_NAME).so

# Android ARM64
android-arm64: $(BUILD_DIR)
	@echo "Building for Android ARM64..."
	GOOS=android GOARCH=arm64 CGO_ENABLED=1 CC=$(ANDROID_NDK_HOME)/toolchains/llvm/prebuilt/darwin-x86_64/bin/aarch64-linux-android21-clang \
	go build -buildmode=c-shared -ldflags="-s -w" -o $(ANDROID_LIB_DIR)/arm64-v8a/lib$(LIB_NAME).so $(PACKAGE)
	@echo "Android ARM64 build complete: $(ANDROID_LIB_DIR)/arm64-v8a/lib$(LIB_NAME).so"
	file $(ANDROID_LIB_DIR)/arm64-v8a/lib$(LIB_NAME).so

android: android-arm64 android-arm

clean:
	@echo "Cleaning build directory..."
	@rm -rf $(BUILD_DIR)/*.so $(BUILD_DIR)/*.dll $(BUILD_DIR)/*.dylib $(BUILD_DIR)/*.h
	@echo "Clean complete."
