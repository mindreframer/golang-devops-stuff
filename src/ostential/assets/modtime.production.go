// +build production

package assets
import (
	"time"
)

var STARTIME = time.Now()
func ModTime() time.Time {
	return STARTIME
}
