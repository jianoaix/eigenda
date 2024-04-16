package main

import (
	"fmt"
	"log"
	"os"

	"github.com/Layr-Labs/eigenda/common"
	"github.com/Layr-Labs/eigenda/common/geth"
	"github.com/Layr-Labs/eigenda/operators/ejector"
	"github.com/Layr-Labs/eigenda/operators/ejector/flags"
	"github.com/urfave/cli"
)

func main() {
	app := cli.NewApp()
	app.Flags = flags.Flags
	app.Version = fmt.Sprintf("%s-%s-%s", ejector.Version, ejector.GitCommit, ejector.GitDate)
	app.Name = ejector.AppName
	app.Usage = "EigenDA Ejector"
	app.Description = "Service for ejecting the non-performing operators"

	app.Action = EjectorMain
	err := app.Run(os.Args)
	if err != nil {
		log.Fatalf("application failed: %v", err)
	}

	select {}
}

func EjectorMain(ctx *cli.Context) error {
	config, err := ejector.NewConfig(ctx)
	if err != nil {
		return err
	}

	logger, err := common.NewLogger(config.LoggerConfig)
	if err != nil {
		return err
	}

	ethConfig := geth.EthClientConfig{
		RPCURLs:          []string{config.ChainRpcUrl},
		PrivateKeyString: *privateKey,
		NumConfirmations: config.NumConfirmations,
	}
	client, err := geth.NewClient(ethConfig, gethcommon.Address{}, 0, logger)
	if err != nil {
		log.Printf("Error: failed to create eth client: %v", err)
		return
	}

	tx, err := eth.NewTransactor(logger, client, config.BLSOperatorStateRetrieverAddr, config.EigenDAServiceManagerAddr)
	if err != nil {
		log.Printf("Error: failed to create EigenDA transactor: %v", err)
		return
	}

	return nil
}
