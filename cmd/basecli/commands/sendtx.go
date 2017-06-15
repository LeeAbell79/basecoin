package commands

import (
	"encoding/hex"
	"strings"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	bc "github.com/tendermint/basecoin/types"
	wire "github.com/tendermint/go-wire"
	lightclient "github.com/tendermint/light-client"
	lctxs "github.com/tendermint/light-client/commands/txs"
	"github.com/tendermint/tendermint/rpc/client"
)

// TxBytes returns the transaction data as well as all signatures
// It should return an error if Sign was never called
func (s *SendTx) TxBytes() ([]byte, error) {
	// TODO: verify it is signed

	// Code and comment from: basecoin/cmd/commands/tx.go
	// Don't you hate having to do this?
	// How many times have I lost an hour over this trick?!
	txBytes := wire.BinaryBytes(struct {
		bc.Tx `json:"unwrap"`
	}{s.Tx})
	return txBytes, nil
}

////////////////////////////////////////////////////////////////////////

var (
	SendTxCmd = &cobra.Command{
		Use:   "send",
		Short: "A SendTx transaction, for sending tokens around",
		RunE:  sendTxCmd,
	}

	_ lightclient.Value = SendTx{}
)

const (
	flagFrom   = "from"
	flagAmount = "amount"
	flagGas    = "gas"
	flagFee    = "fee"
	flagSeq    = "sequence"
	flagTo     = "to"
)

func init() {
	SendTxCmd.Flags().String(flagFrom, "key.json", "Path to a private key to sign the transaction")
	SendTxCmd.Flags().String(flagAmount, "", "Coins to send in transaction of the format <amt><coin>,<amt2><coin2>,... (eg: 1btc,2gold,5silver)")
	SendTxCmd.Flags().Int(flagGas, 0, "The amount of gas for the transaction")
	SendTxCmd.Flags().String(flagFee, "0coin", "Coins for the transaction fee of the format <amt><coin>")
	SendTxCmd.Flags().Int(flagSeq, -1, "Sequence number for the account (-1 to autocalculate)")
	SendTxCmd.Flags().String(flagTo, "", "Destination address for the transaction")
}

func sendTxCmd(cmd *cobra.Command, args []string) error {

	templ := new(bc.SendTx)

	// load data from json or flags
	found, err := LoadJSON(templ)
	if err != nil {
		return err
	}
	if !found {
		var toHex string
		var chainPrefix string
		spl := strings.Split(toFlag, "/")
		switch len(spl) {
		case 1:
			toHex = spl[0]
		case 2:
			chainPrefix = spl[0]
			toHex = spl[1]
		default:
			return errors.Errorf("To address has too many slashes")
		}

		// convert destination address to bytes
		to, err := hex.DecodeString(StripHex(toHex))
		if err != nil {
			return errors.Errorf("To address is invalid hex: %v\n", err)
		}

		if chainPrefix != "" {
			to = []byte(chainPrefix + "/" + string(to))
		}

		// load the priv key
		privKey, err := LoadKey(fromFlag)
		if err != nil {
			return err
		}

		// get the sequence number for the tx
		sequence, err := getSeq(privKey.Address[:])
		if err != nil {
			return err
		}

		//parse the fee and amounts into coin types
		feeCoin, err := types.ParseCoin(feeFlag)
		if err != nil {
			return err
		}
		amountCoins, err := types.ParseCoins(amountFlag)
		if err != nil {
			return err
		}

		// craft the tx
		input := bc.NewTxInput(privKey.PubKey, amountCoins, sequence)
		output := bc.TxOutput{
			Address: to,
			Coins:   amount,
		}
		//tx := &types.SendTx{
		//Gas:     int64(gasFlag),
		//Fee:     feeCoin,
		//Inputs:
		//Outputs:
		//}

		templ.Gas = viper.GetInt64(flagGas)
		templ.Fee = feeCoin
		templ.Inputs = []types.TxInput{input}
		templ.Outputs = []types.TxOutput{output}
	}

	// sign that puppy
	signBytes := templ.SignBytes(chainIDFlag) ////////////////////////// XXX how should we be getting the ChainID here?
	templ.Inputs[0].Signature = privKey.Sign(signBytes)

	// Sign if needed and post.  This it the work-horse
	bres, err := lctxs.SignAndPostTx(templ)
	if err != nil {
		return err
	}

	// output result
	return lctxs.OutputTx(bres)
}

// if the sequence flag is set, return it;
// else, fetch the account by querying the app and return the sequence number
func getSeq(address []byte) (int, error) {
	if seqFlag >= 0 {
		return seqFlag, nil
	}

	//////////////////////////////////////////////////////////////////// XXX what to do here? should be proof?
	httpClient := client.NewHTTP(txNodeFlag, "/websocket")
	acc, err := getAccWithClient(httpClient, address)
	if err != nil {
		return 0, err
	}
	return acc.Sequence + 1, nil
}
