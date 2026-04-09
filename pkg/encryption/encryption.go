package encryption

import (
	"crypto"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"errors"
	"io"
	"strings"
)

func EncryptHybrid(plaintext string, pubKeyStr string) (string, error) {
	if plaintext == "" {
		return "", nil
	}

	aesKey := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, aesKey); err != nil {
		return "", errors.New("error generating AES key: " + err.Error())
	}

	block, err := aes.NewCipher(aesKey)
	if err != nil {
		return "", errors.New("error creating AES cipher: " + err.Error())
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "", errors.New("error creating GCM: " + err.Error())
	}

	nonce := make([]byte, aesGCM.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", errors.New("error generating nonce: " + err.Error())
	}

	encryptedData := aesGCM.Seal(nonce, nonce, []byte(plaintext), nil)

	encryptedAESKey, err := encryptRSA(aesKey, pubKeyStr)
	if err != nil {
		return "", errors.New("error encrypting AES key with RSA: " + err.Error())
	}

	result := base64.StdEncoding.EncodeToString(encryptedAESKey) + ":" + base64.StdEncoding.EncodeToString(encryptedData)
	return result, nil
}

func DecryptHybrid(ciphertext string, privateKey string) (string, error) {
	if ciphertext == "" {
		return "", nil
	}

	parts := strings.SplitN(ciphertext, ":", 2)
	if len(parts) != 2 {
		return "", errors.New("invalid hybrid ciphertext format")
	}

	encryptedAESKey, err := base64.StdEncoding.DecodeString(parts[0])
	if err != nil {
		return "", errors.New("error decoding AES key: " + err.Error())
	}

	decodedPrivateKey, err := base64.StdEncoding.DecodeString(privateKey)
	if err != nil {
		return "", errors.New("error decoding private key: " + err.Error())
	}

	var rsaPrivateKey *rsa.PrivateKey
	pkcs8privateKey, err := x509.ParsePKCS8PrivateKey(decodedPrivateKey)
	if err != nil {
		rsaPrivateKey, err = x509.ParsePKCS1PrivateKey(decodedPrivateKey)
		if err != nil {
			return "", errors.New("error parsing private key: " + err.Error())
		}
	} else {
		var ok bool
		rsaPrivateKey, ok = pkcs8privateKey.(*rsa.PrivateKey)
		if !ok {
			return "", errors.New("private key is not RSA")
		}
	}

	aesKey, err := rsaPrivateKey.Decrypt(rand.Reader, encryptedAESKey, &rsa.OAEPOptions{Hash: crypto.SHA256})
	if err != nil {
		return "", errors.New("error decrypting AES key: " + err.Error())
	}

	encryptedData, err := base64.StdEncoding.DecodeString(parts[1])
	if err != nil {
		return "", errors.New("error decoding encrypted data: " + err.Error())
	}

	block, err := aes.NewCipher(aesKey)
	if err != nil {
		return "", errors.New("error creating AES cipher: " + err.Error())
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "", errors.New("error creating GCM: " + err.Error())
	}

	nonceSize := aesGCM.NonceSize()
	if len(encryptedData) < nonceSize {
		return "", errors.New("encrypted data too short")
	}

	nonce, encryptedDataBody := encryptedData[:nonceSize], encryptedData[nonceSize:]
	decryptedData, err := aesGCM.Open(nil, nonce, encryptedDataBody, nil)
	if err != nil {
		return "", errors.New("error decrypting data: " + err.Error())
	}

	return string(decryptedData), nil
}

func DecryptRSAOAEP(ciphertext string, privateKeyB64 string) (string, error) {
	if ciphertext == "" {
		return "", nil
	}

	decodedMessage, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", errors.New("error decoding ciphertext: " + err.Error())
	}

	decodedPrivateKey, err := base64.StdEncoding.DecodeString(privateKeyB64)
	if err != nil {
		return "", errors.New("error decoding private key: " + err.Error())
	}

	var rsaPrivateKey *rsa.PrivateKey
	pkcs8Key, err := x509.ParsePKCS8PrivateKey(decodedPrivateKey)
	if err != nil {
		rsaPrivateKey, err = x509.ParsePKCS1PrivateKey(decodedPrivateKey)
		if err != nil {
			return "", errors.New("error parsing private key: " + err.Error())
		}
	} else {
		var ok bool
		rsaPrivateKey, ok = pkcs8Key.(*rsa.PrivateKey)
		if !ok {
			return "", errors.New("private key is not RSA")
		}
	}

	decrypted, err := rsaPrivateKey.Decrypt(rand.Reader, decodedMessage, &rsa.OAEPOptions{Hash: crypto.SHA256, MGFHash: crypto.SHA256})
	if err != nil {
		return "", errors.New("error decrypting: " + err.Error())
	}

	return string(decrypted), nil
}

func EncryptRSAOAEP(plaintext string, pubKeyB64 string) (string, error) {
	if plaintext == "" {
		return "", nil
	}

	pubKey, err := parseRSAPublicKey(pubKeyB64)
	if err != nil {
		return "", err
	}

	encrypted, err := rsa.EncryptOAEP(sha256.New(), rand.Reader, pubKey, []byte(plaintext), nil)
	if err != nil {
		return "", errors.New("error encrypting: " + err.Error())
	}

	return base64.StdEncoding.EncodeToString(encrypted), nil
}

func encryptRSA(data []byte, pubKeyStr string) ([]byte, error) {
	pubKey, err := parseRSAPublicKey(pubKeyStr)
	if err != nil {
		return nil, err
	}
	return rsa.EncryptOAEP(sha256.New(), rand.Reader, pubKey, data, nil)
}

func parseRSAPublicKey(keyStr string) (*rsa.PublicKey, error) {
	data, err := base64.RawStdEncoding.DecodeString(keyStr)
	if err != nil {
		return nil, errors.New("error decoding public key: " + err.Error())
	}

	pub, err := x509.ParsePKIXPublicKey(data)
	if err != nil {
		return nil, errors.New("error parsing public key: " + err.Error())
	}

	rsaPub, ok := pub.(*rsa.PublicKey)
	if !ok {
		return nil, errors.New("public key is not RSA")
	}
	return rsaPub, nil
}
