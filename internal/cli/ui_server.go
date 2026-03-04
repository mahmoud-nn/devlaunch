package cli

import (
	"fmt"

	"github.com/mahmoud-nn/devlaunch/internal/web"
)

func NewUIServer(host string, port int) *web.Server {
	return web.NewServer(fmt.Sprintf("%s:%d", host, port))
}
