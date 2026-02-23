package crypto

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"os"
	"path/filepath"
	"testing"
)

// generateTestKeys генерирует тестовую пару ключей и сохраняет их во временные файлы.
//
// Возвращает пути к файлам с публичным и приватным ключами.
func generateTestKeys(t *testing.T) (string, string) {
	t.Helper()

	// Генерируем приватный ключ
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("Failed to generate private key: %v", err)
	}

	// Создаём временный каталог для ключей
	tmpDir := t.TempDir()

	// Сохраняем приватный ключ
	privateKeyBytes := x509.MarshalPKCS1PrivateKey(privateKey)
	privateKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: privateKeyBytes,
	})
	privateKeyPath := filepath.Join(tmpDir, "private.pem")
	if err := os.WriteFile(privateKeyPath, privateKeyPEM, 0600); err != nil {
		t.Fatalf("Failed to write private key: %v", err)
	}

	// Сохраняем публичный ключ
	publicKeyBytes, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	if err != nil {
		t.Fatalf("Failed to marshal public key: %v", err)
	}
	publicKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: publicKeyBytes,
	})
	publicKeyPath := filepath.Join(tmpDir, "public.pem")
	if err := os.WriteFile(publicKeyPath, publicKeyPEM, 0644); err != nil {
		t.Fatalf("Failed to write public key: %v", err)
	}

	return publicKeyPath, privateKeyPath
}

func TestNewEncryptor(t *testing.T) {
	publicKeyPath, _ := generateTestKeys(t)

	encryptor, err := NewEncryptor(publicKeyPath)
	if err != nil {
		t.Fatalf("NewEncryptor() error = %v", err)
	}

	if encryptor == nil {
		t.Fatal("NewEncryptor() returned nil")
	}

	if encryptor.publicKey == nil {
		t.Fatal("NewEncryptor() publicKey is nil")
	}
}

func TestNewEncryptorInvalidPath(t *testing.T) {
	_, err := NewEncryptor("/nonexistent/path/to/key.pem")
	if err == nil {
		t.Fatal("NewEncryptor() expected error for invalid path, got nil")
	}
}

func TestNewDecryptor(t *testing.T) {
	_, privateKeyPath := generateTestKeys(t)

	decryptor, err := NewDecryptor(privateKeyPath)
	if err != nil {
		t.Fatalf("NewDecryptor() error = %v", err)
	}

	if decryptor == nil {
		t.Fatal("NewDecryptor() returned nil")
	}

	if decryptor.privateKey == nil {
		t.Fatal("NewDecryptor() privateKey is nil")
	}
}

func TestNewDecryptorInvalidPath(t *testing.T) {
	_, err := NewDecryptor("/nonexistent/path/to/key.pem")
	if err == nil {
		t.Fatal("NewDecryptor() expected error for invalid path, got nil")
	}
}

func TestEncryptDecrypt(t *testing.T) {
	publicKeyPath, privateKeyPath := generateTestKeys(t)

	encryptor, err := NewEncryptor(publicKeyPath)
	if err != nil {
		t.Fatalf("NewEncryptor() error = %v", err)
	}

	decryptor, err := NewDecryptor(privateKeyPath)
	if err != nil {
		t.Fatalf("NewDecryptor() error = %v", err)
	}

	// Тестируем шифрование и дешифрование
	testData := []byte("secret message")

	encrypted, err := encryptor.Encrypt(testData)
	if err != nil {
		t.Fatalf("Encrypt() error = %v", err)
	}

	if len(encrypted) == 0 {
		t.Fatal("Encrypt() returned empty data")
	}

	decrypted, err := decryptor.Decrypt(encrypted)
	if err != nil {
		t.Fatalf("Decrypt() error = %v", err)
	}

	if !bytes.Equal(testData, decrypted) {
		t.Errorf("Decrypt() = %v, want %v", decrypted, testData)
	}
}

func TestEncryptDecryptLargeData(t *testing.T) {
	publicKeyPath, privateKeyPath := generateTestKeys(t)

	encryptor, err := NewEncryptor(publicKeyPath)
	if err != nil {
		t.Fatalf("NewEncryptor() error = %v", err)
	}

	decryptor, err := NewDecryptor(privateKeyPath)
	if err != nil {
		t.Fatalf("NewDecryptor() error = %v", err)
	}

	// Создаём данные максимального размера для 2048-битного ключа (245 байт)
	testData := make([]byte, 245)
	for i := range testData {
		testData[i] = byte(i % 256)
	}

	encrypted, err := encryptor.Encrypt(testData)
	if err != nil {
		t.Fatalf("Encrypt() error = %v", err)
	}

	decrypted, err := decryptor.Decrypt(encrypted)
	if err != nil {
		t.Fatalf("Decrypt() error = %v", err)
	}

	if !bytes.Equal(testData, decrypted) {
		t.Error("Decrypt() returned different data")
	}
}

func TestEncryptDecryptMultipleTimes(t *testing.T) {
	publicKeyPath, privateKeyPath := generateTestKeys(t)

	encryptor, err := NewEncryptor(publicKeyPath)
	if err != nil {
		t.Fatalf("NewEncryptor() error = %v", err)
	}

	decryptor, err := NewDecryptor(privateKeyPath)
	if err != nil {
		t.Fatalf("NewDecryptor() error = %v", err)
	}

	// Тестируем несколько шифрований и дешифрований
	testCases := []string{
		"test message 1",
		"another test message",
		"1234567890",
		"special chars: !@#$%^&*()",
	}

	for _, tc := range testCases {
		testData := []byte(tc)

		encrypted, err := encryptor.Encrypt(testData)
		if err != nil {
			t.Fatalf("Encrypt() error = %v", err)
		}

		decrypted, err := decryptor.Decrypt(encrypted)
		if err != nil {
			t.Fatalf("Decrypt() error = %v", err)
		}

		if !bytes.Equal(testData, decrypted) {
			t.Errorf("Decrypt() = %v, want %v", decrypted, testData)
		}
	}
}

func TestReadPublicKey(t *testing.T) {
	publicKeyPath, _ := generateTestKeys(t)

	publicKey, err := readPublicKey(publicKeyPath)
	if err != nil {
		t.Fatalf("readPublicKey() error = %v", err)
	}

	if publicKey == nil {
		t.Fatal("readPublicKey() returned nil")
	}

	if publicKey.Size() != 256 {
		t.Errorf("readPublicKey() key size = %d, want 256", publicKey.Size())
	}
}

func TestReadPrivateKey(t *testing.T) {
	_, privateKeyPath := generateTestKeys(t)

	privateKey, err := readPrivateKey(privateKeyPath)
	if err != nil {
		t.Fatalf("readPrivateKey() error = %v", err)
	}

	if privateKey == nil {
		t.Fatal("readPrivateKey() returned nil")
	}

	if privateKey.Size() != 256 {
		t.Errorf("readPrivateKey() key size = %d, want 256", privateKey.Size())
	}
}
