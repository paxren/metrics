package handler

import (
	"net"
	"net/http"
)

// TrustedSubnetMiddleware проверяет IP-адрес клиента на принадлежность к доверенной подсети
type TrustedSubnetMiddleware struct {
	trustedSubnet *net.IPNet
}

// NewTrustedSubnetMiddleware создаёт новый middleware для проверки доверенной подсети
//
// Параметры:
//   - cidr: строка CIDR (например, "192.168.1.0/24")
//
// Возвращает:
//   - *TrustedSubnetMiddleware: указатель на созданный middleware
//   - error: ошибка при парсинге CIDR
func NewTrustedSubnetMiddleware(cidr string) (*TrustedSubnetMiddleware, error) {
	if cidr == "" {
		return &TrustedSubnetMiddleware{trustedSubnet: nil}, nil
	}

	_, ipNet, err := net.ParseCIDR(cidr)
	if err != nil {
		return nil, err
	}

	return &TrustedSubnetMiddleware{trustedSubnet: ipNet}, nil
}

// Check проверяет IP-адрес из заголовка X-Real-IP
//
// Если trustedSubnet равен nil - пропускает запрос без проверки.
// Если IP не в доверенной подсети - возвращает 403 Forbidden.
//
// Параметры:
//   - next: следующий хендлер в цепочке
//
// Возвращает:
//   - http.HandlerFunc: middleware функция
func (tm *TrustedSubnetMiddleware) Check(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Если доверенная подсеть не задана, пропускаем запрос
		if tm.trustedSubnet == nil {
			next(w, r)
			return
		}

		// Получаем IP из заголовка X-Real-IP
		realIP := r.Header.Get("X-Real-IP")
		if realIP == "" {
			http.Error(w, "X-Real-IP header is required", http.StatusForbidden)
			return
		}

		// Парсим IP-адрес
		ip := net.ParseIP(realIP)
		if ip == nil {
			http.Error(w, "Invalid IP address in X-Real-IP header", http.StatusBadRequest)
			return
		}

		// Проверяем, что IP входит в доверенную подсеть
		if !tm.trustedSubnet.Contains(ip) {
			http.Error(w, "IP address is not in trusted subnet", http.StatusForbidden)
			return
		}

		// IP в доверенной подсети, передаём управление следующему хендлеру
		next(w, r)
	}
}

// ParseIP парсит IP-адрес из строки
//
// Параметры:
//   - ipStr: строковое представление IP-адреса
//
// Возвращает:
//   - net.IP: распарсенный IP-адрес или nil при ошибке
func (tm *TrustedSubnetMiddleware) ParseIP(ipStr string) net.IP {
	return net.ParseIP(ipStr)
}

// Contains проверяет, принадлежит ли IP-адрес к доверенной подсети
//
// Если trustedSubnet равен nil - возвращает true (пропускает все IP).
//
// Параметры:
//   - ip: IP-адрес для проверки
//
// Возвращает:
//   - bool: true, если IP в доверенной подсети или подсеть не задана
func (tm *TrustedSubnetMiddleware) Contains(ip net.IP) bool {
	if tm.trustedSubnet == nil {
		return true
	}
	return tm.trustedSubnet.Contains(ip)
}
