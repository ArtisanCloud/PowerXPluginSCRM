package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
)

// DeriveKey32 使用 SHA-256 将任意长度密钥材料导出为 32 字节对称密钥
func DeriveKey32(material string) []byte {
	sum := sha256.Sum256([]byte(material))
	return sum[:]
}

// EncryptAESGCM 对明文进行 AES-GCM 加密，返回 (密文, nonce)
func EncryptAESGCM(key []byte, plaintext []byte, aad []byte) (ciphertext []byte, nonce []byte, err error) {
	if len(key) != 16 && len(key) != 24 && len(key) != 32 {
		return nil, nil, fmt.Errorf("invalid key length: %d", len(key))
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, nil, err
	}
	nonce = make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, nil, err
	}
	ciphertext = gcm.Seal(nil, nonce, plaintext, aad)
	return ciphertext, nonce, nil
}

// DecryptAESGCM 解密 AES-GCM 密文
func DecryptAESGCM(key []byte, ciphertext []byte, nonce []byte, aad []byte) ([]byte, error) {
	if len(key) != 16 && len(key) != 24 && len(key) != 32 {
		return nil, fmt.Errorf("invalid key length: %d", len(key))
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	if len(nonce) != gcm.NonceSize() {
		return nil, errors.New("invalid nonce size")
	}
	return gcm.Open(nil, nonce, ciphertext, aad)
}
