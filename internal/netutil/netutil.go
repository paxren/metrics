package netutil

import (
	"fmt"
	"net"
)

// GetLocalIP возвращает локальный IP-адрес машины
//
// Перебирает сетевые интерфейсы и возвращает первый непустой IP-адрес,
// исключая loopback интерфейсы.
//
// Возвращает:
//   - string: строковое представление IP-адреса
//   - error: ошибка при получении IP
func GetLocalIP() (string, error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return "", err
	}

	for _, iface := range interfaces {
		// Пропускаем отключенные интерфейсы
		if iface.Flags&net.FlagUp == 0 {
			continue
		}

		// Пропускаем loopback интерфейсы
		if iface.Flags&net.FlagLoopback != 0 {
			continue
		}

		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}

			// Пропускаем nil IP
			if ip == nil || ip.IsLoopback() {
				continue
			}

			// Возвращаем первый найденный IPv4 адрес
			ip = ip.To4()
			if ip != nil {
				return ip.String(), nil
			}
		}
	}

	return "", fmt.Errorf("no valid IP address found")
}
