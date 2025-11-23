package handler

import (
	"bytes"
	"fmt"
	"io"
	"net/http"

	"github.com/paxren/metrics/internal/hash"
)

type (
	// берём структуру для хранения сведений об ответе
	responseHashData struct {
		hashKeyBytes []byte
		hash         string
		err          bool
	}

	// добавляем реализацию http.ResponseWriter
	hashResponseWriter struct {
		http.ResponseWriter // встраиваем оригинальный http.ResponseWriter
		responseHD          *responseHashData
		responseBody        bytes.Buffer
		hashComputed        bool
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

func (hs *hasher) HashMiddleware(h http.HandlerFunc) http.HandlerFunc {
	logFn := func(res http.ResponseWriter, req *http.Request) {
		// по умолчанию устанавливаем оригинальный http.ResponseWriter как тот,
		// который будем передавать следующей функции
		var errHash bool = false
		// проверяем, что клиент умеет получать от сервера сжатые данные в формате gzip

		//TUT OHIBKA

		hashString := req.Header.Get("HashSHA256")
		if hashString != "" {
			var buf bytes.Buffer

			_, err := buf.ReadFrom(req.Body)
			req.Body = io.NopCloser(bytes.NewReader(buf.Bytes()))
			// bd := make([]byte, 0, 1000)
			// n, err := req.Body.Read(bd)
			//fmt.Printf("body read = %v size = %d, err= %s\n", bd, n, err)

			if err != nil {
				fmt.Println("ERROR-ERROR")
				http.Error(res, err.Error(), http.StatusBadRequest)
				return
			}
			//req.Body.Close()
			src := buf.Bytes()
			hash, _ := hash.MakeHash(&hs.hashKeyBytes, &src)

			fmt.Printf("hash in header = %s, calculate hash body = %s \n", hashString, hash)

			if hashString != hash {
				fmt.Println("will returning error")
				errHash = true
				// http.Error(res, "не совпал хеш", http.StatusBadRequest)
				// return
			}
		}

		fmt.Printf("errHash = %v \n", errHash)
		responseHD := &responseHashData{
			hashKeyBytes: hs.hashKeyBytes,
			hash:         "",
			err:          errHash,
		}
		hashRes := &hashResponseWriter{
			ResponseWriter: res, // встраиваем оригинальный http.ResponseWriter
			responseHD:     responseHD,
		}
		fmt.Println("===hasher before serve hash ")
		// передаём управление хендлеру
		h.ServeHTTP(hashRes, req)
		fmt.Println("===hasher after serve hash")
		hashRes.computeHash()
	}

	return http.HandlerFunc(logFn)
}

func (hr *hashResponseWriter) Write(b []byte) (int, error) {
	// записываем ответ, используя оригинальный http.ResponseWriter

	hr.responseBody.Write(b)

	return hr.ResponseWriter.Write(b)
}

func (hr *hashResponseWriter) WriteHeader(statusCode int) {
	// записываем код статуса, используя оригинальный http.ResponseWriter
	if hr.responseHD.err {
		hr.ResponseWriter.WriteHeader(http.StatusBadRequest)
	} else {
		hr.ResponseWriter.WriteHeader(statusCode)
	}

	//hr.responseData.status = statusCode // захватываем код статуса
}

func (hr *hashResponseWriter) computeHash() {
	if !hr.hashComputed && hr.responseHD.hashKeyBytes != nil && hr.responseBody.Len() > 0 {
		rb := hr.responseBody.Bytes()
		hash, err := hash.MakeHash(&hr.responseHD.hashKeyBytes, &rb)
		if err == nil {
			hr.Header().Set("HashSHA256", hash)
			fmt.Printf("hash response = %s\n", hash)
		}
		hr.hashComputed = true
	}
}

// func (r *hashResponseWriter) Header() http.Header {

// 	r.responseData.headers = r.ResponseWriter.Header() // захватываем заголовки ответа

// 	return r.ResponseWriter.Header()
// }
