package crypto

// HybridDecryptorAdapter адаптирует HybridDecryptor к интерфейсу Decryptor.
//
// Это позволяет использовать гибридное дешифрование с существующим middleware.
type HybridDecryptorAdapter struct {
	hybridDecryptor *HybridDecryptor
}

// NewHybridDecryptorAdapter создаёт новый адаптер для HybridDecryptor.
//
// Параметры:
//   - hybridDecryptor: гибридный дешифратор
//
// Возвращает:
//   - *HybridDecryptorAdapter: указатель на созданный адаптер
func NewHybridDecryptorAdapter(hybridDecryptor *HybridDecryptor) *HybridDecryptorAdapter {
	return &HybridDecryptorAdapter{
		hybridDecryptor: hybridDecryptor,
	}
}

// Decrypt дешифрует данные с использованием гибридной схемы.
//
// Реализует интерфейс Decryptor для совместимости с middleware.
//
// Параметры:
//   - data: зашифрованные данные
//
// Возвращает:
//   - []byte: дешифрованные данные
//   - error: ошибка при дешифровании
func (a *HybridDecryptorAdapter) Decrypt(data []byte) ([]byte, error) {
	return a.hybridDecryptor.Decrypt(data)
}
