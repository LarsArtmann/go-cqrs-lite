package storage

import (
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/larsartmann/go-cqrs-lite/command/v3"
	"github.com/larsartmann/go-cqrs-lite/id/v3"
	sqlpkg "github.com/larsartmann/go-cqrs-lite/storage/v3/sql"
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
		return nil, command.WrapInfrastructure(err, "storage.scan_command",
			fmt.Sprintf("scan command row for %s/%s command %s (id %s)",
				aggIDStr, aggType, commandType, commandIDStr))
	}

	receivedAt, err := s.Dialect.ParseTime(timeDest)
	if err != nil {
		return nil, command.WrapCorruption(err, "storage.parse_received_at",
			fmt.Sprintf("parse received_at for %s/%s command %s (id %s)",
				aggIDStr, aggType, commandType, commandIDStr))
	}

	parsedCommandID, err := id.ParseCommandID(commandIDStr)
	if err != nil {
		return nil, command.WrapCorruption(
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

	parsedAggID, err := id.ParseAggregateID(aggIDStr)
	if err != nil {
		return nil, command.WrapCorruption(err, "storage.parse_aggregate_id",
			fmt.Sprintf("parse aggregate ID %q for %s command %s", aggIDStr, aggType, commandType))
	}

	parsedAggType, err := command.ParseAggregateType(aggType)
	if err != nil {
		return nil, command.WrapCorruption(err, "storage.parse_aggregate_type",
			fmt.Sprintf("parse aggregate type %q for command %s", aggType, commandType))
	}

	ref := command.NewAggregateRef(parsedAggType, parsedAggID)

	opts := []command.PersistOption{
		command.WithCommandID(parsedCommandID),
		command.WithReceivedAt(receivedAt),
	}

	if len(metadataJSON) > 0 {
		var meta command.Metadata
		if jsonErr := json.Unmarshal(metadataJSON, &meta); jsonErr != nil {
			return nil, command.WrapCorruption(jsonErr, "storage.parse_command_metadata",
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
		return nil, command.WrapCorruption(err, "storage.reconstruct_command",
			fmt.Sprintf("reconstruct command %s for %s", commandType, aggType))
	}

	return cmd, nil
}
