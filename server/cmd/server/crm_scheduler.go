package main

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/multica-ai/multica/server/internal/handler"
)

func runCRMSchedulers(ctx context.Context, pool *pgxpool.Pool) {
	go handler.NewCRMIMAPAutoSyncScheduler(
		pool,
		envDuration("CRM_IMAP_AUTO_SYNC_INTERVAL", 5*time.Minute),
		envPositiveInt("CRM_IMAP_AUTO_SYNC_LIMIT", 100),
	).Run(ctx)
	go handler.NewCRMAIAutoScheduler(
		pool,
		envDuration("CRM_AI_AUTO_INTERVAL", 1*time.Minute),
	).Run(ctx)
}
