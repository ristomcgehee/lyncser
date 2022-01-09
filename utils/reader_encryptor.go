package utils

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"io"
	"io/ioutil"
)

type ReaderEncryptor interface {
	EncryptReader(reader io.Reader) (io.Reader, error)
	DecryptReader(reader io.ReadCloser) (io.ReadCloser, error)
}

type AESGCMEncryptor struct {
	Key []byte
}

func (e *AESGCMEncryptor) EncryptReader(reader io.Reader) (io.Reader, error) {
	plaintext, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}
	block, err := aes.NewCipher(e.Key)
	if err != nil {
		return nil, err
	}
	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, aesGCM.NonceSize())
	ciphertext := aesGCM.Seal(nonce, nonce, plaintext, nil)
	readerEncrypted := bytes.NewReader(ciphertext)
	if err != nil {
		return nil, err
	}

	return readerEncrypted, nil
}

func (e *AESGCMEncryptor) DecryptReader(reader io.ReadCloser) (io.ReadCloser, error) {
	encryptedData, err := ioutil.ReadAll(reader)
	if err != nil {
		return nil, err
	}

	block, err := aes.NewCipher(e.Key)
	if err != nil {
		return nil, err
	}
	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonceSize := aesGCM.NonceSize()
	nonce, ciphertext := encryptedData[:nonceSize], encryptedData[nonceSize:]
	plaintext, err := aesGCM.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, err
	}
	decryptedReader := io.NopCloser(bytes.NewReader(plaintext))

	return decryptedReader, nil
}
