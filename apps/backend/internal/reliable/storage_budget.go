package reliable

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"

	bolt "go.etcd.io/bbolt"
)

type StorageBudgetPolicy struct {
	Revision         int   `json:"revision"`
	CacheLimitBytes  int64 `json:"cache_limit_bytes"`
	TempLimitBytes   int64 `json:"temp_limit_bytes"`
	MinimumFreeBytes int64 `json:"minimum_free_bytes"`
}

type StorageBudgetEdit struct {
	ExpectedRevision int   `json:"expected_revision"`
	CacheLimitBytes  int64 `json:"cache_limit_bytes"`
	TempLimitBytes   int64 `json:"temp_limit_bytes"`
	MinimumFreeBytes int64 `json:"minimum_free_bytes"`
}

type StorageBudgetReport struct {
	StorageBudgetPolicy
	CacheBytes int64    `json:"cache_bytes"`
	TempBytes  int64    `json:"temp_bytes"`
	FreeBytes  uint64   `json:"free_bytes"`
	FreeKnown  bool     `json:"free_known"`
	Partial    bool     `json:"partial"`
	LimitMode  string   `json:"limit_mode"`
	Pressure   string   `json:"pressure"`
	Warnings   []string `json:"warnings"`
}

var storageBudgetBucket = []byte("reliable-storage-budget-v1")

func defaultStorageBudget() StorageBudgetPolicy {
	return StorageBudgetPolicy{CacheLimitBytes: 8 << 30, TempLimitBytes: 2 << 30, MinimumFreeBytes: 2 << 30}
}

func readStorageBudget(tx *bolt.Tx) (StorageBudgetPolicy, error) {
	policy := defaultStorageBudget()
	if tx == nil {
		return policy, nil
	}
	if bucket := tx.Bucket(storageBudgetBucket); bucket != nil {
		if raw := bucket.Get([]byte("policy")); raw != nil {
			if err := json.Unmarshal(raw, &policy); err != nil {
				return policy, err
			}
		}
	}
	return policy, nil
}

func (e *Engine) storageBudgetPolicy() (StorageBudgetPolicy, error) {
	policy := defaultStorageBudget()
	if e.Store == nil {
		return policy, nil
	}
	err := e.Store.db.View(func(tx *bolt.Tx) error {
		var err error
		policy, err = readStorageBudget(tx)
		return err
	})
	return policy, err
}

func (e *Engine) EditStorageBudget(ctx context.Context, edit StorageBudgetEdit) (*StorageBudgetReport, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := validateStorageBudget(edit); err != nil {
		return nil, err
	}
	if e.Store == nil {
		return nil, errors.New("storage_budget_unavailable: persistence is not available")
	}
	var updated StorageBudgetPolicy
	err := e.Store.db.Update(func(tx *bolt.Tx) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		policy, err := readStorageBudget(tx)
		if err != nil {
			return err
		}
		if policy.Revision != edit.ExpectedRevision {
			return errors.New("storage_budget_conflict: reload current limits before saving")
		}
		policy = StorageBudgetPolicy{Revision: policy.Revision + 1, CacheLimitBytes: edit.CacheLimitBytes, TempLimitBytes: edit.TempLimitBytes, MinimumFreeBytes: edit.MinimumFreeBytes}
		raw, err := json.Marshal(policy)
		if err != nil {
			return err
		}
		bucket, err := tx.CreateBucketIfNotExists(storageBudgetBucket)
		if err != nil {
			return err
		}
		updated = policy
		return bucket.Put([]byte("policy"), raw)
	})
	if err != nil {
		return nil, err
	}
	report, err := e.StorageBudget(ctx)
	if err != nil {
		return &StorageBudgetReport{StorageBudgetPolicy: updated, LimitMode: "advisory", Pressure: "unknown", Warnings: []string{"设置已保存，本次空间统计未完成，请稍后刷新。"}}, nil
	}
	return report, nil
}

func validateStorageBudget(edit StorageBudgetEdit) error {
	if edit.ExpectedRevision < 0 {
		return errors.New("storage_budget_invalid: invalid revision")
	}
	for _, amount := range []int64{edit.CacheLimitBytes, edit.TempLimitBytes, edit.MinimumFreeBytes} {
		if amount < 64<<20 || amount > 1<<40 {
			return errors.New("storage_budget_invalid: limits must be between 64 MiB and 1 TiB")
		}
	}
	return nil
}

func (e *Engine) StorageBudget(ctx context.Context) (*StorageBudgetReport, error) {
	policy, err := e.storageBudgetPolicy()
	if err != nil {
		return nil, err
	}
	root, err := openStorageRoot(e.Root)
	if err != nil {
		return nil, err
	}
	defer func() { _ = root.Close() }()
	report := &StorageBudgetReport{StorageBudgetPolicy: policy, LimitMode: "advisory", Warnings: []string{}}
	remaining := storageScanLimit
	for _, name := range []string{"cache", "temp"} {
		entry := StorageEntry{Name: name}
		partial, scanErr := scanStorageEntry(ctx, root, name, &entry, &remaining)
		if scanErr != nil {
			return nil, scanErr
		}
		report.Partial = report.Partial || partial
		if name == "cache" {
			report.CacheBytes = entry.Bytes
		} else {
			report.TempBytes = entry.Bytes
		}
	}
	report.FreeBytes, err = storageAvailableBytes(e.Root)
	report.FreeKnown = err == nil
	classifyStoragePressure(report)
	return report, nil
}

func classifyStoragePressure(report *StorageBudgetReport) {
	report.Pressure = "healthy"
	if report.CacheBytes > report.CacheLimitBytes {
		report.Warnings = append(report.Warnings, "缓存已超过提醒额度，请查看占用；不会自动删除证据或项目。")
	}
	if report.TempBytes > report.TempLimitBytes {
		report.Warnings = append(report.Warnings, "临时文件已超过提醒额度，可检查允许清理的旧临时文件。")
	}
	if len(report.Warnings) > 0 {
		report.Pressure = "warning"
	}
	if report.Partial || !report.FreeKnown {
		report.Warnings = append(report.Warnings, "空间统计未完整取得，请勿将当前数字视为完整占用。")
		if report.Pressure == "healthy" {
			report.Pressure = "unknown"
		}
	}
	if report.FreeKnown && report.FreeBytes < uint64(report.MinimumFreeBytes) {
		report.Pressure = "critical"
		report.Warnings = append(report.Warnings, "数据所在磁盘余量低于设置值，新的工作副本与预览暂缓分配。")
	}
}

// CheckStorageBudget probes disk space without scanning historical task data.
func (e *Engine) CheckStorageBudget(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	policy, err := e.storageBudgetPolicy()
	if err != nil {
		return err
	}
	free, err := storageAvailableBytes(e.Root)
	if err != nil {
		return fmt.Errorf("storage_space_unknown: data disk space could not be checked: %w", err)
	}
	if free < uint64(policy.MinimumFreeBytes) {
		return fmt.Errorf("storage_pressure: %s has %d bytes available; %d required before new allocation", filepath.VolumeName(e.Root), free, policy.MinimumFreeBytes)
	}
	return nil
}
