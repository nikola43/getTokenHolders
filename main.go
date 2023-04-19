package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"os"
	"strings"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	token "github.com/nikola43/nikola43/getTokenHolders/token"
)

const (
	rpcUrl       = "https://rpc.ankr.com/eth"                   // Replace with your Infura project ID
)

type ERC20 struct {
	contractAbi *abi.ABI
	address     common.Address
	client      *ethclient.Client
}

var transferEvent struct {
	Value *big.Int
}

func main() {
	client, err := ethclient.Dial(rpcUrl)
	if err != nil {
		log.Fatalf("Failed to connect to Ethereum network: %v", err)
	}
	defer client.Close()

	tokenList := ReadTokenListFromFile("tokens.json")

	for _, tokenAddress := range tokenList {
		fmt.Println("Token address: ", tokenAddress)

		tokenAddr := common.HexToAddress(tokenAddress)
		token, err := NewERC20(tokenAddr, client)
		if err != nil {
			log.Fatalf("Failed to instantiate token contract: %v", err)
		}

		// init token contract instance
		tokenInstance, instanceErr := token.NewToken(common.HexToAddress(tokenAddr), client)



		query := ethereum.FilterQuery{
			Addresses: []common.Address{tokenAddr},
			FromBlock: big.NewInt(17081000),
			ToBlock:   big.NewInt(17081327),
		}

		logs, err := client.FilterLogs(context.Background(), query)
		if err != nil {
			log.Fatalf("Failed to query logs: %v", err)
		}

		tokenHolders := make(map[common.Address]*big.Int)
		for _, vLog := range logs {

			if len(vLog.Topics) < 3 {
				continue
			}

			err := token.contractAbi.UnpackIntoInterface(&transferEvent, "Transfer", vLog.Data)
			if err != nil {
				log.Fatalf("Failed to unpack event data: %v", err)
			}

			from := common.BytesToAddress(vLog.Topics[1].Bytes())
			to := common.BytesToAddress(vLog.Topics[2].Bytes())

			fmt.Println(transferEvent)

			_, senderHasBalance := tokenHolders[from]
			if !senderHasBalance {
				tokenHolders[from] = big.NewInt(0)
			}

			_, receiverHasBalance := tokenHolders[to]
			if !receiverHasBalance {
				tokenHolders[to] = big.NewInt(0)
			}

			tokenHolders[from].Sub(tokenHolders[from], transferEvent.Value)
			tokenHolders[to].Add(tokenHolders[to], transferEvent.Value)
		}

		for holder, balance := range tokenHolders {
			if balance.Cmp(big.NewInt(0)) > 0 {

				balanceEthUnits := new(big.Float).Quo(new(big.Float).SetInt(balance), big.NewFloat(1e18))
				fmt.Printf("Address: %s, Balance: %s\n", holder.Hex(), balanceEthUnits.String())
			}
		}

	}

}

/**
 * ReadTokenListFromFile reads a list of tokens from a json file and return array of strings
 */
func ReadTokenListFromFile(filename string) []string {
	file, err := os.Open(filename)
	if err != nil {
		log.Fatalf("Failed to open file: %v", err)
	}
	defer file.Close()

	fi, err := file.Stat()
	if err != nil {
		log.Fatalf("Failed to get file info: %v", err)
	}

	data := make([]byte, fi.Size())
	_, err = file.Read(data)
	if err != nil {
		log.Fatalf("Failed to read file: %v", err)
	}

	var tokenList []string
	err = json.Unmarshal(data, &tokenList)
	if err != nil {
		log.Fatalf("Failed to unmarshal json: %v", err)
	}

	return tokenList

}

func ReadAbiFromFile(filename string) string {
	file, err := os.Open(filename)
	if err != nil {
		log.Fatalf("Failed to open file: %v", err)
	}
	defer file.Close()

	fi, err := file.Stat()
	if err != nil {
		log.Fatalf("Failed to get file info: %v", err)
	}

	data := make([]byte, fi.Size())
	_, err = file.Read(data)
	if err != nil {
		log.Fatalf("Failed to read file: %v", err)
	}

	return string(data)
}

func NewERC20(address common.Address, client *ethclient.Client) (*ERC20, error) {
	erc20ABI := ReadAbiFromFile("Token.json")
	parsedABI, err := abi.JSON(strings.NewReader(erc20ABI))
	if err != nil {
		return nil, err
	}

	return &ERC20{
		contractAbi: &parsedABI,
		address:     address,
		client:      client,
	}, nil
}
