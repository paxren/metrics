package hash

import (
	"crypto/hmac"
	"crypto/sha256"
	"fmt"
)

// MakeHash создаёт HMAC-хеш для данных с использованием указанного ключа.
//
// Использует алгоритм HMAC с хеш-функцией SHA-256 для создания
// криптографически стойкого хеша. Применяется для проверки целостности
// данных и аутентификации сообщений.
//
// Параметры:
//   - key: указатель на массив байт ключа для HMAC
//   - src: указатель на массив байт данных для хеширования
//
// Возвращает:
//   - string: хеш в шестнадцатеричном формате
//   - error: ошибка при записи данных в HMAC
//
// Пример использования:
//
//	key := []byte("secret-key")
//	data := []byte("message to hash")
//	hash, err := MakeHash(&key, &data)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Printf("Hash: %s", hash)
func MakeHash(key *[]byte, src *[]byte) (string, error) {

	//fmt.Printf("original: %s\n", src)
	// подписываемое сообщение
	//src := []byte("Видишь гофера? Нет. И я нет. А он есть.")

	// создаём случайный ключ
	// key, err := generateRandom(16)
	// if err != nil {
	//     fmt.Printf("error: %v\n", err)
	//     return
	// }

	// подписываем алгоритмом HMAC, используя SHA-256
	h := hmac.New(sha256.New, *key)
	_, err := h.Write(*src)
	if err != nil {
		return "", err
	}
	dst := h.Sum(nil)

	//fmt.Printf("dest %x \n", dst)
	return fmt.Sprintf("%x", dst), nil
}
