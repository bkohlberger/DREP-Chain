package rpc

import (
	"gopkg.in/urfave/cli.v1"
	"strings"

	"github.com/drep-project/DREP-Chain/common"

	"github.com/drep-project/rpc"
)

var (
	// RPC settings
	HTTPEnabledFlag = cli.BoolFlag{
		Name:  "http",
		Usage: "StartMiner the HTTP-RPC server",
	}
	HTTPListenAddrFlag = cli.StringFlag{
		Name:  "httpaddr",
		Usage: "HTTP-RPC server listening interface",
		Value: rpc.DefaultHTTPHost,
	}
	HTTPPortFlag = cli.IntFlag{
		Name:  "httpport",
		Usage: "HTTP-RPC server listening port",
		Value: rpc.DefaultHTTPPort,
	}
	HTTPCORSDomainFlag = cli.StringFlag{
		Name:  "httpcorsdomain",
		Usage: "Comma separated list of domains from which to accept cross origin requests (browser enforced)",
		Value: "",
	}
	HTTPVirtualHostsFlag = cli.StringFlag{
		Name:  "httpvhosts",
		Usage: "Comma separated list of virtual hostnames from which to accept requests (server enforced). Accepts '*' wildcard.",
		Value: strings.Join([]string{"localhost"}, ","),
	}
	HTTPApiFlag = cli.StringFlag{
		Name:  "httpapi",
		Usage: "API's offered over the HTTP-RPC interface",
		Value: "",
	}
	IPCDisabledFlag = cli.BoolFlag{
		Name:  "ipcdisable",
		Usage: "Disable the IPC-RPC server",
	}
	IPCPathFlag = common.DirectoryFlag{
		Name:  "ipcpath",
		Usage: "Filename for IPC socket/pipe within the datadir (explicit paths escape it)",
	}
	WSEnabledFlag = cli.BoolFlag{
		Name:  "ws",
		Usage: "StartMiner the WS-RPC server",
	}
	WSListenAddrFlag = cli.StringFlag{
		Name:  "wsaddr",
		Usage: "WS-RPC server listening interface",
		Value: rpc.DefaultWSHost,
	}
	WSPortFlag = cli.IntFlag{
		Name:  "wsport",
		Usage: "WS-RPC server listening port",
		Value: rpc.DefaultWSPort,
	}
	WSApiFlag = cli.StringFlag{
		Name:  "wsapi",
		Usage: "API's offered over the WS-RPC interface",
		Value: "",
	}
	WSAllowedOriginsFlag = cli.StringFlag{
		Name:  "wsorigins",
		Usage: "Origins from which to accept websockets requests",
		Value: "",
	}
	// RPC settings
	RESTEnabledFlag = cli.BoolFlag{
		Name:  "rest",
		Usage: "StartMiner the REST-RPC server",
	}
	RESTListenAddrFlag = cli.StringFlag{
		Name:  "restaddr",
		Usage: "REST-RPC server listening interface",
		Value: rpc.DefaultRestHost,
	}
	RESTPortFlag = cli.IntFlag{
		Name:  "restport",
		Usage: "REST-RPC server listening port",
		Value: rpc.DefaultRestPort,
	}
)
