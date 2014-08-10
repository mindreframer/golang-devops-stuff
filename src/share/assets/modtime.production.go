// +build production

package assets
import (
	"time"
	"sync"
)

var uncompressedassets struct {
	cache map[string][]byte
	mutex sync.Mutex
}

func Uncompressedasset(name string) ([]byte, error) {
	uncompressedassets.mutex.Lock()
	defer uncompressedassets.mutex.Unlock()
	if text, ok := uncompressedassets.cache[name]; ok {
		return text, nil
	}
	text, err := Asset(name)
	if err != nil {
		uncompressedassets.cache[name] = text
	}
	return text, err
}

var STARTIME = time.Now()
func ModTime(string, string) (time.Time, error) {
	return STARTIME, nil
}
