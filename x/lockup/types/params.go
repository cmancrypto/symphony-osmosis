package types

import (
	fmt "fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
)

// Parameter store keys.
var (
	KeyForceUnlockAllowedAddresses         = []byte("ForceUnlockAllowedAddresses")
	KeyNonOwnerForceUnlockAllowedAddresses = []byte("NonOwnerForceUnlockAllowedAddresses")

	_ paramtypes.ParamSet = &Params{}
)

// ParamTable for lockup module.
func ParamKeyTable() paramtypes.KeyTable {
	return paramtypes.NewKeyTable().RegisterParamSet(&Params{})
}

func NewParams(forceUnlockAllowedAddresses, nonOwnerForceUnlockAllowedAddresses []string) Params {
	return Params{
		ForceUnlockAllowedAddresses:         forceUnlockAllowedAddresses,
		NonOwnerForceUnlockAllowedAddresses: nonOwnerForceUnlockAllowedAddresses,
	}
}

// DefaultParams returns default lockup module parameters.
func DefaultParams() Params {
	return Params{
		ForceUnlockAllowedAddresses:         []string{},
		NonOwnerForceUnlockAllowedAddresses: []string{},
	}
}

// validate params.
func (p Params) Validate() error {
	if err := validateAddresses(p.ForceUnlockAllowedAddresses); err != nil {
		return err
	}
	if err := validateAddresses(p.NonOwnerForceUnlockAllowedAddresses); err != nil {
		return err
	}
	return nil
}

// Implements params.ParamSet.
func (p *Params) ParamSetPairs() paramtypes.ParamSetPairs {
	return paramtypes.ParamSetPairs{
		paramtypes.NewParamSetPair(KeyForceUnlockAllowedAddresses, &p.ForceUnlockAllowedAddresses, validateAddresses),
		paramtypes.NewParamSetPair(KeyNonOwnerForceUnlockAllowedAddresses, &p.NonOwnerForceUnlockAllowedAddresses, validateAddresses),
	}
}

func validateAddresses(i interface{}) error {
	addresses, ok := i.([]string)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}
	for _, address := range addresses {
		_, err := sdk.AccAddressFromBech32(address)
		if err != nil {
			return err
		}
	}

	return nil
}
