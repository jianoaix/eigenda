package ejector

import (
	"github.com/Layr-Labs/eigenda/common"
	"github.com/Layr-Labs/eigenda/common/geth"
	"github.com/Layr-Labs/eigenda/operators/churner/flags"
	"github.com/urfave/cli"
)

type Config struct {
	EthClientConfig geth.EthClientConfig
	LoggerConfig    common.LoggerConfig
	MetricsConfig   MetricsConfig
}

func NewConfig(ctx *cli.Context) (*Config, error) {
	if err != nil {
		return nil, err
	}
	return &Config{
		EthClientConfig: geth.ReadEthClientConfig(ctx),
		LoggerConfig:    *loggerConfig,
		MetricsConfig: MetricsConfig{
			HTTPPort:      ctx.GlobalString(flags.MetricsHTTPPort.Name),
			EnableMetrics: ctx.GlobalBool(flags.EnableMetrics.Name),
		},
	}, nil
}
