package crypto

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestHybridEncryptor(t *testing.T) {
	// Создаем временные файлы для тестовых ключей
	tempDir := t.TempDir()
	publicKeyPath := filepath.Join(tempDir, "public.pem")
	privateKeyPath := filepath.Join(tempDir, "private.pem")

	// Генерируем тестовые ключи
	if err := generateHybridTestKeys(publicKeyPath, privateKeyPath); err != nil {
		t.Fatalf("Failed to generate test keys: %v", err)
	}

	tests := []struct {
		name      string
		data      []byte
		wantError bool
	}{
		{
			name:      "Small data",
			data:      []byte("Hello, World!"),
			wantError: false,
		},
		{
			name:      "Medium data",
			data:      make([]byte, 1024),
			wantError: false,
		},
		{
			name:      "Large data",
			data:      make([]byte, 10240),
			wantError: false,
		},
		{
			name:      "Empty data",
			data:      []byte(""),
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Создаем шифратор
			encryptor, err := NewHybridEncryptor(publicKeyPath)
			if err != nil {
				t.Fatalf("Failed to create encryptor: %v", err)
			}

			// Шифруем данные
			encrypted, err := encryptor.Encrypt(tt.data)
			if (err != nil) != tt.wantError {
				t.Errorf("Encrypt() error = %v, wantError %v", err, tt.wantError)
				return
			}

			if !tt.wantError {
				// Проверяем, что зашифрованные данные не пустые
				if len(encrypted) == 0 {
					t.Error("Encrypted data is empty")
				}

				// Проверяем, что зашифрованные данные отличаются от исходных
				if len(encrypted) == len(tt.data) {
					same := true
					for i := range encrypted {
						if encrypted[i] != tt.data[i] {
							same = false
							break
						}
					}
					if same {
						t.Error("Encrypted data is the same as original")
					}
				}
			}
		})
	}
}

func TestHybridDecryptor(t *testing.T) {
	// Создаем временные файлы для тестовых ключей
	tempDir := t.TempDir()
	publicKeyPath := filepath.Join(tempDir, "public.pem")
	privateKeyPath := filepath.Join(tempDir, "private.pem")

	// Генерируем тестовые ключи
	if err := generateHybridTestKeys(publicKeyPath, privateKeyPath); err != nil {
		t.Fatalf("Failed to generate test keys: %v", err)
	}

	tests := []struct {
		name      string
		data      []byte
		wantError bool
	}{
		{
			name:      "Small data",
			data:      []byte("Hello, World!"),
			wantError: false,
		},
		{
			name:      "Medium data",
			data:      make([]byte, 1024),
			wantError: false,
		},
		{
			name:      "Large data",
			data:      make([]byte, 10240),
			wantError: false,
		},
		{
			name:      "Empty data",
			data:      []byte(""),
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Создаем шифратор и дешифратор
			encryptor, err := NewHybridEncryptor(publicKeyPath)
			if err != nil {
				t.Fatalf("Failed to create encryptor: %v", err)
			}

			decryptor, err := NewHybridDecryptor(privateKeyPath)
			if err != nil {
				t.Fatalf("Failed to create decryptor: %v", err)
			}

			// Шифруем данные
			encrypted, err := encryptor.Encrypt(tt.data)
			if err != nil {
				t.Fatalf("Failed to encrypt data: %v", err)
			}

			// Дешифруем данные
			decrypted, err := decryptor.Decrypt(encrypted)
			if (err != nil) != tt.wantError {
				t.Errorf("Decrypt() error = %v, wantError %v", err, tt.wantError)
				return
			}

			if !tt.wantError {
				// Проверяем, что дешифрованные данные совпадают с исходными
				if len(decrypted) != len(tt.data) {
					t.Errorf("Decrypted data length = %d, want %d", len(decrypted), len(tt.data))
				}

				for i := range decrypted {
					if decrypted[i] != tt.data[i] {
						t.Errorf("Decrypted data differs at position %d", i)
						break
					}
				}
			}
		})
	}
}

func TestHybridEncryptDecrypt(t *testing.T) {
	// Создаем временные файлы для тестовых ключей
	tempDir := t.TempDir()
	publicKeyPath := filepath.Join(tempDir, "public.pem")
	privateKeyPath := filepath.Join(tempDir, "private.pem")

	// Генерируем тестовые ключи
	if err := generateHybridTestKeys(publicKeyPath, privateKeyPath); err != nil {
		t.Fatalf("Failed to generate test keys: %v", err)
	}

	// Создаем шифратор и дешифратор
	encryptor, err := NewHybridEncryptor(publicKeyPath)
	if err != nil {
		t.Fatalf("Failed to create encryptor: %v", err)
	}

	decryptor, err := NewHybridDecryptor(privateKeyPath)
	if err != nil {
		t.Fatalf("Failed to create decryptor: %v", err)
	}

	// Тестируем различные размеры данных
	testData := [][]byte{
		[]byte(""),
		[]byte("Hello"),
		make([]byte, 100),
		make([]byte, 1000),
		make([]byte, 10000),
	}

	for i, data := range testData {
		t.Run(fmt.Sprintf("TestData_%d", i), func(t *testing.T) {
			// Шифруем
			encrypted, err := encryptor.Encrypt(data)
			if err != nil {
				t.Fatalf("Failed to encrypt: %v", err)
			}

			// Дешифруем
			decrypted, err := decryptor.Decrypt(encrypted)
			if err != nil {
				t.Fatalf("Failed to decrypt: %v", err)
			}

			// Сравниваем
			if string(decrypted) != string(data) {
				t.Errorf("Decrypted data doesn't match original")
			}
		})
	}
}

func TestHybridEncryptorInvalidPath(t *testing.T) {
	_, err := NewHybridEncryptor("/nonexistent/path")
	if err == nil {
		t.Error("Expected error for invalid path")
	}
}

func TestHybridDecryptorInvalidPath(t *testing.T) {
	_, err := NewHybridDecryptor("/nonexistent/path")
	if err == nil {
		t.Error("Expected error for invalid path")
	}
}

func TestHybridDecryptorAdapter(t *testing.T) {
	// Создаем временные файлы для тестовых ключей
	tempDir := t.TempDir()
	publicKeyPath := filepath.Join(tempDir, "public.pem")
	privateKeyPath := filepath.Join(tempDir, "private.pem")

	// Генерируем тестовые ключи
	if err := generateHybridTestKeys(publicKeyPath, privateKeyPath); err != nil {
		t.Fatalf("Failed to generate test keys: %v", err)
	}

	// Создаем гибридный дешифратор
	hybridDecryptor, err := NewHybridDecryptor(privateKeyPath)
	if err != nil {
		t.Fatalf("Failed to create hybrid decryptor: %v", err)
	}

	// Создаем адаптер
	adapter := NewHybridDecryptorAdapter(hybridDecryptor)

	// Создаем шифратор
	encryptor, err := NewHybridEncryptor(publicKeyPath)
	if err != nil {
		t.Fatalf("Failed to create encryptor: %v", err)
	}

	// Тестируем шифрование/дешифрование через адаптер
	testData := []byte("Test data for adapter")

	encrypted, err := encryptor.Encrypt(testData)
	if err != nil {
		t.Fatalf("Failed to encrypt: %v", err)
	}

	decrypted, err := adapter.Decrypt(encrypted)
	if err != nil {
		t.Fatalf("Failed to decrypt via adapter: %v", err)
	}

	if string(decrypted) != string(testData) {
		t.Error("Decrypted data doesn't match original")
	}
}

// generateTestKeys генерирует тестовые RSA ключи
func generateHybridTestKeys(publicKeyPath, privateKeyPath string) error {
	// Создаем шаблон сертификата
	cert := &x509.Certificate{
		SerialNumber: big.NewInt(1658),
		Subject: pkix.Name{
			Organization: []string{"Test"},
			Country:      []string{"RU"},
		},
		IPAddresses:  []net.IP{net.IPv4(127, 0, 0, 1), net.IPv6loopback},
		NotBefore:    time.Now(),
		NotAfter:     time.Now().AddDate(10, 0, 0),
		SubjectKeyId: []byte{1, 2, 3, 4, 6},
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		KeyUsage:     x509.KeyUsageDigitalSignature,
	}

	// Генерируем приватный RSA-ключ
	privateKey, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return err
	}

	// Создаем сертификат x.509
	certBytes, err := x509.CreateCertificate(rand.Reader, cert, cert, &privateKey.PublicKey, privateKey)
	if err != nil {
		return err
	}

	// Кодируем сертификат и ключ в формате PEM
	var certPEM, privateKeyPEM bytes.Buffer

	err = pem.Encode(&certPEM, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certBytes,
	})
	if err != nil {
		return err
	}

	err = pem.Encode(&privateKeyPEM, &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	})
	if err != nil {
		return err
	}

	// Сохраняем в файлы
	if err := os.WriteFile(publicKeyPath, certPEM.Bytes(), 0644); err != nil {
		return err
	}

	return os.WriteFile(privateKeyPath, privateKeyPEM.Bytes(), 0644)
}
