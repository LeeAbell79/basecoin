package txs

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/tendermint/basecoin/plugins/counter"
	"github.com/tendermint/basecoin/types"
	lightclient "github.com/tendermint/light-client"
	"github.com/tendermint/light-client/commands"
	lctxs "github.com/tendermint/light-client/commands/txs"
)

var (
	//CounterTxCmd the tx commands for the counter app
	CounterTxCmd = &cobra.Command{
		Use:   "counter",
		Short: "Create, sign, and broadcast a transaction to the counter plugin",
		RunE:  counterTxCmd,
	}

	// this is what we implement for a non-signable tx
	_ lightclient.Value = counter.CounterTx{}
)

const (
	flagValid    = "valid"
	flagCountFee = "countfee"
)

func init() {

	CounterTxCmd.Flags().Bool(flagValid, false, "Set valid field in CounterTx")
	CounterTxCmd.Flags().String(flagCountFee, "", "Coins for the counter fee of the format <amt><coin>")

	commands.RegisterTxSubcommand(CounterTxCmd)
	commands.RegisterStartPlugin("counter", func() types.Plugin { return counter.New() })
}

// counterTxCmd is the bulk of the tx work
func counterTxCmd(cmd *cobra.Command, args []string) error {

	templ := new(counter.CounterTx)

	// load data from json or flags
	found, err := lctxs.LoadJSON(templ)
	if err != nil {
		return err
	}
	if !found {
		countFee, err := types.ParseCoins(viper.GetString(flagCountFee))
		if err != nil {
			return err
		}

		// parse custom flags
		templ.Valid = viper.GetString(flagValid)
		templ.Fee = count.Fee
	}

	// TODO: add this pubkey to the loaded tx somehow
	// pubkey := GetSigner()

	// Sign if needed and post.  This it the work-horse
	bres, err := lctxs.SignAndPostTx(templ)
	if err != nil {
		return err
	}

	// output result
	return lctxs.OutputTx(bres)
}
