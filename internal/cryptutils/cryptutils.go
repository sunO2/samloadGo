package cryptutils

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/md5"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"hash" // Import hash package
	"hash/crc32"
	"io"
	"os"

	"samsung-firmware-tool/internal/util"
)

const (
	key1 = "vicopx7dqu06emacgpnpy8j8zwhduwlh"
	key2 = "9u7qab84rpc16gvk"
)

// unpad removes custom Samsung AES padding from data.
func unpad(d []byte) []byte {
	if len(d) == 0 {
		return d
	}
	lastByte := int(d[len(d)-1])
	padIndex := len(d) - lastByte
	if padIndex < 0 || padIndex > len(d) {
		// This indicates invalid padding, return original or error
		return d
	}
	return d[:padIndex]
}

// pad adds custom Samsung AES padding to data.
func pad(d []byte) []byte {
	blockSize := aes.BlockSize // AES block size is 16 bytes
	padding := blockSize - (len(d) % blockSize)
	if padding == 0 {
		padding = blockSize
	}
	padText := bytes.Repeat([]byte{byte(padding)}, padding)
	return append(d, padText...)
}

// aesEncrypt encrypts data using AES CBC with custom padding.
func aesEncrypt(input, key []byte) ([]byte, error) {
	paddedInput := pad(input)
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	iv := key[:aes.BlockSize] // IV is the first 16 bytes of the key
	mode := cipher.NewCBCEncrypter(block, iv)

	encrypted := make([]byte, len(paddedInput))
	mode.CryptBlocks(encrypted, paddedInput)

	return encrypted, nil
}

// aesDecrypt decrypts data using AES CBC with custom padding.
func aesDecrypt(input, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	iv := key[:aes.BlockSize] // IV is the first 16 bytes of the key
	mode := cipher.NewCBCDecrypter(block, iv)

	decrypted := make([]byte, len(input))
	mode.CryptBlocks(decrypted, input)

	return unpad(decrypted), nil
}

// getFKey generates a key given a specific input.
func getFKey(input []byte) []byte {
	key := ""
	for i := 0; i < 16; i++ {
		key += string(key1[int(input[i])%len(key1)])
	}
	key += key2
	return []byte(key)
}

// GetAuth generates an auth token with a given nonce.
func GetAuth(nonce string) (string, error) {
	keyData := make([]byte, len(nonce))
	for i, char := range nonce {
		keyData[i] = byte(int(char) % 16)
	}
	fKey := getFKey(keyData)

	encrypted, err := aesEncrypt([]byte(nonce), fKey)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(encrypted), nil
}

// DecryptNonce decrypts a provided nonce string.
func DecryptNonce(input string) (string, error) {
	d, err := base64.StdEncoding.DecodeString(input)
	if err != nil {
		return "", err
	}
	decrypted, err := aesDecrypt(d, []byte(key1))
	if err != nil {
		return "", err
	}
	return string(decrypted), nil
}

// getV2Key creates the decryption key for a .enc2 firmware file.
func GetV2Key(version, model, region string) ([]byte, string) {
	decKey := fmt.Sprintf("%s:%s:%s", region, model, version)
	hasher := md5.New()
	hasher.Write([]byte(decKey))
	return hasher.Sum(nil), decKey
}

// decryptProgress decrypts a provided file to a specified target, with a progress callback.
// This function assumes ECB mode for decryption as per Kotlin's aesEcbProvider.
func DecryptProgress(
	inf *os.File,
	outf *os.File,
	key []byte,
	length int64,
	chunkSize int,
	progressCallback func(current, max, bps int64),
) error {
	block, err := aes.NewCipher(key)
	if err != nil {
		return err
	}

	// Manual ECB decryption
	buf := make([]byte, chunkSize)
	totalRead := int64(0)

	for totalRead < length {
		n, err := inf.Read(buf)
		if err != nil && err != io.EOF {
			return err
		}
		if n == 0 {
			break
		}

		encryptedBlock := buf[:n]
		decryptedBlock := make([]byte, n)

		for i := 0; i < n; i += block.BlockSize() {
			block.Decrypt(decryptedBlock[i:], encryptedBlock[i:])
		}

		_, err = outf.Write(decryptedBlock)
		if err != nil {
			return err
		}

		totalRead += int64(n)
		progressCallback(totalRead, length, 0) // bps not implemented yet
	}

	return nil
}

// CheckCrc32 checks the CRC32 of a given encrypted firmware file.
func CheckCrc32(
	enc *os.File,
	encSize int64,
	expected uint32,
	progressCallback func(current, max, bps int64),
) (bool, error) {
	if enc == nil {
		return false, nil
	}

	buffer := make([]byte, util.DEFAULT_CHUNK_SIZE)
	crc := crc32.NewIEEE()
	totalRead := int64(0)

	for totalRead < encSize {
		n, err := enc.Read(buffer)
		if err != nil && err != io.EOF {
			return false, err
		}
		if n == 0 {
			break
		}

		crc.Write(buffer[:n])
		totalRead += int64(n)
		progressCallback(totalRead, encSize, 0) // bps not implemented yet
	}

	// Reset file pointer for future reads if needed
	_, err := enc.Seek(0, io.SeekStart)
	if err != nil {
		return false, err
	}

	return crc.Sum32() == expected, nil
}

// CheckMD5 checks an MD5 hash given an input file and an expected value.
func CheckMD5(md5Sum string, updateFile *os.File) (bool, error) {
	if md5Sum == "" || updateFile == nil {
		return false, nil
	}

	calculatedDigest, err := calculateMD5(updateFile)
	if err != nil {
		return false, err
	}

	// Reset file pointer for future reads if needed
	_, err = updateFile.Seek(0, io.SeekStart)
	if err != nil {
		return false, err
	}

	return bytes.EqualFold([]byte(calculatedDigest), []byte(md5Sum)), nil
}

// calculateMD5 calculates an MD5 hash for a given input file.
func calculateMD5(updateFile *os.File) (string, error) {
	hasher := md5.New()
	buffer := make([]byte, 8192) // 8KB buffer

	for {
		n, err := updateFile.Read(buffer)
		if err != nil && err != io.EOF {
			return "", err
		}
		if n == 0 {
			break
		}
		hasher.Write(buffer[:n])
	}

	return hex.EncodeToString(hasher.Sum(nil)), nil
}

// MD5Hasher returns a new MD5 hash.Hash.
func MD5Hasher() hash.Hash {
	return md5.New()
}
