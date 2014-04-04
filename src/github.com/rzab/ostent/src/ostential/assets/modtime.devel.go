// +build !production

package assets
import (
	"time"
)

func ModTime() time.Time {
	return time.Now()
}
