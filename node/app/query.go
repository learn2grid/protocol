/*
	Copyright 2017-2018 OneLedger

	Implement all of the query mechanics for the node and the chain
*/
package app

import (
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/Oneledger/protocol/node/comm"
	"github.com/Oneledger/protocol/node/data"
	"github.com/Oneledger/protocol/node/err"
	"github.com/Oneledger/protocol/node/id"
	"github.com/Oneledger/protocol/node/log"
	"github.com/Oneledger/protocol/node/version"
)

// Top-level list of all query types
func HandleQuery(app Application, path string, message []byte) []byte {

	switch path {
	case "/identity":
		return HandleIdentityQuery(app, message)

	case "/account":
		return HandleAccountQuery(app, message)

	case "/utxo":
		return HandleUtxoQuery(app, message)

	case "/version":
		return HandleVersionQuery(app, message)

	case "/accountKey":
		return HandleAccountKeyQuery(app, message)

	case "/balance":
		return HandleBalanceQuery(app, message)
	}

	return HandleError("Unknown Path", path, message)
}

// Get the account information for a given user
func HandleAccountKeyQuery(app Application, message []byte) []byte {
	log.Debug("AccountKeyQuery", "message", message)

	text := string(message)

	name := ""
	parts := strings.Split(text, "=")
	if len(parts) > 1 {
		name = parts[1]
	}
	return AccountKey(app, name)
}

func AccountKey(app Application, name string) []byte {
	identity, _ := app.Identities.FindName(name)

	if identity != nil {
		return []byte(hex.EncodeToString(identity.AccountKey))
	}

	// Maybe this is an AccountName, not an identity
	account, _ := app.Accounts.FindName(name)
	if account != nil {
		return []byte(hex.EncodeToString(account.AccountKey()))
	}

	return []byte(nil)
}

// Get the account information for a given user
func HandleIdentityQuery(app Application, message []byte) []byte {
	log.Debug("IdentityQuery", "message", message)

	text := string(message)

	name := ""
	parts := strings.Split(text, "=")
	if len(parts) > 1 {
		name = parts[1]
	}
	return IdentityInfo(app, name)
}

func IdentityInfo(app Application, name string) []byte {
	if name == "" {
		identities := app.Identities.FindAll()

		count := fmt.Sprintf("%d", len(identities))
		buffer := "Answer: " + count + " "

		for _, curr := range identities {
			buffer += curr.AsString() + ", "
		}
		return []byte(buffer)
	}
	identity, _ := app.Identities.FindName(name)

	return []byte(identity.AsString())
}

// Get the account information for a given user
func HandleAccountQuery(app Application, message []byte) []byte {
	log.Debug("AccountQuery", "message", message)

	text := string(message)

	name := ""
	parts := strings.Split(text, "=")
	if len(parts) > 1 {
		name = parts[1]
	}
	return AccountInfo(app, name)
}

type AccountQuery struct {
	Accounts []id.AccountExport
}

func getAccountExport(app Application, account id.Account) id.AccountExport {
	export := account.Export()
	if export.Type == "OneLedger" {
		export.Balance = GetBalance(app, account)
	}
	return export
}

// AccountInfo returns the information for a given account
func AccountInfo(app Application, name string) []byte {
	if name == "" {
		var result AccountQuery
		accounts := app.Accounts.FindAll()

		for _, account := range accounts {
			accountExport := getAccountExport(app, account)
			result.Accounts = append(result.Accounts, accountExport)
		}

		buffer, err := comm.Serialize(result)
		if err != nil {
			log.Warn("Failed to Serialize plural AccountInfo query")
		}
		return buffer
	}

	account, _ := app.Accounts.FindName(name)
	accountExport := getAccountExport(app, account)
	result := &AccountQuery{Accounts: []id.AccountExport{accountExport}}

	buffer, err := comm.Serialize(result)
	if err != nil {
		log.Warn("Failed to Serialize singular AccountInfo query")
	}

	log.Debug("Accounts", "name", name, "account", account)
	return buffer
}

func HandleUtxoQuery(app Application, message []byte) []byte {
	log.Debug("UtxoQuery", "message", message)

	text := string(message)

	name := ""
	parts := strings.Split(text, "=")
	if len(parts) > 1 {
		name = parts[1]
	}
	result := UtxoInfo(app, name)
	log.Debug("Returning", "result", string(result))
	return result
}

func UtxoInfo(app Application, name string) []byte {
	buffer := ""
	if name == "" {
		entries := app.Utxo.FindAll()
		for key, value := range entries {
			account, errs := app.Accounts.FindKey([]byte(key))
			if errs != err.SUCCESS {
				log.Fatal("Accounts", "err", errs, "key", key)
			}

			var name string
			if account == nil {
				name = fmt.Sprintf("%X", key)
			} else {
				name = account.Name() + "@" + fmt.Sprintf("%X", key)
			}

			if value != nil {
				buffer += name + ":" + value.AsString() + ", "
			} else {
				buffer += name + ":EMPTY, "
			}

		}

	} else {
		value := app.Utxo.Find(data.DatabaseKey(name))
		buffer += name + ":" + value.AsString()

	}
	return []byte(buffer)
}

// Get the balancd for an account
func GetBalance(app Application, account id.Account) string {
	result := app.Utxo.Find(account.AccountKey())
	if result == nil {
		log.Debug("Balance Not Found", "key", account.AccountKey())
		return " [nil]"
	}

	return result.AsString()
}

// Return a nicely formatted error message
func HandleError(text string, path string, massage []byte) []byte {
	return []byte("Invalid Query")
}

func HandleVersionQuery(app Application, message []byte) []byte {
	return []byte(version.Current.String())
}

// Get the account information for a given user
func HandleBalanceQuery(app Application, message []byte) []byte {
	log.Debug("BalanceQuery", "message", message)

	text := string(message)

	var key []byte
	parts := strings.Split(text, "=")
	if len(parts) > 1 {
		key, _ = hex.DecodeString(parts[1])
	}
	return Balance(app, key)
}

func Balance(app Application, accountKey []byte) []byte {

	balance := app.Utxo.Find(accountKey)
	if balance == nil {
		//log.Fatal("Balance FAILED", "accountKey", accountKey)
		log.Warn("Balance FAILED", "accountKey", accountKey)
		result := data.NewBalance(0, "OLT")
		balance = &result
	}
	//log.Debug("Balance", "key", accountKey, "balance", balance)

	buffer, _ := comm.Serialize(balance)
	return buffer
}
