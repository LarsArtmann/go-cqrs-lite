package storage

import (
	"database/sql"
	"encoding/json/v2"
	"fmt"

	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	sqlpkg "github.com/larsartmann/go-cqrs-lite/storage/v4/sql"
)

func (s *SQLCommandStore) scanCommands(rows *sql.Rows) ([]*command.PersistedCommand, error) {
	return sqlpkg.ScanSlice(rows, s.scanCommand)
}

func (s *SQLCommandStore) scanCommand(rows *sql.Rows) (*command.PersistedCommand, error) {
	var (
		commandIDStr string
		commandType  string
		aggType      string
		aggIDStr     string
		payload      []byte
		metadataJSON []byte
	)
	timeDest := s.Dialect.ScanTimeDest()
	err := rows.Scan(
		&commandIDStr,
		&commandType,
		&aggType,
		&aggIDStr,
		&payload,
		&metadataJSON,
		timeDest,
	)
	if err != nil {
		return nil, errorfamily.WrapInfrastructure(err, "storage.scan_command",
			fmt.Sprintf("scan command row for %s/%s command %s (id %s)",
				aggIDStr, aggType, commandType, commandIDStr))
	}

	receivedAt, err := s.Dialect.ParseTime(timeDest)
	if err != nil {
		return nil, errorfamily.WrapCorruption(err, "storage.parse_received_at",
			fmt.Sprintf("parse received_at for %s/%s command %s (id %s)",
				aggIDStr, aggType, commandType, commandIDStr))
	}

	parsedCommandID, err := id.ParseCommandID(commandIDStr)
	if err != nil {
		return nil, errorfamily.WrapCorruption(
			err,
			"storage.parse_command_id",
			fmt.Sprintf(
				"parse command ID %q for %s command %s",
				commandIDStr,
				aggType,
				commandType,
			),
		)
	}

	parsedAggID, err := id.ParseStreamID(aggIDStr)
	if err != nil {
		return nil, errorfamily.WrapCorruption(err, "storage.parse_aggregate_id",
			fmt.Sprintf("parse stream ID %q for %s command %s", aggIDStr, aggType, commandType))
	}

	parsedAggType, err := command.ParseStreamType(aggType)
	if err != nil {
		return nil, errorfamily.WrapCorruption(err, "storage.parse_aggregate_type",
			fmt.Sprintf("parse stream type %q for command %s", aggType, commandType))
	}

	ref := command.NewStreamRef(parsedAggType, parsedAggID)

	opts := []command.PersistOption{
		command.WithPersistedCommandID(parsedCommandID),
		command.WithReceivedAt(receivedAt),
	}

	if len(metadataJSON) > 0 {
		var meta command.Metadata
		if jsonErr := json.Unmarshal(
			metadataJSON,
			&meta,
			json.MatchCaseInsensitiveNames(true),
		); jsonErr != nil {
			return nil, errorfamily.WrapCorruption(jsonErr, "storage.parse_command_metadata",
				fmt.Sprintf("unmarshal metadata for %s command (id %s)", commandType, commandIDStr))
		}
		opts = append(opts, command.WithCommandMetadata(meta))
	}

	cmd, err := command.NewPersistedCommand(
		command.Type(commandType),
		ref,
		payload,
		opts...,
	)
	if err != nil {
		return nil, errorfamily.WrapCorruption(err, "storage.reconstruct_command",
			fmt.Sprintf("reconstruct command %s for %s", commandType, aggType))
	}

	return cmd, nil
}
