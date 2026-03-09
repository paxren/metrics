package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"fmt"
)

// HybridEncryptor предоставляет методы для гибридного шифрования данных.
//
// Использует AES для шифрования данных и RSA для шифрования AES ключа.
type HybridEncryptor struct {
	publicKey *rsa.PublicKey
}

// HybridDecryptor предоставляет методы для гибридного дешифрования данных.
//
// Использует RSA для дешифрования AES ключа и AES для дешифрования данных.
type HybridDecryptor struct {
	privateKey *rsa.PrivateKey
}

// EncryptedData представляет структуру зашифрованных данных.
type EncryptedData struct {
	EncryptedKey []byte `json:"encrypted_key"`
	IV           []byte `json:"iv"`
	Data         []byte `json:"data"`
}

// NewHybridEncryptor создаёт новый HybridEncryptor из файла с публичным ключом.
//
// Параметры:
//   - publicKeyPath: путь к файлу с публичным ключом в PEM формате
//
// Возвращает:
//   - *HybridEncryptor: указатель на созданный шифратор
//   - error: ошибка при чтении или парсинге ключа
func NewHybridEncryptor(publicKeyPath string) (*HybridEncryptor, error) {
	publicKey, err := readPublicKey(publicKeyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read public key: %w", err)
	}

	return &HybridEncryptor{
		publicKey: publicKey,
	}, nil
}

// NewHybridDecryptor создаёт новый HybridDecryptor из файла с приватным ключом.
//
// Параметры:
//   - privateKeyPath: путь к файлу с приватным ключом в PEM формате
//
// Возвращает:
//   - *HybridDecryptor: указатель на созданный дешифратор
//   - error: ошибка при чтении или парсинге ключа
func NewHybridDecryptor(privateKeyPath string) (*HybridDecryptor, error) {
	privateKey, err := readPrivateKey(privateKeyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read private key: %w", err)
	}

	return &HybridDecryptor{
		privateKey: privateKey,
	}, nil
}

// Encrypt шифрует данные с использованием гибридной схемы.
//
// Алгоритм:
// 1. Генерирует случайный AES ключ (256 бит) и IV (128 бит)
// 2. Шифрует данные AES в режиме GCM
// 3. Шифрует AES ключ RSA публичным ключом
// 4. Возвращает JSON с зашифрованным ключом, IV и данными
//
// Параметры:
//   - data: данные для шифрования
//
// Возвращает:
//   - []byte: зашифрованные данные в формате JSON
//   - error: ошибка при шифровании
func (e *HybridEncryptor) Encrypt(data []byte) ([]byte, error) {
	// Генерируем случайный AES ключ (256 бит)
	aesKey := make([]byte, 32)
	if _, err := rand.Read(aesKey); err != nil {
		return nil, fmt.Errorf("failed to generate AES key: %w", err)
	}

	// Создаем AES cipher
	block, err := aes.NewCipher(aesKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create AES cipher: %w", err)
	}

	// Создаем GCM mode
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	// Генерируем случайный IV (nonce)
	iv := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(iv); err != nil {
		return nil, fmt.Errorf("failed to generate IV: %w", err)
	}

	// Шифруем данные
	encryptedData := gcm.Seal(nil, iv, data, nil)

	// Шифруем AES ключ RSA
	encryptedKey, err := rsa.EncryptPKCS1v15(rand.Reader, e.publicKey, aesKey)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt AES key: %w", err)
	}

	// Создаем структуру для сериализации
	encData := EncryptedData{
		EncryptedKey: encryptedKey,
		IV:           iv,
		Data:         encryptedData,
	}

	// Сериализуем в JSON
	return json.Marshal(encData)
}

// Decrypt дешифрует данные с использованием гибридной схемы.
//
// Алгоритм:
// 1. Десериализует JSON
// 2. Дешифрует AES ключ RSA приватным ключом
// 3. Дешифрует данные AES в режиме GCM
//
// Параметры:
//   - data: зашифрованные данные в формате JSON
//
// Возвращает:
//   - []byte: дешифрованные данные
//   - error: ошибка при дешифровании
func (d *HybridDecryptor) Decrypt(data []byte) ([]byte, error) {
	// Десериализуем JSON
	var encData EncryptedData
	if err := json.Unmarshal(data, &encData); err != nil {
		return nil, fmt.Errorf("failed to unmarshal encrypted data: %w", err)
	}

	// Дешифруем AES ключ
	aesKey, err := rsa.DecryptPKCS1v15(rand.Reader, d.privateKey, encData.EncryptedKey)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt AES key: %w", err)
	}

	// Создаем AES cipher
	block, err := aes.NewCipher(aesKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create AES cipher: %w", err)
	}

	// Создаем GCM mode
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	// Проверяем размер IV
	if len(encData.IV) != gcm.NonceSize() {
		return nil, fmt.Errorf("invalid IV size")
	}

	// Дешифруем данные
	decryptedData, err := gcm.Open(nil, encData.IV, encData.Data, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt data: %w", err)
	}

	return decryptedData, nil
}
