package main

import (
	"context"
	"fmt"
	"math"

	"github.com/portto/solana-go-sdk/client"
	"github.com/portto/solana-go-sdk/common"
	"github.com/portto/solana-go-sdk/program/token"
	"github.com/portto/solana-go-sdk/rpc"
)

var addressByToken = map[string]string{
	"iot": "iotEVVZLEywoTn1QdwNPddxPWszn3zFhEot3MfL9fns",
	"hnt": "hntyVP6YFm1Hg25TN9WGLqM12b8TQmcknKrdu1oxWux",
}

/*
 We have to do this weirdness as this Go lib
 doesn't provide the 'decimals' field
 https://docs.solana.com/api/http#token-balances-structure
*/
var divisorByToken = map[string]float64{
	"iotEVVZLEywoTn1QdwNPddxPWszn3zFhEot3MfL9fns": math.Pow(10, 6),
	"hntyVP6YFm1Hg25TN9WGLqM12b8TQmcknKrdu1oxWux": math.Pow(10, 8),
}

func fetchSolanaAccountBalance(address string, token string) (float64, error) {

	if token == "sol" {
		return fetchSolanaBalance(address)
	}

	tokenAddress := addressByToken[token]

	if tokenAddress == "" {
		return 0, fmt.Errorf("Error")
	}

	return fetchSPLBalance(address, tokenAddress)
}

func fetchSolanaBalance(address string) (float64, error) {
	c := client.NewClient(rpc.MainnetRPCEndpoint)

	balance, err := c.GetBalance(
		context.TODO(),
		address,
	)

	if err != nil {
		return 0, err
	}

	return float64(balance) / math.Pow(10, 9), nil
}

func filterAccountsByToken(accounts map[common.PublicKey]token.TokenAccount, tokenAddress string) (token.TokenAccount, error) {
	for pk := range accounts {
		if accounts[pk].Mint.String() == tokenAddress {
			return accounts[pk], nil
		}
	}

	return token.TokenAccount{}, fmt.Errorf("Unable to find token on account")
}

func fetchSPLBalance(address string, tokenAddress string) (float64, error) {
	c := client.NewClient(rpc.MainnetRPCEndpoint)

	accounts, err := c.GetTokenAccountsByOwner(
		context.TODO(),
		address,
	)

	if err != nil {
		return 0, err
	}

	account, err := filterAccountsByToken(accounts, tokenAddress)

	if err != nil {
		fmt.Printf("Unable to find token balance on account")
	}

	return float64(account.Amount) / divisorByToken[tokenAddress], nil
}
