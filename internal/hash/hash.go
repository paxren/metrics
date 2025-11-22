package hash

import (
	"crypto/hmac"
	"crypto/sha256"
	"fmt"
)

func MakeHash(key *[]byte, src *[]byte) (string, error) {

	fmt.Printf("original: %s\n", src)
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

	fmt.Printf("dest %x \n", dst)
	return fmt.Sprintf("%x", dst), nil
}
