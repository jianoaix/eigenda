package flags

import (
	"time"

	"github.com/Layr-Labs/eigenda/common"
	"github.com/urfave/cli"
)

const (
	FlagPrefix   = "ejector"
	envVarPrefix = "EJECTOR"
)

var (
	DataApiHostnameFlag = cli.StringFlag{
		Name:     common.PrefixFlag(FlagPrefix, "eigenda-dataapi-hostname"),
		Usage:    "HostName of EigenDA DataApi server",
		Required: true,
		EnvVar:   common.PrefixEnvVar(envVarPrefix, "EIGENDA_DATAAPI_HOSTNAME"),
	}
	NonsigningRateApiPathFlag = cli.StringFlag{
		Name:     common.PrefixFlag(FlagPrefix, "eigenda-nonsigning-rate-api-path"),
		Usage:    "API path for the nonsigning rate",
		Required: true,
		EnvVar:   common.PrefixEnvVar(envVarPrefix, "EIGENDA_NONSIGNING_RATE_API_PATH"),
	}
	BlsOperatorStateRetrieverFlag = cli.StringFlag{
		Name:     common.PrefixFlag(FlagPrefix, "bls-operator-state-retriever"),
		Usage:    "Address of the BLS Operator State Retriever",
		Required: true,
		EnvVar:   common.PrefixEnvVar(envVarPrefix, "BLS_OPERATOR_STATE_RETRIVER"),
	}
	EigenDAServiceManagerFlag = cli.StringFlag{
		Name:     common.PrefixFlag(FlagPrefix, "eigenda-service-manager"),
		Usage:    "Address of the EigenDA Service Manager",
		Required: true,
		EnvVar:   common.PrefixEnvVar(envVarPrefix, "EIGENDA_SERVICE_MANAGER"),
	}
	EnableMetricsFlag = cli.BoolFlag{
		Name:     common.PrefixFlag(FlagPrefix, "enable-metrics"),
		Usage:    "start metrics server",
		Required: true,
		EnvVar:   common.PrefixEnvVar(envVarPrefix, "ENABLE_METRICS"),
	}
	SocketAddrFlag = cli.StringFlag{
		Name:     common.PrefixFlag(FlagPrefix, "socket-addr"),
		Usage:    "the socket address of the ejector server",
		EnvVar:   common.PrefixEnvVar(envVarPrefix, "SOCKET_ADDR"),
		Required: true,
	}
	AllowOriginsFlag = cli.StringSliceFlag{
		Name:     common.PrefixFlag(FlagPrefix, "allow-origins"),
		Usage:    "Set the allowed origins for CORS requests",
		EnvVar:   common.PrefixEnvVar(envVarPrefix, "ALLOW_ORIGINS"),
		Required: true,
	}

	// Optional flags
	EjectionIntervalFlag = cli.DurationFlag{
		Name:     common.PrefixFlag(FlagPrefix, "ejection-interval"),
		Usage:    "Interval at which to perform periodic ejection. If set to 0, periodic ejection will be disabled.",
		Required: false,
		Value:    24 * time.Day,
		EnvVar:   common.PrefixEnvVar(EnvVarPrefix, "EJECTION_INTERVAL"),
	}
	MetricsPortFlag = cli.StringFlag{
		Name:     common.PrefixFlag(FlagPrefix, "metrics-port"),
		Usage:    "Port at which node listens for metrics calls",
		Required: false,
		Value:    "9091",
		EnvVar:   common.PrefixEnvVar(EnvVarPrefix, "METRICS_PORT"),
	}
	ServerModeFlag = cli.StringFlag{
		Name:     common.PrefixFlag(FlagPrefix, "server-mode"),
		Usage:    "Set the mode of the server (debug, release or test)",
		EnvVar:   common.PrefixEnvVar(envVarPrefix, "SERVER_MODE"),
		Required: false,
		Value:    "debug",
	}
)
