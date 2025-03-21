package sqldb

import (
	"errors"
	"fmt"
	"strings"

	sdkmath "cosmossdk.io/math"
	"gorm.io/gorm"

	corespdb "github.com/bnb-chain/greenfield-storage-provider/core/spdb"
	sptypes "github.com/bnb-chain/greenfield/x/sp/types"
)

// UpdateAllSp update(maybe overwrite) all sp info in db
func (s *SpDBImpl) UpdateAllSp(spList []*sptypes.StorageProvider) error {
	for _, value := range spList {
		queryReturn := &SpInfoTable{}
		// 1. check record whether exists
		result := s.db.Where("operator_address = ? and is_own = false", value.GetOperatorAddress()).First(queryReturn)
		recordNotFound := errors.Is(result.Error, gorm.ErrRecordNotFound)
		if result.Error != nil && !recordNotFound {
			return fmt.Errorf("failed to query record in sp info table: %s", result.Error)
		}
		// 2. if there is no record, insert new record; otherwise delete old record, then insert new record
		if recordNotFound { // insert
			if err := s.insertNewRecordInSpInfoTable(value); err != nil {
				return err
			}
		} else { // update
			result = s.db.Model(&SpInfoTable{}).
				Where("operator_address = ? and is_own = false", value.GetOperatorAddress()).Updates(&SpInfoTable{
				OperatorAddress: value.GetOperatorAddress(),
				IsOwn:           false,
				ID:              value.GetId(),
				FundingAddress:  value.GetFundingAddress(),
				SealAddress:     value.GetSealAddress(),
				ApprovalAddress: value.GetApprovalAddress(),
				TotalDeposit:    value.GetTotalDeposit().String(),
				Status:          int32(value.Status),
				Endpoint:        value.GetEndpoint(),
				Moniker:         value.GetDescription().Moniker,
				Identity:        value.GetDescription().Identity,
				Website:         value.GetDescription().Website,
				SecurityContact: value.GetDescription().SecurityContact,
				Details:         value.GetDescription().Identity,
			})
			if result.Error != nil {
				return fmt.Errorf("failed to detele record in sp info table: %s", result.Error)
			}
		}
	}
	return nil
}

// insertNewRecordInSpInfoTable insert a new record in sp info table
func (s *SpDBImpl) insertNewRecordInSpInfoTable(sp *sptypes.StorageProvider) error {
	insertRecord := &SpInfoTable{
		OperatorAddress: sp.GetOperatorAddress(),
		IsOwn:           false,
		ID:              sp.GetId(),
		FundingAddress:  sp.GetFundingAddress(),
		SealAddress:     sp.GetSealAddress(),
		ApprovalAddress: sp.GetApprovalAddress(),
		TotalDeposit:    sp.GetTotalDeposit().String(),
		Status:          int32(sp.Status),
		Endpoint:        sp.GetEndpoint(),
		Moniker:         sp.GetDescription().Moniker,
		Identity:        sp.GetDescription().Identity,
		Website:         sp.GetDescription().Website,
		SecurityContact: sp.GetDescription().SecurityContact,
		Details:         sp.GetDescription().Identity,
	}
	result := s.db.Create(insertRecord)
	if result.Error != nil || result.RowsAffected != 1 {
		return fmt.Errorf("failed to insert record in sp info table: %s", result.Error)
	}
	return nil
}

// FetchAllSp get all sp info
func (s *SpDBImpl) FetchAllSp(status ...sptypes.Status) ([]*sptypes.StorageProvider, error) {
	queryReturn := []SpInfoTable{}
	if len(status) == 0 {
		result := s.db.Where("is_own = false").Find(&queryReturn)
		if result.Error != nil {
			return nil, fmt.Errorf("failed to query sp info table: %s", result.Error)
		}
	} else {
		for _, val := range status {
			temp := []SpInfoTable{}
			result := s.db.Where("is_own = false and status = ?", int32(val)).Find(&temp)
			if result.Error != nil {
				return nil, fmt.Errorf("failed to query sp info table: %s", result.Error)
			}
			queryReturn = append(queryReturn, temp...)
		}
	}
	records := []*sptypes.StorageProvider{}
	for _, value := range queryReturn {
		totalDeposit, ok := sdkmath.NewIntFromString(value.TotalDeposit)
		if !ok {
			return records, fmt.Errorf("failed to parse int")
		}
		records = append(records, &sptypes.StorageProvider{
			Id:              value.ID,
			OperatorAddress: value.OperatorAddress,
			FundingAddress:  value.FundingAddress,
			SealAddress:     value.SealAddress,
			ApprovalAddress: value.ApprovalAddress,
			TotalDeposit:    totalDeposit,
			Status:          sptypes.Status(value.Status),
			Endpoint:        value.Endpoint,
			Description: sptypes.Description{
				Moniker:         value.Moniker,
				Identity:        value.Identity,
				Website:         value.Website,
				SecurityContact: value.SecurityContact,
				Details:         value.Details,
			},
		})
	}
	return records, nil
}

// FetchAllSpWithoutOwnSp get all spp info without own sp info, own sp is identified by is_own field in db
func (s *SpDBImpl) FetchAllSpWithoutOwnSp(status ...sptypes.Status) ([]*sptypes.StorageProvider, error) {
	ownSp, err := s.GetOwnSpInfo()
	if err != nil {
		return nil, err
	}
	queryReturn := []SpInfoTable{}
	if len(status) == 0 {
		result := s.db.Where("operator_address != ?", ownSp.GetOperatorAddress()).Find(&queryReturn)
		if result.Error != nil {
			return nil, fmt.Errorf("failed to query sp info table: %s", result.Error)
		}
	} else {
		for _, val := range status {
			temp := []SpInfoTable{}
			result := s.db.Where("status = ? and operator_address != ?", int32(val), ownSp.GetOperatorAddress()).Find(&temp)
			if result.Error != nil {
				return nil, fmt.Errorf("failed to query sp info table: %s", result.Error)
			}
			queryReturn = append(queryReturn, temp...)
		}
	}

	records := []*sptypes.StorageProvider{}
	for _, value := range queryReturn {
		totalDeposit, ok := sdkmath.NewIntFromString(value.TotalDeposit)
		if !ok {
			return records, fmt.Errorf("failed to parse int")
		}
		records = append(records, &sptypes.StorageProvider{
			Id:              value.ID,
			OperatorAddress: value.OperatorAddress,
			FundingAddress:  value.FundingAddress,
			SealAddress:     value.SealAddress,
			ApprovalAddress: value.ApprovalAddress,
			TotalDeposit:    totalDeposit,
			Status:          sptypes.Status(value.Status),
			Endpoint:        value.Endpoint,
			Description: sptypes.Description{
				Moniker:         value.Moniker,
				Identity:        value.Identity,
				Website:         value.Website,
				SecurityContact: value.SecurityContact,
				Details:         value.Details,
			},
		})
	}
	return records, nil
}

// GetSpByAddress query sp info in db by address and address type
func (s *SpDBImpl) GetSpByAddress(address string, addressType corespdb.SpAddressType) (*sptypes.StorageProvider, error) {
	condition, err := getAddressCondition(addressType)
	if err != nil {
		return nil, err
	}
	queryReturn := &SpInfoTable{}
	result := s.db.First(queryReturn, condition, address)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to query sp info table: %s", result.Error)
	}
	totalDeposit, ok := sdkmath.NewIntFromString(queryReturn.TotalDeposit)
	if !ok {
		return nil, fmt.Errorf("failed to parse int")
	}
	return &sptypes.StorageProvider{
		Id:              queryReturn.ID,
		OperatorAddress: queryReturn.OperatorAddress,
		FundingAddress:  queryReturn.FundingAddress,
		SealAddress:     queryReturn.SealAddress,
		ApprovalAddress: queryReturn.ApprovalAddress,
		TotalDeposit:    totalDeposit,
		Status:          sptypes.Status(queryReturn.Status),
		Endpoint:        queryReturn.Endpoint,
		Description: sptypes.Description{
			Moniker:         queryReturn.Moniker,
			Identity:        queryReturn.Identity,
			Website:         queryReturn.Website,
			SecurityContact: queryReturn.SecurityContact,
			Details:         queryReturn.Details,
		},
	}, nil
}

// getAddressCondition return different condition by address type
func getAddressCondition(addressType corespdb.SpAddressType) (string, error) {
	var condition string
	switch addressType {
	case corespdb.OperatorAddressType:
		condition = "operator_address = ? and is_own = false"
	case corespdb.FundingAddressType:
		condition = "funding_address = ? and is_own = false"
	case corespdb.SealAddressType:
		condition = "seal_address = ? and is_own = false"
	case corespdb.ApprovalAddressType:
		condition = "approval_address = ? and is_own = false"
	default:
		return "", fmt.Errorf("unknown address type")
	}
	return condition, nil
}

// GetSpByEndpoint query sp info by endpoint
func (s *SpDBImpl) GetSpByEndpoint(endpoint string) (*sptypes.StorageProvider, error) {
	queryReturn := &SpInfoTable{}
	result := s.db.First(queryReturn, "endpoint = ? and is_own = false", endpoint)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to query sp info table: %s", result.Error)
	}
	totalDeposit, ok := sdkmath.NewIntFromString(queryReturn.TotalDeposit)
	if !ok {
		return nil, fmt.Errorf("failed to parse int")
	}
	return &sptypes.StorageProvider{
		Id:              queryReturn.ID,
		OperatorAddress: queryReturn.OperatorAddress,
		FundingAddress:  queryReturn.FundingAddress,
		SealAddress:     queryReturn.SealAddress,
		ApprovalAddress: queryReturn.ApprovalAddress,
		TotalDeposit:    totalDeposit,
		Status:          sptypes.Status(queryReturn.Status),
		Endpoint:        queryReturn.Endpoint,
		Description: sptypes.Description{
			Moniker:         queryReturn.Moniker,
			Identity:        queryReturn.Identity,
			Website:         queryReturn.Website,
			SecurityContact: queryReturn.SecurityContact,
			Details:         queryReturn.Details,
		},
	}, nil
}

// GetSpByID query sp info by id
func (s *SpDBImpl) GetSpByID(id uint32) (*sptypes.StorageProvider, error) {
	queryReturn := &SpInfoTable{}
	result := s.db.First(queryReturn, "id = ? and is_own = false", id)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to query sp info table by id: %s", result.Error)
	}
	totalDeposit, ok := sdkmath.NewIntFromString(queryReturn.TotalDeposit)
	if !ok {
		return nil, fmt.Errorf("failed to parse int")
	}
	return &sptypes.StorageProvider{
		Id:              queryReturn.ID,
		OperatorAddress: queryReturn.OperatorAddress,
		FundingAddress:  queryReturn.FundingAddress,
		SealAddress:     queryReturn.SealAddress,
		ApprovalAddress: queryReturn.ApprovalAddress,
		TotalDeposit:    totalDeposit,
		Status:          sptypes.Status(queryReturn.Status),
		Endpoint:        queryReturn.Endpoint,
		Description: sptypes.Description{
			Moniker:         queryReturn.Moniker,
			Identity:        queryReturn.Identity,
			Website:         queryReturn.Website,
			SecurityContact: queryReturn.SecurityContact,
			Details:         queryReturn.Details,
		},
	}, nil
}

// GetOwnSpInfo query own sp info in db
func (s *SpDBImpl) GetOwnSpInfo() (*sptypes.StorageProvider, error) {
	queryReturn := &SpInfoTable{}
	result := s.db.First(queryReturn, "is_own = true")
	if result.Error != nil {
		return nil, fmt.Errorf("failed to query own sp record in sp info table: %s", result.Error)
	}
	totalDeposit, ok := sdkmath.NewIntFromString(queryReturn.TotalDeposit)
	if !ok {
		return nil, fmt.Errorf("failed to parse int")
	}
	return &sptypes.StorageProvider{
		Id:              queryReturn.ID,
		OperatorAddress: queryReturn.OperatorAddress,
		FundingAddress:  queryReturn.FundingAddress,
		SealAddress:     queryReturn.SealAddress,
		ApprovalAddress: queryReturn.ApprovalAddress,
		TotalDeposit:    totalDeposit,
		Status:          sptypes.Status(queryReturn.Status),
		Endpoint:        queryReturn.Endpoint,
		Description: sptypes.Description{
			Moniker:         queryReturn.Moniker,
			Identity:        queryReturn.Identity,
			Website:         queryReturn.Website,
			SecurityContact: queryReturn.SecurityContact,
			Details:         queryReturn.Details,
		},
	}, nil
}

// SetOwnSpInfo set(maybe overwrite) own sp info to db
func (s *SpDBImpl) SetOwnSpInfo(sp *sptypes.StorageProvider) error {
	spInfo, err := s.GetOwnSpInfo()
	if err != nil && !strings.Contains(err.Error(), gorm.ErrRecordNotFound.Error()) {
		return err
	}

	insertRecord := &SpInfoTable{
		ID:              sp.GetId(),
		OperatorAddress: sp.GetOperatorAddress(),
		IsOwn:           true,
		FundingAddress:  sp.GetFundingAddress(),
		SealAddress:     sp.GetSealAddress(),
		ApprovalAddress: sp.GetApprovalAddress(),
		TotalDeposit:    sp.GetTotalDeposit().String(),
		Status:          int32(sp.GetStatus()),
		Endpoint:        sp.GetEndpoint(),
		Moniker:         sp.GetDescription().Moniker,
		Identity:        sp.GetDescription().Identity,
		Website:         sp.GetDescription().Website,
		SecurityContact: sp.GetDescription().SecurityContact,
		Details:         sp.GetDescription().Details,
	}
	// if there is no records in SPInfoTable, insert a new record
	if spInfo == nil {
		result := s.db.Create(insertRecord)
		if result.Error != nil || result.RowsAffected != 1 {
			return fmt.Errorf("failed to insert own sp record in sp info table: %s", result.Error)
		}
		return nil
	} else {
		// if there is a record in SPInfoTable, update record
		result := s.db.Model(&SpInfoTable{}).Where("is_own = true").Updates(insertRecord)
		if result.Error != nil {
			return fmt.Errorf("failed to update own sp record in sp info table: %s", result.Error)
		}
		return nil
	}
}
