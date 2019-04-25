package action

import (
	"os"

	"github.com/Masterminds/glide/cache"
	"github.com/Masterminds/glide/msg"
)

// CacheClear clears the Glide cache
func CacheClear() {
	l := cache.Location()

	var err error
	for i := 0; i < 20; i++ {
		err = os.RemoveAll(l)
		if err != nil {
			msg.Warn("Unable to clear the cache: %d: %s", i, err)
			continue
		}
		break
	}
	if err != nil {
		msg.Die("Unable to clear the cache: %s", err)
	}

	cache.SetupReset()
	cache.Setup()

	msg.Info("Glide cache has been cleared.")
}
