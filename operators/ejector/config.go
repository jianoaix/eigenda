package ejector

import (
	"github.com/Layr-Labs/eigenda/common"
	"github.com/Layr-Labs/eigenda/common/geth"
	"github.com/Layr-Labs/eigenda/operators/churner/flags"
	"github.com/urfave/cli"
)

const (
	AppName   = "da-ejector"
	Version   = "0.6.1"
	GitCommit = ""
	GitDate   = ""
)

type Config struct {
	EthClientConfig geth.EthClientConfig
	LoggerConfig    common.LoggerConfig
	MetricsConfig   MetricsConfig

	DataApiHostName       string
	NonsigningRateApiPath string

	SocketAddr   string
	ServerMode   string
	AllowOrigins []string
}

func NewConfig(ctx *cli.Context) (*Config, error) {
	loggerConfig, err := common.ReadLoggerCLIConfig(ctx, flags.FlagPrefix)
	if err != nil {
		return nil, err
	}
	return &Config{
		EthClientConfig: geth.ReadEthClientConfig(ctx),
		LoggerConfig:    *loggerConfig,
		MetricsConfig: MetricsConfig{
			MetricsPort:   ctx.GlobalString(flags.MetricsPortFlag.Name),
			EnableMetrics: ctx.GlobalBool(flags.EnableMetrics.Name),
		},
		SocketAddr:                    ctx.GlobalString(flags.SocketAddrFlag.Name),
		ServerMode:                    ctx.GlobalString(flags.ServerModeFlag.Name),
		AllowOrigins:                  ctx.GlobalStringSlice(flags.AllowOriginsFlag.Name),
		DataApiHostName:               ctx.GlobalString(flags.DataApiHostnameFlag.Name),
		NonsigningRateApiPath:         ctx.GlobalString(flags.NonsigningRateApiPath.Name),
		BLSOperatorStateRetrieverAddr: ctx.GlobalString(flags.BlsOperatorStateRetrieverFlag.Name),
		EigenDAServiceManagerAddr:     ctx.GlobalString(flags.EigenDAServiceManagerFlag.Name),
	}, nil
}
