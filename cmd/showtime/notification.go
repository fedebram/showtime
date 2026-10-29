package main

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/spf13/cobra"

	"github.com/fedebram/showtime/internal/notification"
	notificationmemory "github.com/fedebram/showtime/internal/notification/memory"
	notificationserver "github.com/fedebram/showtime/internal/notification/server"
	notificationsqlite "github.com/fedebram/showtime/internal/notification/sqlite"
)

// newServeNotificationCmd runs the notification service alone.
func newServeNotificationCmd() *cobra.Command {
	var addr, db string
	cmd := &cobra.Command{
		Use:   "notification",
		Short: "Run the notification service: receives and sends notifications",
		RunE: func(cmd *cobra.Command, args []string) error {
			repo, closeRepo, err := openNotificationRepo(db)
			if err != nil {
				return err
			}
			defer closeRepo()

			notifications := notification.NewService(repo, time.Now)
			return serve(cmd.Context(), addr, notificationserver.NewServer(notifications))
		},
	}
	// 8081: booking takes 8080, both can run on the same machine
	cmd.Flags().StringVar(&addr, "addr", ":8081", "listen address")
	cmd.Flags().StringVar(&db, "db", "", "SQLite database file (empty: in memory)")
	return cmd
}

// openNotificationRepo opens SQLite at path, or memory if path is empty.
// The returned func closes it.
func openNotificationRepo(path string) (notification.Repository, func() error, error) {
	if path == "" {
		slog.Info("notification: using in-memory repository")
		return notificationmemory.NewRepository(), func() error { return nil }, nil
	}
	repo, err := notificationsqlite.Open(path)
	if err != nil {
		return nil, nil, fmt.Errorf("open notification db: %w", err)
	}
	slog.Info("notification: using SQLite", "path", path)
	return repo, repo.Close, nil
}
