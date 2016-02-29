package data

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
