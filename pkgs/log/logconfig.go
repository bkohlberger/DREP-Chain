package log

import (
	"github.com/drep-project/drep-chain/common"
	"gopkg.in/urfave/cli.v1"
)

var (
	//log
	LogDirFlag = common.DirectoryFlag{
		Name:  "logdir",
		Usage: "Directory for the logdir (default = inside the homedir)",
	}
	LogLevelFlag = cli.IntFlag{
		Name:  "loglevel",
		Usage: "Logging level: 0=silent, 1=error, 2=warn, 3=info, 4=debug, 5=detail",
		Value: 3,
	}
	VmoduleFlag = cli.StringFlag{
		Name:  "vmodule",
		Usage: "Per-module verbosity: comma-separated list of <pattern>=<level> (e.g. eth/*=5,p2p=4)",
		Value: "",
	}
)

type LogConfig struct {
	DataDir  string `json:"-"`
	LogLevel int    `json:"logLevel"`
	Vmodule  string `json:"vmodule,omitempty"`
}
