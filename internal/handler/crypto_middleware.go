package handler

import (
	"bytes"
	"io"
	"net/http"

	"github.com/paxren/metrics/internal/crypto"
)

// CryptoMiddleware обёртка для дешифрования тела запроса.
//
// Если дешифратор не задан, middleware пропускает запрос без изменений.
type CryptoMiddleware struct {
	decryptor crypto.DecryptorInterface
}

// NewCryptoMiddleware создаёт новый middleware для дешифрования.
//
// Параметры:
//   - decryptor: дешифратор для расшифровки тела запроса
//
// Возвращает:
//   - *CryptoMiddleware: указатель на созданный middleware
func NewCryptoMiddleware(decryptor crypto.DecryptorInterface) *CryptoMiddleware {
	return &CryptoMiddleware{
		decryptor: decryptor,
	}
}

// DecryptMiddleware дешифрует тело запроса перед передачей хендлеру.
//
// Если дешифратор не задан, запрос передаётся без изменений.
// При ошибке дешифрования возвращается статус 400 Bad Request.
//
// Параметры:
//   - next: следующий хендлер в цепочке
//
// Возвращает:
//   - http.HandlerFunc: middleware функция
func (cm *CryptoMiddleware) DecryptMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Если дешифратор не задан, передаём запрос как есть
		if cm.decryptor == nil {
			next(w, r)
			return
		}

		// Читаем тело запроса
		bodyBytes, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "failed to read request body", http.StatusBadRequest)
			return
		}
		defer r.Body.Close()

		// Дешифруем тело запроса
		decryptedBody, err := cm.decryptor.Decrypt(bodyBytes)
		if err != nil {
			http.Error(w, "failed to decrypt request body", http.StatusBadRequest)
			return
		}

		// Заменяем тело запроса на дешифрованное
		r.Body = io.NopCloser(bytes.NewBuffer(decryptedBody))

		// Передаём управление следующему хендлеру
		next(w, r)
	}
}
