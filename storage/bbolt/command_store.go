package bbolt

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"time"

	bolt "go.etcd.io/bbolt"

	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

// CommandStore persists commands in a bbolt database. It implements
// command.Store, command.CommandJournal, and command.SeekableCommandJournal.
type CommandStore struct {
	storeBase
}

// NewCommandStore creates a CommandStore sharing the given *bbolt.DB.
func NewCommandStore(db *bolt.DB, logger *slog.Logger) (*CommandStore, error) {
	return &CommandStore{storeBase{db: db, logger: logger}}, nil
}

func commandStreamKey(ref id.StreamRef, cmdID id.CommandID) []byte {
	return fmt.Appendf(nil, "%s/%s/%s", ref.Type, ref.ID, cmdID)
}

func commandStreamPrefix(ref id.StreamRef) []byte {
	return fmt.Appendf(nil, "%s/%s/", ref.Type, ref.ID)
}

func commandJournalKey(cmdID id.CommandID) []byte {
	return []byte(cmdID.String())
}

// Save persists a single command. Returns command.ErrDuplicateCommand if a
// command with the same ID already exists.
func (s *CommandStore) Save(
	_ context.Context,
	ref id.StreamRef,
	cmd *command.PersistedCommand,
) error {
	data, err := marshalCommand(cmd)
	if err != nil {
		return err
	}

	cmdID := cmd.ID()
	streamKey := commandStreamKey(ref, cmdID)
	journalKey := commandJournalKey(cmdID)

	return s.db.Update(func(tx *bolt.Tx) error {
		jBucket := tx.Bucket([]byte(bucketCmdJournal))
		if jBucket.Get(journalKey) != nil {
			return command.ErrDuplicateCommand
		}

		sBucket := tx.Bucket([]byte(bucketCommands))
		if sBucket.Get(streamKey) != nil {
			return command.ErrDuplicateCommand
		}

		if err := sBucket.Put(streamKey, data); err != nil {
			return wrapBucketErr(err, "bbolt.command_save", "put command stream key")
		}

		return wrapBucketErr(
			jBucket.Put(journalKey, data),
			"bbolt.command_save",
			"put command journal key",
		)
	})
}

// AppendBatch persists multiple commands atomically. All commands are written
// or none. Returns command.ErrDuplicateCommand if any command ID already exists.
func (s *CommandStore) AppendBatch(
	_ context.Context,
	ref id.StreamRef,
	cmds []*command.PersistedCommand,
) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		sBucket := tx.Bucket([]byte(bucketCommands))
		jBucket := tx.Bucket([]byte(bucketCmdJournal))

		for _, cmd := range cmds {
			data, err := marshalCommand(cmd)
			if err != nil {
				return err
			}

			cmdID := cmd.ID()
			streamKey := commandStreamKey(ref, cmdID)
			journalKey := commandJournalKey(cmdID)

			if jBucket.Get(journalKey) != nil || sBucket.Get(streamKey) != nil {
				return command.ErrDuplicateCommand
			}

			if err := sBucket.Put(streamKey, data); err != nil {
				return wrapBucketErr(err, "bbolt.command_batch", "put command stream key")
			}

			if err := jBucket.Put(journalKey, data); err != nil {
				return wrapBucketErr(err, "bbolt.command_batch", "put command journal key")
			}
		}

		return nil
	})
}

// Load retrieves all commands for a stream, ordered by command ID.
func (s *CommandStore) Load(
	_ context.Context,
	ref id.StreamRef,
) ([]*command.PersistedCommand, error) {
	prefix := commandStreamPrefix(ref)

	var cmds []*command.PersistedCommand

	err := s.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(bucketCommands))
		c := bucket.Cursor()

		for k, v := c.Seek(prefix); k != nil && bytes.HasPrefix(k, prefix); k, v = c.Next() {
			cmd, err := unmarshalCommand(v)
			if err != nil {
				return err
			}

			cmds = append(cmds, cmd)
		}

		return nil
	})

	return cmds, wrapBucketErr(err, "bbolt.command_load", "load commands for stream")
}

// LoadFromTimestamp returns commands for a stream received after the given time.
func (s *CommandStore) LoadFromTimestamp(
	_ context.Context,
	ref id.StreamRef,
	after time.Time,
) ([]*command.PersistedCommand, error) {
	return s.loadByTimestamp(ref, after, false)
}

// LoadToTimestamp returns commands for a stream received up to the given time.
func (s *CommandStore) LoadToTimestamp(
	_ context.Context,
	ref id.StreamRef,
	maxTime time.Time,
) ([]*command.PersistedCommand, error) {
	return s.loadByTimestamp(ref, maxTime, true)
}

func (s *CommandStore) loadByTimestamp(
	ref id.StreamRef,
	ts time.Time,
	before bool,
) ([]*command.PersistedCommand, error) {
	prefix := commandStreamPrefix(ref)
	var cmds []*command.PersistedCommand

	err := s.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(bucketCommands))
		c := bucket.Cursor()

		for k, v := c.Seek(prefix); k != nil && bytes.HasPrefix(k, prefix); k, v = c.Next() {
			cmd, err := unmarshalCommand(v)
			if err != nil {
				return err
			}

			if before {
				if !cmd.ReceivedAt().Before(ts) {
					break
				}
			} else {
				if !cmd.ReceivedAt().After(ts) {
					continue
				}
			}

			cmds = append(cmds, cmd)
		}

		return nil
	})

	return cmds, wrapBucketErr(err, "bbolt.command_load_ts", "load commands by timestamp")
}

// ReadAll returns all commands across all streams, ordered by command ID.
func (s *CommandStore) ReadAll(_ context.Context) ([]*command.PersistedCommand, error) {
	var cmds []*command.PersistedCommand

	err := s.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(bucketCmdJournal))

		return bucket.ForEach(func(_ []byte, v []byte) error {
			cmd, err := unmarshalCommand(v)
			if err != nil {
				return err
			}

			cmds = append(cmds, cmd)

			return nil
		})
	})

	return cmds, wrapBucketErr(err, "bbolt.command_read_all", "read all commands from journal")
}

// ReadFrom returns commands from the journal starting after the given command
// ID, up to limit entries. A limit of 0 means no limit.
func (s *CommandStore) ReadFrom(
	_ context.Context,
	afterCmdID id.CommandID,
	limit int,
) ([]*command.PersistedCommand, error) {
	seekKey := commandJournalKey(afterCmdID)
	var cmds []*command.PersistedCommand

	err := s.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(bucketCmdJournal))
		c := bucket.Cursor()

		k, v := c.Seek(seekKey)

		if k != nil && bytes.Equal(k, seekKey) {
			k, v = c.Next()
		}

		for ; k != nil; k, v = c.Next() {
			cmd, err := unmarshalCommand(v)
			if err != nil {
				return err
			}

			cmds = append(cmds, cmd)

			if limit > 0 && len(cmds) >= limit {
				break
			}
		}

		return nil
	})

	return cmds, wrapBucketErr(
		err,
		"bbolt.command_read_from",
		"read commands from journal position",
	)
}
