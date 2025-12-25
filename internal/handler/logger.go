package handler

import (
	"fmt"
	"net/http"
	"time"

	"go.uber.org/zap"
)

type (
	// берём структуру для хранения сведений об ответе
	responseData struct {
		status  int
		size    int
		headers http.Header
	}

	// добавляем реализацию http.ResponseWriter
	loggingResponseWriter struct {
		http.ResponseWriter // встраиваем оригинальный http.ResponseWriter
		responseData        *responseData
	}
)

type Logger struct {
	logger *zap.Logger
	sugar  *zap.SugaredLogger
}

func NewLogger(z *zap.Logger) *Logger {
	return &Logger{
		logger: z,
		sugar:  z.Sugar(),
	}
}

func (l Logger) WithLogging(h http.HandlerFunc) http.HandlerFunc {
	logFn := func(w http.ResponseWriter, r *http.Request) {
		// функция Now() возвращает текущее время
		start := time.Now()

		responseData := &responseData{
			status:  0,
			size:    0,
			headers: nil,
		}
		lw := loggingResponseWriter{
			ResponseWriter: w, // встраиваем оригинальный http.ResponseWriter
			responseData:   responseData,
		}

		// точка, где выполняется хендлер pingHandler
		fmt.Println("===logger before serve hash")
		h.ServeHTTP(&lw, r) // обслуживание оригинального запроса

		fmt.Println("===logger after serve hash")
		// Since возвращает разницу во времени между start
		// и моментом вызова Since. Таким образом можно посчитать
		// время выполнения запроса.
		duration := time.Since(start)

		// отправляем сведения о запросе в zap
		l.sugar.Infoln(
			"uri", r.RequestURI,
			"method", r.Method,
			"status", responseData.status, // получаем перехваченный код статуса ответа
			"duration", duration,
			"size", responseData.size, // получаем перехваченный размер ответа
			"requestHeaders", r.Header,
			"responceHeaders", responseData.headers,
		)

	}
	// возвращаем функционально расширенный хендлер
	return http.HandlerFunc(logFn)
}

func (r *loggingResponseWriter) Write(b []byte) (int, error) {
	// записываем ответ, используя оригинальный http.ResponseWriter
	size, err := r.ResponseWriter.Write(b)
	r.responseData.size += size // захватываем размер
	return size, err
}

func (r *loggingResponseWriter) WriteHeader(statusCode int) {
	// записываем код статуса, используя оригинальный http.ResponseWriter
	r.ResponseWriter.WriteHeader(statusCode)
	r.responseData.status = statusCode // захватываем код статуса
}

func (r *loggingResponseWriter) Header() http.Header {

	r.responseData.headers = r.ResponseWriter.Header() // захватываем заголовки ответа

	return r.ResponseWriter.Header()
}
