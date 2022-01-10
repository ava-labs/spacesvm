// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package vm

import (
	"net/http"
)

type PrivateService struct {
	vm *VM
}

type SetBeneficiaryArgs struct {
	Beneficiary []byte `serialize:"true" json:"beneficiary"`
}

func (svc *PrivateService) SetBeneficiary(_ *http.Request, args *SetBeneficiaryArgs, _ *struct{}) error {
	svc.vm.SetBeneficiary(args.Beneficiary)
	return nil
}
