package utils

import "github.com/widaT/golib/config"

var Conf config.Config

func init() {
	Conf = config.NewConfig("web.conf")
}
