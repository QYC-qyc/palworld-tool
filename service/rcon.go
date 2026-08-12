package service

import (
	"encoding/json"

	"github.com/google/uuid"
	"go.etcd.io/bbolt"
	"paladmin/internal/database"
)

func AddRconCommand(db *bbolt.DB, cmd database.RconCommand) (database.RconCommandList, error) {
	item := database.RconCommandList{UUID: uuid.NewString(), RconCommand: cmd}
	err := db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(database.BucketRcons))
		v, err := json.Marshal(item)
		if err != nil {
			return err
		}
		return b.Put([]byte(item.UUID), v)
	})
	return item, err
}

func PutRconCommand(db *bbolt.DB, id string, cmd database.RconCommand) error {
	return db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(database.BucketRcons))
		item := database.RconCommandList{UUID: id, RconCommand: cmd}
		v, err := json.Marshal(item)
		if err != nil {
			return err
		}
		return b.Put([]byte(id), v)
	})
}

func ListRconCommands(db *bbolt.DB) ([]database.RconCommandList, error) {
	result := make([]database.RconCommandList, 0)
	err := db.View(func(tx *bbolt.Tx) error {
		return tx.Bucket([]byte(database.BucketRcons)).ForEach(func(k, v []byte) error {
			var c database.RconCommandList
			if err := json.Unmarshal(v, &c); err == nil {
				result = append(result, c)
			}
			return nil
		})
	})
	return result, err
}

func RemoveRconCommand(db *bbolt.DB, id string) error {
	return db.Update(func(tx *bbolt.Tx) error {
		return tx.Bucket([]byte(database.BucketRcons)).Delete([]byte(id))
	})
}
