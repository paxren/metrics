package handler

import (
	"bytes"
	"fmt"
	"net/http"

	"github.com/paxren/metrics/internal/hash"
)

type (
	// берём структуру для хранения сведений об ответе
	responseHashData struct {
		hashKeyBytes []byte
		hash         string
	}

	// добавляем реализацию http.ResponseWriter
	hashResponseWriter struct {
		http.ResponseWriter // встраиваем оригинальный http.ResponseWriter
		responseHD          *responseHashData
	}
)

// compressWriter реализует интерфейс http.ResponseWriter и позволяет прозрачно для сервера
// сжимать передаваемые данные и выставлять правильные HTTP-заголовки
type hasher struct {
	hashKey      string
	hashKeyBytes []byte
}

func NewHasher(key string) *hasher {
	var hashKeyBytes []byte = nil
	if key != "" {
		hashKeyBytes = []byte(key)
	}

	return &hasher{
		hashKey:      key,
		hashKeyBytes: hashKeyBytes,
	}
}

func (hs hasher) HashMiddleware(h http.HandlerFunc) http.HandlerFunc {
	return func(res http.ResponseWriter, req *http.Request) {
		// по умолчанию устанавливаем оригинальный http.ResponseWriter как тот,
		// который будем передавать следующей функции

		// проверяем, что клиент умеет получать от сервера сжатые данные в формате gzip
		hashString := req.Header.Get("HashSHA256")
		if hashString != "" {
			var buf bytes.Buffer

			_, err := buf.ReadFrom(req.Body)

			if err != nil {
				http.Error(res, err.Error(), http.StatusBadRequest)
				return
			}
			src := buf.Bytes()
			hash, _ := hash.MakeHash(&hs.hashKeyBytes, &src)

			fmt.Printf("hash in header = %s, calculate hash body = %s \n", hashString, hash)

			if hashString != hash {
				fmt.Println("returning error")
				http.Error(res, "не совпал хеш", http.StatusBadRequest)
				return
			}
		}

		responseHD := &responseHashData{
			hashKeyBytes: hs.hashKeyBytes,
			hash:         "",
		}
		hashRes := &hashResponseWriter{
			ResponseWriter: res, // встраиваем оригинальный http.ResponseWriter
			responseHD:     responseHD,
		}
		fmt.Println("before serve hash")
		// передаём управление хендлеру
		h.ServeHTTP(hashRes, req)
		fmt.Println("after serve hash")

	}
}

func (hr *hashResponseWriter) Write(b []byte) (int, error) {
	// записываем ответ, используя оригинальный http.ResponseWriter

	fmt.Println("перед проверкой необходимости хеширования")
	if hr.responseHD.hashKeyBytes != nil {
		fmt.Println("будем хешировать")
		src := make([]byte, len(b))
		copy(src, b)
		hash, err := hash.MakeHash(&hr.responseHD.hashKeyBytes, &src)
		if err == nil {
			fmt.Println(hash)
			hr.Header().Set(`HashSHA256`, hash)
			fmt.Printf("hash = %s\n", hash)
		}
	}

	size, err := hr.ResponseWriter.Write(b)
	return size, err
}

// func (r *hashResponseWriter) WriteHeader(statusCode int) {
// 	// записываем код статуса, используя оригинальный http.ResponseWriter
// 	r.ResponseWriter.WriteHeader(statusCode)
// 	r.responseData.status = statusCode // захватываем код статуса
// }

// func (r *hashResponseWriter) Header() http.Header {

// 	r.responseData.headers = r.ResponseWriter.Header() // захватываем заголовки ответа

// 	return r.ResponseWriter.Header()
// }
