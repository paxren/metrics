package audit

import (
	"encoding/json"
	"os"
	"sync"

	"github.com/paxren/metrics/internal/models"
)

type FileObserver struct {
	filePath string
	mutex    sync.Mutex
}

func NewFileObserver(filePath string) *FileObserver {
	return &FileObserver{
		filePath: filePath,
	}
}

func (f *FileObserver) Notify(event *models.AuditEvent) error {
	f.mutex.Lock()
	defer f.mutex.Unlock()

	file, err := os.OpenFile(f.filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	data, err := json.Marshal(event)
	if err != nil {
		return err
	}

	_, err = file.Write(append(data, '\n'))
	return err
}
