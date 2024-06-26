package ticketstore

import (
	"cinema/common"
	ticketmodel "cinema/module/ticket/model"
	"context"
)

func (store *sqlStore) GetTickets(_ context.Context, id []int) ([]ticketmodel.Ticket, error) {
	var result []ticketmodel.Ticket

	db := store.db.Table(ticketmodel.TableName)

	if len(id) > 0 {
		db = db.Where("id in (?)", id)

		if err := db.Find(&result).Error; err != nil {
			return nil, common.ErrDB(err)
		}
		return result, nil
	} else {
		return nil, nil
	}
}
