package network_delegation

import (
	"github.com/Oneledger/protocol/action"
	"github.com/pkg/errors"
)

func EnableNetworkDelegation(r action.Router) error {
	err := r.AddHandler(action.NETWORKDELEGATE, networkDelegateTx{})
	if err != nil {
		return errors.Wrap(err, "NetworkDelegate")
	}
	err = r.AddHandler(action.WITHDRAW_NETWORK_DELEGATION, withdrawNetworkDelegationTx{})
	if err != nil {
		return errors.Wrap(err, "WithdrawNetworkDelegation")
	}

	return nil
}
