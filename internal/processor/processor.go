// internal/processor/processor.go
package processor

import (
	"context"
	"fmt"
	"github.com/bytedance/sonic"
	"log/slog"
	"sync"

	"tx-processor/internal/models"
)

type Processor struct {
	logger     *slog.Logger
	mu         sync.Mutex
	workerMaps map[int]map[string]*models.UserAnalytics
}

func New(logger *slog.Logger) *Processor {
	return &Processor{
		logger:     logger,
		workerMaps: make(map[int]map[string]*models.UserAnalytics),
	}
}

func (p *Processor) ProcessStream(ctx context.Context, id int, lines <-chan string, batchSize int) error {
	p.mu.Lock()
	p.workerMaps[id] = make(map[string]*models.UserAnalytics)
	p.mu.Unlock()

	var batch []models.Transaction

	for line := range lines {
		var tx models.Transaction
		if err := sonic.UnmarshalString(line, &tx); err != nil {
			p.logger.Warn("skipping invalid JSON", "error", err)
			continue
		}

		batch = append(batch, tx)

		if len(batch) >= batchSize {
			if err := p.applyTransactions(ctx, id, batch); err != nil {
				return fmt.Errorf("processing batch: %w", err)
			}
			batch = batch[:0]
		}
	}

	if len(batch) > 0 {
		return p.applyTransactions(ctx, id, batch)
	}

	return nil
}

func (p *Processor) applyTransactions(ctx context.Context, workerID int, txs []models.Transaction) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	localMap := p.workerMaps[workerID]
	seen := make(map[string]struct{}, len(txs))

	for _, tx := range txs {
		ua, ok := localMap[tx.UserID]
		if !ok {
			ua = &models.UserAnalytics{UserID: tx.UserID}
			localMap[tx.UserID] = ua
		}
		ua.TotalOrders++
		ua.TotalSpent += tx.Price * float64(tx.Quantity)
		seen[tx.UserID] = struct{}{}
	}

	p.logger.Info("batch processed",
		"transactions", len(txs),
		"users_affected", len(seen))

	return nil
}

func (p *Processor) Snapshot() map[string]*models.UserAnalytics {
	merged := make(map[string]*models.UserAnalytics)
	for _, workerMap := range p.workerMaps {
		for id, ua := range workerMap {
			if existing, ok := merged[id]; ok {
				existing.TotalOrders += ua.TotalOrders
				existing.TotalSpent += ua.TotalSpent
			} else {
				merged[id] = ua
			}
		}
	}
	return merged
}
