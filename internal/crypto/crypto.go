package crypto

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
)

// Encryptor предоставляет методы для шифрования данных публичным ключом.
//
// Использует RSA с PKCS1v15 padding для шифрования данных.
type Encryptor struct {
	publicKey *rsa.PublicKey
}

// Decryptor предоставляет методы для дешифрования данных приватным ключом.
//
// Использует RSA с PKCS1v15 padding для дешифрования данных.
type Decryptor struct {
	privateKey *rsa.PrivateKey
}

// NewEncryptor создаёт новый Encryptor из файла с публичным ключом.
//
// Параметры:
//   - publicKeyPath: путь к файлу с публичным ключом в PEM формате
//
// Возвращает:
//   - *Encryptor: указатель на созданный шифратор
//   - error: ошибка при чтении или парсинге ключа
func NewEncryptor(publicKeyPath string) (*Encryptor, error) {
	publicKey, err := readPublicKey(publicKeyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read public key: %w", err)
	}

	return &Encryptor{
		publicKey: publicKey,
	}, nil
}

// NewDecryptor создаёт новый Decryptor из файла с приватным ключом.
//
// Параметры:
//   - privateKeyPath: путь к файлу с приватным ключом в PEM формате
//
// Возвращает:
//   - *Decryptor: указатель на созданный дешифратор
//   - error: ошибка при чтении или парсинге ключа
func NewDecryptor(privateKeyPath string) (*Decryptor, error) {
	privateKey, err := readPrivateKey(privateKeyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read private key: %w", err)
	}

	return &Decryptor{
		privateKey: privateKey,
	}, nil
}

// Encrypt шифрует данные публичным ключом.
//
// Использует RSA с PKCS1v15 padding.
//
// Параметры:
//   - data: данные для шифрования
//
// Возвращает:
//   - []byte: зашифрованные данные
//   - error: ошибка при шифровании
func (e *Encryptor) Encrypt(data []byte) ([]byte, error) {
	return rsa.EncryptPKCS1v15(rand.Reader, e.publicKey, data)
}

// Decrypt дешифрует данные приватным ключом.
//
// Использует RSA с PKCS1v15 padding.
//
// Параметры:
//   - data: зашифрованные данные
//
// Возвращает:
//   - []byte: дешифрованные данные
//   - error: ошибка при дешифровании
func (d *Decryptor) Decrypt(data []byte) ([]byte, error) {
	return rsa.DecryptPKCS1v15(rand.Reader, d.privateKey, data)
}

// readPublicKey читает публичный ключ из PEM файла.
//
// Поддерживает два формата:
//   - CERTIFICATE (сертификат x.509)
//   - RSA PUBLIC KEY (публичный RSA ключ)
//
// Параметры:
//   - path: путь к файлу с публичным ключом
//
// Возвращает:
//   - *rsa.PublicKey: публичный RSA ключ
//   - error: ошибка при чтении или парсинге
func readPublicKey(path string) (*rsa.PublicKey, error) {
	keyBytes, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read public key file: %w", err)
	}

	block, _ := pem.Decode(keyBytes)
	if block == nil {
		return nil, fmt.Errorf("failed to decode PEM block")
	}

	// Пытаемся распарсить как сертификат x.509
	if block.Type == "CERTIFICATE" {
		cert, err := x509.ParseCertificate(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("failed to parse certificate: %w", err)
		}
		publicKey, ok := cert.PublicKey.(*rsa.PublicKey)
		if !ok {
			return nil, fmt.Errorf("certificate does not contain RSA public key")
		}
		return publicKey, nil
	}

	// Пытаемся распарсить как PKIX публичный ключ
	if block.Type == "PUBLIC KEY" {
		pubKey, err := x509.ParsePKIXPublicKey(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("failed to parse PKIX public key: %w", err)
		}
		publicKey, ok := pubKey.(*rsa.PublicKey)
		if !ok {
			return nil, fmt.Errorf("key is not RSA public key")
		}
		return publicKey, nil
	}

	// Пытаемся распарсить как RSA публичный ключ (устаревший формат)
	if block.Type == "RSA PUBLIC KEY" {
		pubKey, err := x509.ParsePKCS1PublicKey(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("failed to parse PKCS1 public key: %w", err)
		}
		return pubKey, nil
	}

	return nil, fmt.Errorf("unsupported PEM block type: %s", block.Type)
}

// readPrivateKey читает приватный ключ из PEM файла.
//
// Поддерживает формат RSA PRIVATE KEY (PKCS1).
//
// Параметры:
//   - path: путь к файлу с приватным ключом
//
// Возвращает:
//   - *rsa.PrivateKey: приватный RSA ключ
//   - error: ошибка при чтении или парсинге
func readPrivateKey(path string) (*rsa.PrivateKey, error) {
	keyBytes, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read private key file: %w", err)
	}

	block, _ := pem.Decode(keyBytes)
	if block == nil {
		return nil, fmt.Errorf("failed to decode PEM block")
	}

	// Пытаемся распарсить как PKCS1 приватный ключ
	if block.Type == "RSA PRIVATE KEY" {
		privateKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("failed to parse PKCS1 private key: %w", err)
		}
		return privateKey, nil
	}

	// Пытаемся распарсить как PKCS8 приватный ключ
	if block.Type == "PRIVATE KEY" {
		privateKey, err := x509.ParsePKCS8PrivateKey(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("failed to parse PKCS8 private key: %w", err)
		}
		rsaPrivateKey, ok := privateKey.(*rsa.PrivateKey)
		if !ok {
			return nil, fmt.Errorf("key is not RSA private key")
		}
		return rsaPrivateKey, nil
	}

	return nil, fmt.Errorf("unsupported PEM block type: %s", block.Type)
}
