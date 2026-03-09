package handler

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/paxren/metrics/internal/crypto"
)

// generateTestKeys генерирует тестовую пару ключей и сохраняет их во временные файлы.
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

func TestCryptoMiddleware_DecryptMiddleware(t *testing.T) {
	publicKeyPath, privateKeyPath := generateTestKeys(t)

	encryptor, err := crypto.NewEncryptor(publicKeyPath)
	if err != nil {
		t.Fatalf("Failed to create encryptor: %v", err)
	}

	decryptor, err := crypto.NewDecryptor(privateKeyPath)
	if err != nil {
		t.Fatalf("Failed to create decryptor: %v", err)
	}

	middleware := NewCryptoMiddleware(decryptor)

	// Создаём тестовый хендлер
	nextCalled := false
	var receivedBody []byte
	next := func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
		body, _ := io.ReadAll(r.Body)
		receivedBody = body
		w.WriteHeader(http.StatusOK)
	}

	// Тестируем дешифрование
	testData := []byte("test message")
	encryptedData, err := encryptor.Encrypt(testData)
	if err != nil {
		t.Fatalf("Failed to encrypt test data: %v", err)
	}

	req := httptest.NewRequest("POST", "/test", bytes.NewReader(encryptedData))
	w := httptest.NewRecorder()

	middleware.DecryptMiddleware(next).ServeHTTP(w, req)

	if !nextCalled {
		t.Error("Next handler was not called")
	}

	if !bytes.Equal(testData, receivedBody) {
		t.Errorf("Received body = %v, want %v", receivedBody, testData)
	}

	if w.Code != http.StatusOK {
		t.Errorf("Status code = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestCryptoMiddleware_NoDecryptor(t *testing.T) {
	middleware := NewCryptoMiddleware(nil)

	// Создаём тестовый хендлер
	nextCalled := false
	var receivedBody []byte
	next := func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
		body, _ := io.ReadAll(r.Body)
		receivedBody = body
		w.WriteHeader(http.StatusOK)
	}

	// Тестируем без дешифратора
	testData := []byte("test message")
	req := httptest.NewRequest("POST", "/test", bytes.NewReader(testData))
	w := httptest.NewRecorder()

	middleware.DecryptMiddleware(next).ServeHTTP(w, req)

	if !nextCalled {
		t.Error("Next handler was not called")
	}

	if !bytes.Equal(testData, receivedBody) {
		t.Errorf("Received body = %v, want %v", receivedBody, testData)
	}

	if w.Code != http.StatusOK {
		t.Errorf("Status code = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestCryptoMiddleware_DecryptionError(t *testing.T) {
	_, privateKeyPath := generateTestKeys(t)

	decryptor, err := crypto.NewDecryptor(privateKeyPath)
	if err != nil {
		t.Fatalf("Failed to create decryptor: %v", err)
	}

	middleware := NewCryptoMiddleware(decryptor)

	// Создаём тестовый хендлер
	nextCalled := false
	next := func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
		w.WriteHeader(http.StatusOK)
	}

	// Тестируем с невалидными зашифрованными данными
	invalidData := []byte("invalid encrypted data")
	req := httptest.NewRequest("POST", "/test", bytes.NewReader(invalidData))
	w := httptest.NewRecorder()

	middleware.DecryptMiddleware(next).ServeHTTP(w, req)

	if nextCalled {
		t.Error("Next handler should not be called on decryption error")
	}

	if w.Code != http.StatusBadRequest {
		t.Errorf("Status code = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestCryptoMiddleware_EmptyBody(t *testing.T) {
	publicKeyPath, privateKeyPath := generateTestKeys(t)

	encryptor, err := crypto.NewEncryptor(publicKeyPath)
	if err != nil {
		t.Fatalf("Failed to create encryptor: %v", err)
	}

	decryptor, err := crypto.NewDecryptor(privateKeyPath)
	if err != nil {
		t.Fatalf("Failed to create decryptor: %v", err)
	}

	middleware := NewCryptoMiddleware(decryptor)

	// Создаём тестовый хендлер
	nextCalled := false
	var receivedBody []byte
	next := func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
		body, _ := io.ReadAll(r.Body)
		receivedBody = body
		w.WriteHeader(http.StatusOK)
	}

	// Тестируем с пустым телом
	testData := []byte("")
	encryptedData, err := encryptor.Encrypt(testData)
	if err != nil {
		t.Fatalf("Failed to encrypt test data: %v", err)
	}

	req := httptest.NewRequest("POST", "/test", bytes.NewReader(encryptedData))
	w := httptest.NewRecorder()

	middleware.DecryptMiddleware(next).ServeHTTP(w, req)

	if !nextCalled {
		t.Error("Next handler was not called")
	}

	if !bytes.Equal(testData, receivedBody) {
		t.Errorf("Received body = %v, want %v", receivedBody, testData)
	}
}

func TestCryptoMiddleware_LargeData(t *testing.T) {
	publicKeyPath, privateKeyPath := generateTestKeys(t)

	encryptor, err := crypto.NewEncryptor(publicKeyPath)
	if err != nil {
		t.Fatalf("Failed to create encryptor: %v", err)
	}

	decryptor, err := crypto.NewDecryptor(privateKeyPath)
	if err != nil {
		t.Fatalf("Failed to create decryptor: %v", err)
	}

	middleware := NewCryptoMiddleware(decryptor)

	// Создаём тестовый хендлер
	nextCalled := false
	var receivedBody []byte
	next := func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
		body, _ := io.ReadAll(r.Body)
		receivedBody = body
		w.WriteHeader(http.StatusOK)
	}

	// Создаём данные максимального размера для 2048-битного ключа (245 байт)
	testData := make([]byte, 245)
	for i := range testData {
		testData[i] = byte(i % 256)
	}

	encryptedData, err := encryptor.Encrypt(testData)
	if err != nil {
		t.Fatalf("Failed to encrypt test data: %v", err)
	}

	req := httptest.NewRequest("POST", "/test", bytes.NewReader(encryptedData))
	w := httptest.NewRecorder()

	middleware.DecryptMiddleware(next).ServeHTTP(w, req)

	if !nextCalled {
		t.Error("Next handler was not called")
	}

	if !bytes.Equal(testData, receivedBody) {
		t.Error("Received body does not match test data")
	}
}

func TestNewCryptoMiddleware(t *testing.T) {
	_, privateKeyPath := generateTestKeys(t)

	decryptor, err := crypto.NewDecryptor(privateKeyPath)
	if err != nil {
		t.Fatalf("Failed to create decryptor: %v", err)
	}

	middleware := NewCryptoMiddleware(decryptor)

	if middleware == nil {
		t.Fatal("NewCryptoMiddleware() returned nil")
	}

	if middleware.decryptor != decryptor {
		t.Error("NewCryptoMiddleware() did not set decryptor correctly")
	}
}

func TestNewCryptoMiddleware_Nil(t *testing.T) {
	middleware := NewCryptoMiddleware(nil)

	if middleware == nil {
		t.Fatal("NewCryptoMiddleware() returned nil")
	}

	if middleware.decryptor != nil {
		t.Error("NewCryptoMiddleware() should accept nil decryptor")
	}
}
