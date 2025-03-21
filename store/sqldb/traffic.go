package sqldb

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	corespdb "github.com/bnb-chain/greenfield-storage-provider/core/spdb"
	"github.com/bnb-chain/greenfield-storage-provider/pkg/log"
	"github.com/bnb-chain/greenfield-storage-provider/pkg/metrics"
)

const (
	// SPDBSuccessCheckQuotaAndAddReadRecord defines the metrics label of successfully check and add read record
	SPDBSuccessCheckQuotaAndAddReadRecord = "check_and_add_read_record_success"
	// SPDBFailureCheckQuotaAndAddReadRecord defines the metrics label of unsuccessfully check and add read record
	SPDBFailureCheckQuotaAndAddReadRecord = "check_and_add_read_record_failure"
	// SPDBSuccessGetBucketTraffic defines the metrics label of successfully get bucket traffic
	SPDBSuccessGetBucketTraffic = "get_bucket_traffic_success"
	// SPDBFailureGetBucketTraffic defines the metrics label of unsuccessfully get bucket traffic
	SPDBFailureGetBucketTraffic = "get_bucket_traffic_failure"
	// SPDBSuccessGetReadRecord defines the metrics label of successfully get read record
	SPDBSuccessGetReadRecord = "get_read_record_success"
	// SPDBFailureGetReadRecord defines the metrics label of unsuccessfully get read record
	SPDBFailureGetReadRecord = "get_read_record_failure"
	// SPDBSuccessGetBucketReadRecord defines the metrics label of successfully get bucket read record
	SPDBSuccessGetBucketReadRecord = "get_bucket_read_record_success"
	// SPDBFailureGetBucketReadRecord defines the metrics label of unsuccessfully get bucket read record
	SPDBFailureGetBucketReadRecord = "get_bucket_read_record_failure"
	// SPDBSuccessGetObjectReadRecord defines the metrics label of successfully get object read record
	SPDBSuccessGetObjectReadRecord = "get_object_read_record_success"
	// SPDBFailureGetObjectReadRecord defines the metrics label of unsuccessfully get object read record
	SPDBFailureGetObjectReadRecord = "get_object_read_record_failure"
	// SPDBSuccessGetUserReadRecord defines the metrics label of successfully get user read record
	SPDBSuccessGetUserReadRecord = "get_user_read_record_success"
	// SPDBFailureGetUserReadRecord defines the metrics label of unsuccessfully get user read record
	SPDBFailureGetUserReadRecord = "get_user_read_record_failure"
)

const metaDeleteLimit = 100

// CheckQuotaAndAddReadRecord check current quota, and add read record
func (s *SpDBImpl) CheckQuotaAndAddReadRecord(record *corespdb.ReadRecord, quota *corespdb.BucketQuota) (err error) {
	startTime := time.Now()
	defer func() {
		if err != nil {
			metrics.SPDBCounter.WithLabelValues(SPDBFailureCheckQuotaAndAddReadRecord).Inc()
			metrics.SPDBTime.WithLabelValues(SPDBFailureCheckQuotaAndAddReadRecord).Observe(
				time.Since(startTime).Seconds())
			return
		}
		metrics.SPDBCounter.WithLabelValues(SPDBSuccessCheckQuotaAndAddReadRecord).Inc()
		metrics.SPDBTime.WithLabelValues(SPDBSuccessCheckQuotaAndAddReadRecord).Observe(
			time.Since(startTime).Seconds())
	}()

	err = s.updateConsumedQuota(record, quota)
	if err != nil {
		log.Errorw("failed to commit the transaction of updating bucketTraffic table, ", "error", err)
		return err
	}

	// add read record
	insertReadRecord := &ReadRecordTable{
		BucketID:        record.BucketID,
		ObjectID:        record.ObjectID,
		UserAddress:     record.UserAddress,
		ReadTimestampUs: record.ReadTimestampUs,
		BucketName:      record.BucketName,
		ObjectName:      record.ObjectName,
		ReadSize:        record.ReadSize,
	}
	result := s.db.Create(insertReadRecord)
	if result.Error != nil || result.RowsAffected != 1 {
		err = fmt.Errorf("failed to insert read record table: %s", result.Error)
		return err
	}
	return nil
}

// getUpdatedConsumedQuotaV2 compute the updated quota of traffic table by the incoming read cost and the newest record.
// it returns the updated consumed free quota,consumed charged quota and remained free quota
func getUpdatedConsumedQuotaV2(recordQuotaCost, freeQuotaRemain, consumeFreeQuota, consumeChargedQuota, chargedQuota, monthlyFreeQuotaRemain, consumeMonthlyFreeQuota uint64) (uint64, uint64, uint64, uint64, uint64, error) {
	log.Infow("quota info", "freeQuotaRemain", freeQuotaRemain, "consumeFreeQuota", consumeFreeQuota, "consumeChargedQuota", consumeChargedQuota, "chargedQuota", chargedQuota, "monthlyFreeQuotaRemain", monthlyFreeQuotaRemain, "consumeMonthlyFreeQuota", consumeMonthlyFreeQuota)
	defer log.Infow("quota info", "freeQuotaRemain", freeQuotaRemain, "consumeFreeQuota", consumeFreeQuota, "consumeChargedQuota", consumeChargedQuota, "chargedQuota", chargedQuota, "monthlyFreeQuotaRemain", monthlyFreeQuotaRemain, "consumeMonthlyFreeQuota", consumeMonthlyFreeQuota)
	chargedQuotaInt := int64(chargedQuota) - int64(consumeChargedQuota)
	if chargedQuotaInt >= int64(recordQuotaCost) {
		consumeChargedQuota += recordQuotaCost
		return consumeFreeQuota, consumeChargedQuota, consumeMonthlyFreeQuota, freeQuotaRemain, monthlyFreeQuotaRemain, nil
	}
	if chargedQuotaInt > 0 {
		consumeChargedQuota += uint64(chargedQuotaInt)
		recordQuotaCost -= uint64(chargedQuotaInt)
	}

	if monthlyFreeQuotaRemain >= recordQuotaCost {
		consumeMonthlyFreeQuota += recordQuotaCost
		monthlyFreeQuotaRemain -= recordQuotaCost
		return consumeFreeQuota, consumeChargedQuota, consumeMonthlyFreeQuota, freeQuotaRemain, monthlyFreeQuotaRemain, nil
	}

	if monthlyFreeQuotaRemain > 0 {
		consumeMonthlyFreeQuota += monthlyFreeQuotaRemain
		recordQuotaCost -= monthlyFreeQuotaRemain
		monthlyFreeQuotaRemain = 0
	}

	if freeQuotaRemain >= recordQuotaCost {
		freeQuotaRemain -= recordQuotaCost
		consumeFreeQuota += recordQuotaCost
		return consumeFreeQuota, consumeChargedQuota, consumeMonthlyFreeQuota, freeQuotaRemain, monthlyFreeQuotaRemain, nil
	}

	return 0, 0, 0, 0, 0, ErrCheckQuotaEnough
}

// updateConsumedQuota update the consumed quota of BucketTraffic table in the transaction way
func (s *SpDBImpl) updateConsumedQuota(record *corespdb.ReadRecord, quota *corespdb.BucketQuota) error {
	yearMonth := TimeToYearMonth(TimestampUsToTime(record.ReadTimestampUs))
	err := s.db.Transaction(func(tx *gorm.DB) error {
		var bucketTraffic BucketTrafficTable
		var err error
		if err = tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("bucket_id = ? and month = ?", record.BucketID, yearMonth).First(&bucketTraffic).Error; err != nil {
			return fmt.Errorf("failed to query bucket traffic table: %v", err)
		}

		// if charged quota changed, update the new value
		if bucketTraffic.ChargedQuotaSize != quota.ChargedQuotaSize {
			result := tx.Model(&bucketTraffic).
				Updates(BucketTrafficTable{
					ChargedQuotaSize: quota.ChargedQuotaSize,
					ModifiedTime:     time.Now(),
				})

			if result.Error != nil {
				return fmt.Errorf("failed to update bucket traffic table: %s", result.Error)
			}
			if result.RowsAffected != 1 {
				return fmt.Errorf("update traffic of %s has affected more than one rows %d, "+
					"update charged quota %d", bucketTraffic.BucketName, result.RowsAffected, quota.ChargedQuotaSize)
			}
			log.CtxDebugw(context.Background(), "updated quota", "charged quota", quota.ChargedQuotaSize)
		}

		// compute the new consumed quota size to be updated by the newest record and the read cost size
		updatedConsumedFreeQuota, updatedConsumedChargedQuota, updatedConsumedMonthlyFreeQuota, updatedRemainedFreeQuota, updatedRemainedMonthlyFreeQuota, err := getUpdatedConsumedQuotaV2(record.ReadSize,
			bucketTraffic.FreeQuotaSize, bucketTraffic.FreeQuotaConsumedSize,
			bucketTraffic.ReadConsumedSize, quota.ChargedQuotaSize, bucketTraffic.MonthlyQuotaSize, bucketTraffic.MonthlyFreeQuotaConsumedSize)
		if err != nil {
			return err
		}

		// it is needed to add select items if you need to update a value to zero in gorm db
		err = tx.Model(&bucketTraffic).
			Select("read_consumed_size", "free_quota_consumed_size", "free_quota_size", "monthly_free_quota_consumed_size", "monthly_quota_size", "modified_time").Updates(BucketTrafficTable{
			ReadConsumedSize:             updatedConsumedChargedQuota,
			FreeQuotaConsumedSize:        updatedConsumedFreeQuota,
			FreeQuotaSize:                updatedRemainedFreeQuota,
			MonthlyFreeQuotaConsumedSize: updatedConsumedMonthlyFreeQuota,
			MonthlyQuotaSize:             updatedRemainedMonthlyFreeQuota,
			ModifiedTime:                 time.Now(),
		}).Error
		if err != nil {
			return fmt.Errorf("failed to update bucket traffic table: %v", err)
		}

		return nil
	})

	if err != nil {
		log.CtxErrorw(context.Background(), "updated quota transaction fail", "error", err)
	}
	return err
}

// InitBucketTraffic init the bucket traffic table
func (s *SpDBImpl) InitBucketTraffic(record *corespdb.ReadRecord, quota *corespdb.BucketQuota) error {
	bucketID := record.BucketID
	bucketName := record.BucketName
	yearMonth := TimestampYearMonth(record.ReadTimestampUs)
	// if not created, init the bucket id in transaction
	err := s.db.Transaction(func(tx *gorm.DB) error {
		var insertBucketTraffic *BucketTrafficTable
		var bucketTraffic BucketTrafficTable
		result := s.db.Where("bucket_id = ?", bucketID).First(&bucketTraffic)
		if result.Error != nil {
			if !errors.Is(result.Error, gorm.ErrRecordNotFound) {
				return result.Error
			} else {
				// If the record of this bucket id does not exist, then the free quota consumed is initialized to 0
				insertBucketTraffic = &BucketTrafficTable{
					BucketID:                     bucketID,
					Month:                        yearMonth,
					FreeQuotaSize:                quota.FreeQuotaSize,
					FreeQuotaConsumedSize:        0,
					BucketName:                   bucketName,
					ReadConsumedSize:             0,
					ChargedQuotaSize:             quota.ChargedQuotaSize,
					MonthlyQuotaSize:             quota.MonthlyFreeQuotaSize,
					MonthlyFreeQuotaConsumedSize: 0,
					ModifiedTime:                 time.Now(),
				}
			}
		} else {
			// If the record of this bucket id already exist, then read the record of the newest month
			// and use the free quota consumed of this record to init free quota item
			var newestTraffic BucketTrafficTable
			queryErr := s.db.Where("bucket_id = ?", bucketID).Order("month DESC").Limit(1).Find(&newestTraffic).Error
			if queryErr != nil {
				return queryErr
			}

			insertBucketTraffic = &BucketTrafficTable{
				BucketID:                     bucketID,
				Month:                        yearMonth,
				FreeQuotaSize:                newestTraffic.FreeQuotaSize,
				FreeQuotaConsumedSize:        0,
				BucketName:                   bucketName,
				ReadConsumedSize:             0,
				ChargedQuotaSize:             quota.ChargedQuotaSize,
				MonthlyQuotaSize:             quota.MonthlyFreeQuotaSize,
				MonthlyFreeQuotaConsumedSize: 0,
				ModifiedTime:                 time.Now(),
			}
		}
		result = tx.Create(insertBucketTraffic)
		if result.Error != nil && MysqlErrCode(result.Error) != ErrDuplicateEntryCode {
			return fmt.Errorf("failed to create bucket traffic table: %s", result.Error)
		}

		return nil
	})

	if err != nil {
		log.CtxErrorw(context.Background(), "init traffic table error ", "bucket name", bucketName, "error", err)
	}
	return err
}

// GetBucketTraffic return bucket traffic info by the year and month info
// year_month is the query bucket quota's month, like "2023-03"
func (s *SpDBImpl) GetBucketTraffic(bucketID uint64, yearMonth string) (traffic *corespdb.BucketTraffic, err error) {
	var (
		result      *gorm.DB
		queryReturn BucketTrafficTable
	)

	startTime := time.Now()
	defer func() {
		if err != nil {
			metrics.SPDBCounter.WithLabelValues(SPDBFailureGetBucketTraffic).Inc()
			metrics.SPDBTime.WithLabelValues(SPDBFailureGetBucketTraffic).Observe(
				time.Since(startTime).Seconds())
			return
		}
		metrics.SPDBCounter.WithLabelValues(SPDBSuccessGetBucketTraffic).Inc()
		metrics.SPDBTime.WithLabelValues(SPDBSuccessGetBucketTraffic).Observe(
			time.Since(startTime).Seconds())
	}()

	result = s.db.Where("bucket_id = ? and month = ?", bucketID, yearMonth).First(&queryReturn)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		err = result.Error
		return nil, err
	}
	if result.Error != nil {
		err = fmt.Errorf("failed to query bucket traffic table: %s", result.Error)
		return nil, err
	}

	return &corespdb.BucketTraffic{
		BucketID:                     queryReturn.BucketID,
		YearMonth:                    queryReturn.Month,
		FreeQuotaSize:                queryReturn.FreeQuotaSize,
		FreeQuotaConsumedSize:        queryReturn.FreeQuotaConsumedSize,
		BucketName:                   queryReturn.BucketName,
		ReadConsumedSize:             queryReturn.ReadConsumedSize,
		ChargedQuotaSize:             queryReturn.ChargedQuotaSize,
		MonthlyFreeQuotaConsumedSize: queryReturn.MonthlyFreeQuotaConsumedSize,
		MonthlyFreeQuotaSize:         queryReturn.MonthlyQuotaSize,
		ModifyTime:                   queryReturn.ModifiedTime.Unix(),
	}, nil
}

// ListBucketTraffic return bucket traffic info by the year and month info
// year_month is the query bucket quota's month, like "2023-03"
// Support paging query
func (s *SpDBImpl) ListBucketTraffic(yearMonth string, offset, limit int) (traffics []*corespdb.BucketTraffic, err error) {
	var (
		dbResult *gorm.DB
		result   []BucketTrafficTable
	)

	startTime := time.Now()
	defer func() {
		if err != nil {
			metrics.SPDBCounter.WithLabelValues(SPDBFailureGetBucketTraffic).Inc()
			metrics.SPDBTime.WithLabelValues(SPDBFailureGetBucketTraffic).Observe(
				time.Since(startTime).Seconds())
			return
		}
		metrics.SPDBCounter.WithLabelValues(SPDBSuccessGetBucketTraffic).Inc()
		metrics.SPDBTime.WithLabelValues(SPDBSuccessGetBucketTraffic).Observe(
			time.Since(startTime).Seconds())
	}()

	dbResult = s.db.Where("month = ?", yearMonth).Offset(offset).Limit(limit).Find(&result)
	if errors.Is(dbResult.Error, gorm.ErrRecordNotFound) {
		err = dbResult.Error
		return nil, err
	}
	if dbResult.Error != nil {
		err = fmt.Errorf("failed to query bucket traffic table: %s", dbResult.Error)
		return nil, err
	}

	for _, res := range result {
		traffics = append(traffics, &corespdb.BucketTraffic{
			BucketID:                     res.BucketID,
			YearMonth:                    res.Month,
			FreeQuotaSize:                res.FreeQuotaSize,
			FreeQuotaConsumedSize:        res.FreeQuotaConsumedSize,
			BucketName:                   res.BucketName,
			ReadConsumedSize:             res.ReadConsumedSize,
			ChargedQuotaSize:             res.ChargedQuotaSize,
			MonthlyFreeQuotaConsumedSize: res.MonthlyFreeQuotaConsumedSize,
			MonthlyFreeQuotaSize:         res.MonthlyQuotaSize,
			ModifyTime:                   res.ModifiedTime.Unix(),
		})
	}
	return
}

func (s *SpDBImpl) GetBucketTrafficCount(yearMonth string) (count int64, err error) {
	var (
		dbResult *gorm.DB
	)

	startTime := time.Now()
	defer func() {
		if err != nil {
			metrics.SPDBCounter.WithLabelValues(SPDBFailureGetBucketTraffic).Inc()
			metrics.SPDBTime.WithLabelValues(SPDBFailureGetBucketTraffic).Observe(
				time.Since(startTime).Seconds())
			return
		}
		metrics.SPDBCounter.WithLabelValues(SPDBSuccessGetBucketTraffic).Inc()
		metrics.SPDBTime.WithLabelValues(SPDBSuccessGetBucketTraffic).Observe(
			time.Since(startTime).Seconds())
	}()

	dbResult = s.db.Model(&BucketTrafficTable{}).Where("month = ?", yearMonth).Count(&count)
	if dbResult.Error != nil {
		err = fmt.Errorf("failed to query bucket traffic table: %s", dbResult.Error)
		return 0, err
	}
	return
}

// UpdateExtraQuota update the read consumed quota and free consumed quota in traffic db with the extra quota
func (s *SpDBImpl) UpdateExtraQuota(bucketID, extraQuota uint64, yearMonth string) error {
	log.CtxErrorw(context.Background(), "begin to update extra quota for traffic db", "extra quota", extraQuota)
	err := s.db.Transaction(func(tx *gorm.DB) error {
		var bucketTraffic BucketTrafficTable
		var err error
		yearMonthOfNow := TimestampYearMonth(GetCurrentTimestampUs())
		var extraUpdateOnNextMonth bool

		// In most cases, the month of extra quota compensation should be the same as the current month,
		// if not, the current month is the next month of the compensation month
		if IsNextMonth(yearMonthOfNow, yearMonth) {
			err = tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("bucket_id = ? and month = ?", bucketID, yearMonthOfNow).First(&bucketTraffic).Error
			if err != nil {
				// if it is a new month but the record has not been inserted, just update the extra quota to the old month
				// the new month will init by the newest old month record.
				if errors.Is(err, gorm.ErrRecordNotFound) {
					extraUpdateOnNextMonth = false
				}
			} else {
				extraUpdateOnNextMonth = true
			}
		} else {
			if yearMonthOfNow != yearMonth {
				return fmt.Errorf("the month of record of traffic table is invalid %s", yearMonth)
			}
			if err = tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("bucket_id = ? and month = ?", bucketID, yearMonth).First(&bucketTraffic).Error; err != nil {
				return fmt.Errorf("failed to query bucket traffic table: %v", err)
			}
		}

		consumedFreeQuota := bucketTraffic.FreeQuotaConsumedSize
		remainedFreeQuota := bucketTraffic.FreeQuotaSize
		log.CtxDebugw(context.Background(), "updated read consumed quota with extra quota:", "month", yearMonth, "consumed:", bucketTraffic.ReadConsumedSize,
			"remained free quota", remainedFreeQuota, "extra", extraQuota)

		if extraUpdateOnNextMonth {
			// if the extra quota generate on the different month, needed to add the extra quota to free quota
			// for example, the extra quota was generated at the last second on August 31, and the action of replenishing the quota began on September 1.
			// At this time, the latest quota record of traffic table is in September instead of August.
			// This situation is very unlikely to happen, in this case, the quota will be restored to the free quota.
			err = tx.Model(&bucketTraffic).
				Updates(BucketTrafficTable{
					FreeQuotaSize: bucketTraffic.FreeQuotaSize + extraQuota,
					ModifiedTime:  time.Now(),
				}).Error
		} else {
			consumedChargeQuota := bucketTraffic.ReadConsumedSize
			consumedMonthlyFreeQuota := bucketTraffic.MonthlyFreeQuotaConsumedSize
			monthlyFreeQuotaRemain := bucketTraffic.MonthlyQuotaSize
			log.Infow("quota info", "consumedFreeQuota", consumedFreeQuota, "remainedFreeQuota", remainedFreeQuota, "consumedChargeQuota", consumedChargeQuota, "consumedMonthlyFreeQuota", consumedMonthlyFreeQuota, "monthlyFreeQuotaRemain", monthlyFreeQuotaRemain)
			// The priority of compensation is chargeQuota > monthlyFreeQuota > freeQuota
			// ChargeQuota
			if consumedChargeQuota >= extraQuota {
				consumedChargeQuota -= extraQuota
				extraQuota = 0
			} else if consumedChargeQuota > 0 {
				extraQuota -= consumedChargeQuota
				consumedChargeQuota = 0
			}
			// MonthlyFreeQuota
			if extraQuota > 0 && consumedMonthlyFreeQuota >= extraQuota {
				consumedMonthlyFreeQuota -= extraQuota
				monthlyFreeQuotaRemain += extraQuota
				extraQuota = 0
			} else if extraQuota > 0 && consumedMonthlyFreeQuota > 0 {
				extraQuota -= consumedMonthlyFreeQuota
				monthlyFreeQuotaRemain += consumedMonthlyFreeQuota
				consumedMonthlyFreeQuota = 0
			}
			// FreeQuota
			if extraQuota > 0 {
				consumedFreeQuota -= extraQuota
				remainedFreeQuota += extraQuota
			}
			log.Infow("quota info", "consumedFreeQuota", consumedFreeQuota, "remainedFreeQuota", remainedFreeQuota, "consumedChargeQuota", consumedChargeQuota, "consumedMonthlyFreeQuota", consumedMonthlyFreeQuota, "monthlyFreeQuotaRemain", monthlyFreeQuotaRemain)
			err = tx.Model(&bucketTraffic).
				Select("read_consumed_size", "free_quota_consumed_size", "monthly_free_quota_consumed_size", "free_quota_size", "monthly_quota_size", "modified_time").Updates(BucketTrafficTable{
				ReadConsumedSize:             consumedChargeQuota,
				FreeQuotaConsumedSize:        consumedFreeQuota,
				MonthlyFreeQuotaConsumedSize: consumedMonthlyFreeQuota,
				FreeQuotaSize:                remainedFreeQuota,
				MonthlyQuotaSize:             monthlyFreeQuotaRemain,
				ModifiedTime:                 time.Now(),
			}).Error
		}

		if err != nil {
			return fmt.Errorf("failed to update bucket traffic table: %v", err)
		}

		return nil
	})

	if err != nil {
		log.CtxErrorw(context.Background(), "failed to update the table by extra quota ", "bucket id", bucketID, "error", err)
	}

	return err
}

// GetLatestBucketTraffic return the latest bucket traffic info of the bucket
func (s *SpDBImpl) GetLatestBucketTraffic(bucketID uint64) (traffic *corespdb.BucketTraffic, err error) {
	var (
		result      *gorm.DB
		queryReturn BucketTrafficTable
	)

	startTime := time.Now()
	defer func() {
		if err != nil {
			metrics.SPDBCounter.WithLabelValues(SPDBFailureGetBucketTraffic).Inc()
			metrics.SPDBTime.WithLabelValues(SPDBFailureGetBucketTraffic).Observe(
				time.Since(startTime).Seconds())
			return
		}
		metrics.SPDBCounter.WithLabelValues(SPDBSuccessGetBucketTraffic).Inc()
		metrics.SPDBTime.WithLabelValues(SPDBSuccessGetBucketTraffic).Observe(
			time.Since(startTime).Seconds())
	}()

	result = s.db.Where("bucket_id = ?", bucketID).Order("month DESC").Limit(1).First(&queryReturn)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		err = result.Error
		return nil, err
	}
	if result.Error != nil {
		err = fmt.Errorf("failed to query bucket traffic table: %s", result.Error)
		return nil, err
	}

	return &corespdb.BucketTraffic{
		BucketID:              queryReturn.BucketID,
		YearMonth:             queryReturn.Month,
		FreeQuotaSize:         queryReturn.FreeQuotaSize,
		FreeQuotaConsumedSize: queryReturn.FreeQuotaConsumedSize,
		BucketName:            queryReturn.BucketName,
		ReadConsumedSize:      queryReturn.ReadConsumedSize,
		ChargedQuotaSize:      queryReturn.ChargedQuotaSize,
		ModifyTime:            queryReturn.ModifiedTime.Unix(),
	}, nil
}

// UpdateBucketTraffic update the bucket traffic in traffic db with the new traffic
func (s *SpDBImpl) UpdateBucketTraffic(bucketID uint64, update *corespdb.BucketTraffic) (err error) {
	var (
		result      *gorm.DB
		queryReturn BucketTrafficTable
		needInsert  = false
	)

	result = s.db.Where("bucket_id = ? and month = ?", bucketID, update.YearMonth).First(&queryReturn)

	if result.Error != nil && !errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return result.Error
	}
	if result.Error != nil {
		needInsert = errors.Is(result.Error, gorm.ErrRecordNotFound)
	}

	updateRecord := &BucketTrafficTable{
		BucketID:              update.BucketID,
		Month:                 update.YearMonth,
		FreeQuotaSize:         update.FreeQuotaSize,
		FreeQuotaConsumedSize: update.FreeQuotaConsumedSize,
		BucketName:            update.BucketName,
		ReadConsumedSize:      update.ReadConsumedSize,
		ChargedQuotaSize:      update.ChargedQuotaSize,
		ModifiedTime:          time.Now(),
	}

	if needInsert {
		result = s.db.Create(updateRecord)
		if result.Error != nil || result.RowsAffected != 1 {
			return fmt.Errorf("failed to insert record in bucket traffic table: %s", result.Error)
		}
	} else { // update
		result = s.db.Model(&BucketTrafficTable{}).
			Where("bucket_id = ? and month = ?", bucketID, update.YearMonth).Updates(updateRecord)
		if result.Error != nil {
			return fmt.Errorf("failed to update record in bucket traffic table: %s", result.Error)
		}
	}
	return nil
}

// DeleteExpiredBucketTraffic update the bucket traffic in traffic db with the new traffic
func (s *SpDBImpl) DeleteExpiredBucketTraffic(yearMonth string) (err error) {
	var bucketTraffic []BucketTrafficTable
	result := s.db.Where("month < ?", yearMonth).Limit(metaDeleteLimit).Find(&bucketTraffic).Delete(&bucketTraffic)
	if result.Error != nil {
		return fmt.Errorf("failed to delete bucket traffic record in bucket traffic table: %s, year_month:%s", result.Error, yearMonth)
	}
	return nil
}

// GetReadRecord return record list by time range
func (s *SpDBImpl) GetReadRecord(timeRange *corespdb.TrafficTimeRange) (records []*corespdb.ReadRecord, err error) {
	var (
		result       *gorm.DB
		queryReturns []ReadRecordTable
	)
	startTime := time.Now()
	defer func() {
		if err != nil {
			metrics.SPDBCounter.WithLabelValues(SPDBFailureGetReadRecord).Inc()
			metrics.SPDBTime.WithLabelValues(SPDBFailureGetReadRecord).Observe(
				time.Since(startTime).Seconds())
			return
		}
		metrics.SPDBCounter.WithLabelValues(SPDBSuccessGetReadRecord).Inc()
		metrics.SPDBTime.WithLabelValues(SPDBSuccessGetReadRecord).Observe(
			time.Since(startTime).Seconds())
	}()

	if timeRange.LimitNum <= 0 {
		result = s.db.Where("read_timestamp_us >= ? and read_timestamp_us < ?", timeRange.StartTimestampUs, timeRange.EndTimestampUs).
			Find(&queryReturns)
	} else {
		result = s.db.Where("read_timestamp_us >= ? and read_timestamp_us < ?", timeRange.StartTimestampUs, timeRange.EndTimestampUs).
			Limit(timeRange.LimitNum).Find(&queryReturns)
	}
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		err = result.Error
		return nil, err
	}
	if result.Error != nil {
		err = fmt.Errorf("failed to read record table: %s", result.Error)
		return records, err
	}
	for _, record := range queryReturns {
		records = append(records, &corespdb.ReadRecord{
			BucketID:        record.BucketID,
			ObjectID:        record.ObjectID,
			UserAddress:     record.UserAddress,
			BucketName:      record.BucketName,
			ObjectName:      record.ObjectName,
			ReadSize:        record.ReadSize,
			ReadTimestampUs: record.ReadTimestampUs,
		})
	}
	return records, nil
}

// GetBucketReadRecord return bucket record list by time range
func (s *SpDBImpl) GetBucketReadRecord(bucketID uint64, timeRange *corespdb.TrafficTimeRange) (records []*corespdb.ReadRecord, err error) {
	var (
		result       *gorm.DB
		queryReturns []ReadRecordTable
	)
	startTime := time.Now()
	defer func() {
		if err != nil {
			metrics.SPDBCounter.WithLabelValues(SPDBFailureGetBucketReadRecord).Inc()
			return
		}
		metrics.SPDBCounter.WithLabelValues(SPDBSuccessGetBucketReadRecord).Inc()
		metrics.SPDBTime.WithLabelValues(SPDBSuccessGetBucketReadRecord).Observe(
			time.Since(startTime).Seconds())
	}()

	if timeRange.LimitNum <= 0 {
		result = s.db.Where("read_timestamp_us >= ? and read_timestamp_us < ? and bucket_id = ?",
			timeRange.StartTimestampUs, timeRange.EndTimestampUs, bucketID).
			Find(&queryReturns)
	} else {
		result = s.db.Where("read_timestamp_us >= ? and read_timestamp_us < ? and bucket_id = ?",
			timeRange.StartTimestampUs, timeRange.EndTimestampUs, bucketID).
			Limit(timeRange.LimitNum).Find(&queryReturns)
	}
	if result.Error != nil {
		err = fmt.Errorf("failed to query read record table: %s", result.Error)
		return records, err
	}
	for _, record := range queryReturns {
		records = append(records, &corespdb.ReadRecord{
			BucketID:        record.BucketID,
			ObjectID:        record.ObjectID,
			UserAddress:     record.UserAddress,
			BucketName:      record.BucketName,
			ObjectName:      record.ObjectName,
			ReadSize:        record.ReadSize,
			ReadTimestampUs: record.ReadTimestampUs,
		})
	}
	return records, nil
}

// GetObjectReadRecord return object record list by time range
func (s *SpDBImpl) GetObjectReadRecord(objectID uint64, timeRange *corespdb.TrafficTimeRange) (records []*corespdb.ReadRecord, err error) {
	var (
		result       *gorm.DB
		queryReturns []ReadRecordTable
	)
	startTime := time.Now()
	defer func() {
		if err != nil {
			metrics.SPDBCounter.WithLabelValues(SPDBFailureGetObjectReadRecord).Inc()
			return
		}
		metrics.SPDBCounter.WithLabelValues(SPDBSuccessGetObjectReadRecord).Inc()
		metrics.SPDBTime.WithLabelValues(SPDBSuccessGetObjectReadRecord).Observe(
			time.Since(startTime).Seconds())
	}()

	if timeRange.LimitNum <= 0 {
		result = s.db.Where("read_timestamp_us >= ? and read_timestamp_us < ? and object_id = ?",
			timeRange.StartTimestampUs, timeRange.EndTimestampUs, objectID).
			Find(&queryReturns)
	} else {
		result = s.db.Where("read_timestamp_us >= ? and read_timestamp_us < ? and object_id = ?",
			timeRange.StartTimestampUs, timeRange.EndTimestampUs, objectID).
			Limit(timeRange.LimitNum).Find(&queryReturns)
	}
	if result.Error != nil {
		err = fmt.Errorf("failed to query read record table: %s", result.Error)
		return records, err
	}
	for _, record := range queryReturns {
		records = append(records, &corespdb.ReadRecord{
			BucketID:        record.BucketID,
			ObjectID:        record.ObjectID,
			UserAddress:     record.UserAddress,
			BucketName:      record.BucketName,
			ObjectName:      record.ObjectName,
			ReadSize:        record.ReadSize,
			ReadTimestampUs: record.ReadTimestampUs,
		})
	}
	return records, nil
}

// GetUserReadRecord return user record list by time range
func (s *SpDBImpl) GetUserReadRecord(userAddress string, timeRange *corespdb.TrafficTimeRange) (records []*corespdb.ReadRecord, err error) {
	var (
		result       *gorm.DB
		queryReturns []ReadRecordTable
	)
	startTime := time.Now()
	defer func() {
		if err != nil {
			metrics.SPDBCounter.WithLabelValues(SPDBFailureGetUserReadRecord).Inc()
			return
		}
		metrics.SPDBCounter.WithLabelValues(SPDBSuccessGetUserReadRecord).Inc()
		metrics.SPDBTime.WithLabelValues(SPDBSuccessGetUserReadRecord).Observe(
			time.Since(startTime).Seconds())
	}()

	if timeRange.LimitNum <= 0 {
		result = s.db.Where("read_timestamp_us >= ? and read_timestamp_us < ? and user_address = ?",
			timeRange.StartTimestampUs, timeRange.EndTimestampUs, userAddress).
			Find(&queryReturns)
	} else {
		result = s.db.Where("read_timestamp_us >= ? and read_timestamp_us < ? and user_address = ?",
			timeRange.StartTimestampUs, timeRange.EndTimestampUs, userAddress).
			Limit(timeRange.LimitNum).Find(&queryReturns)
	}
	if result.Error != nil {
		err = fmt.Errorf("failed to query read record table: %s", result.Error)
		return records, err
	}
	for _, record := range queryReturns {
		records = append(records, &corespdb.ReadRecord{
			BucketID:        record.BucketID,
			ObjectID:        record.ObjectID,
			UserAddress:     record.UserAddress,
			BucketName:      record.BucketName,
			ObjectName:      record.ObjectName,
			ReadSize:        record.ReadSize,
			ReadTimestampUs: record.ReadTimestampUs,
		})
	}
	return records, nil
}

// DeleteExpiredReadRecord delete all read record before ts(ts is UnixMicro)
func (s *SpDBImpl) DeleteExpiredReadRecord(ts, limit uint64) (err error) {
	var readRecords []ReadRecordTable
	result := s.db.Where("read_timestamp_us < ?", ts).Limit(int(limit)).Find(&readRecords).Delete(&readRecords)
	if result.Error != nil {
		return fmt.Errorf("failed to delete read record in read record table: %s, ts:%d", result.Error, ts)
	}
	return nil
}
