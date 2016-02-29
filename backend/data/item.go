package data

import (
	"io"
)

func MarkItemRead(db Queryer, userID, itemID int32) error {
	commandTag, err := db.Exec("markItemRead", userID, itemID)
	if err != nil {
		return err
	}
	if commandTag != "DELETE 1" {
		return ErrNotFound
	}

	return nil
}

func CopySubscriptionsForUserAsJSON(db Queryer, w io.Writer, userID int32) error {
	var b []byte
	err := db.QueryRow("getFeedsForUser", userID).Scan(&b)
	if err != nil {
		return err
	}

	_, err = w.Write(b)
	return err
}

func CopyUnreadItemsAsJSONByUserID(db Queryer, w io.Writer, userID int32) error {
	var b []byte
	err := db.QueryRow("getUnreadItems", userID).Scan(&b)
	if err != nil {
		return err
	}

	_, err = w.Write(b)
	return err
}
