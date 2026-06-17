package main

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/maverick0322/taskify/backend/internal/core/ports"
)

const avatarStorageWorkerInterval = time.Minute

type avatarStorageJob struct {
	ID        string
	UserID    string
	LocalPath string
	Bucket    string
	ObjectKey string
}

func startAvatarStorageWorker(ctx context.Context, database *sql.DB, supabaseURL, serviceKey string, logger ports.Logger) {
	if supabaseURL == "" || serviceKey == "" {
		logger.Warn("avatar storage sync disabled because supabase storage configuration is missing")
		return
	}

	ticker := time.NewTicker(avatarStorageWorkerInterval)
	defer ticker.Stop()

	for {
		if err := processPendingAvatarStorageJobs(ctx, database, supabaseURL, serviceKey); err != nil {
			logger.Warn("avatar storage sync cycle failed", "error", err)
		}

		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func processPendingAvatarStorageJobs(ctx context.Context, database *sql.DB, supabaseURL, serviceKey string) error {
	rows, err := database.QueryContext(ctx, `
		SELECT id, user_id, local_path, bucket, object_key
		FROM storage_sync_jobs
		WHERE status = 'pending'
		ORDER BY updated_at ASC
		LIMIT 10
	`)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var job avatarStorageJob
		if err := rows.Scan(&job.ID, &job.UserID, &job.LocalPath, &job.Bucket, &job.ObjectKey); err != nil {
			return err
		}
		if err := uploadAvatarStorageJob(ctx, database, supabaseURL, serviceKey, job); err != nil {
			_, _ = database.ExecContext(ctx, `
				UPDATE storage_sync_jobs
				SET status = 'pending', attempts = attempts + 1, last_error = ?, updated_at = ?
				WHERE id = ?
		`, err.Error(), storageTimeValue(time.Now()), job.ID)
		}
	}
	return rows.Err()
}

func uploadAvatarStorageJob(ctx context.Context, database *sql.DB, supabaseURL, serviceKey string, job avatarStorageJob) error {
	fileBytes, err := os.ReadFile(job.LocalPath)
	if err != nil {
		return fmt.Errorf("read avatar file: %w", err)
	}

	uploadURL := fmt.Sprintf("%s/storage/v1/object/%s/%s", supabaseURL, url.PathEscape(job.Bucket), pathEscapeObjectKey(job.ObjectKey))
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, uploadURL, bytes.NewReader(fileBytes))
	if err != nil {
		return err
	}
	request.Header.Set("Authorization", "Bearer "+serviceKey)
	request.Header.Set("apikey", serviceKey)
	request.Header.Set("Content-Type", http.DetectContentType(fileBytes))
	request.Header.Set("x-upsert", "true")

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	if response.StatusCode < 200 || response.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(response.Body, 1024))
		return fmt.Errorf("supabase storage upload failed: %s %s", response.Status, string(body))
	}

	publicURL := fmt.Sprintf("%s/storage/v1/object/public/%s/%s", supabaseURL, url.PathEscape(job.Bucket), pathEscapeObjectKey(job.ObjectKey))
	now := time.Now()
	if _, err := database.ExecContext(ctx, `
		UPDATE users SET avatar_url = ?, updated_at = ? WHERE id = ? AND deleted_at IS NULL
	`, publicURL, storageTimeValue(now), job.UserID); err != nil {
		return err
	}
	if _, err := database.ExecContext(ctx, `
		UPDATE storage_sync_jobs SET status = 'completed', updated_at = ?, last_error = NULL WHERE id = ?
	`, storageTimeValue(now), job.ID); err != nil {
		return err
	}
	return nil
}

func pathEscapeObjectKey(objectKey string) string {
	escaped := url.PathEscape(objectKey)
	return strings.ReplaceAll(escaped, "%2F", "/")
}

func storageTimeValue(value time.Time) string {
	return value.UTC().Format(time.RFC3339Nano)
}
