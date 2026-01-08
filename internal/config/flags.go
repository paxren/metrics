package config

import (
	"net"
	"strconv"
	"strings"
)

// HostAddressParseError представляет ошибку парсинга адреса хоста.
//
// Возвращается при неверном формате адреса хоста.
type HostAddressParseError struct {
	message string
}

// Error возвращает текстовое представление ошибки.
//
// Реализует интерфейс error.
//
// Возвращает:
//   - string: сообщение об ошибке
func (e HostAddressParseError) Error() string {
	return e.message
}

// HostAddress представляет сетевой адрес в формате хост:порт.
//
// Поддерживает как IP-адреса, так и строку "localhost".
// Реализует интерфейс flag.Value для использования с флагами командной строки.
type HostAddress struct {
	Host string
	Port int
}

// NewHostAddress создаёт новый адрес хоста со значениями по умолчанию.
//
// Возвращает адрес с хостом "localhost" и портом 8080.
//
// Возвращает:
//   - *HostAddress: указатель на созданный адрес
func NewHostAddress() *HostAddress {
	return &HostAddress{
		Host: "localhost",
		Port: 8080,
	}
}

// String возвращает строковое представление адреса в формате хост:порт.
//
// Реализует метод интерфейса flag.Value.
//
// Возвращает:
//   - string: адрес в формате хост:порт
func (a HostAddress) String() string {
	return a.Host + ":" + strconv.Itoa(a.Port)
}

// Set устанавливает адрес из строки в формате хост:порт.
//
// Парсит строку и проверяет корректность хоста и порта.
// Для хоста допускается либо "localhost", либо валидный IP-адрес.
//
// Параметры:
//   - s: строка с адресом в формате хост:порт
//
// Возвращает:
//   - error: ошибка при неверном формате адреса
func (a *HostAddress) Set(s string) error {
	hp := strings.Split(s, ":")
	if len(hp) != 2 {
		return HostAddressParseError{message: "need address in a form host:port"}
	}
	port, err := strconv.Atoi(hp[1])
	if err != nil {
		return HostAddressParseError{message: err.Error()}
	}

	if hp[0] != "localhost" {
		ip := net.ParseIP(hp[0])
		if ip == nil {
			return HostAddressParseError{message: "need host in ip form or localhost string"}
		}
	}
	a.Host = hp[0]
	a.Port = port
	return nil
}
