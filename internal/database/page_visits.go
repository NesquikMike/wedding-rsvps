package database

import (
	"fmt"
	"log"
)

func (i InviteeStore) createPageVisitsTable() error {
	createTableQuery := `CREATE TABLE IF NOT EXISTS page_visits (
        id INTEGER NOT NULL,
        page_name TEXT NOT NULL,
		visit_count INTEGER DEFAULT 1,
		first_visit_time TEXT,
		latest_visit_time TEXT,
		PRIMARY KEY (id, page_name)
    );`

	_, err := i.db.Exec(createTableQuery)
	if err != nil {
		return err
	}

	log.Println("page_visit table set up and populated successfully!")
	return nil
}

func (i InviteeStore) UpdatePageVisit(id int, page string) error {
	query := `INSERT INTO 
	    page_visits (id, page_name, visit_count, first_visit_time, latest_visit_time)
	VALUES (?, ?, 1, datetime('now'), datetime('now'))
	ON CONFLICT(id, page_name)
	DO UPDATE SET
		visit_count = visit_count + 1,
		latest_visit_time = datetime('now');
	`

	result, err := i.db.Exec(query, id, page)
	if err != nil {
		return fmt.Errorf("failed to save page visit: %v", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to retrieve affected rows: %v", err)
	}

	if rowsAffected != 1 {
		return fmt.Errorf("rowsAffected %v for id %v and page %s was not 1", rowsAffected, id, page)
	}

	return nil
}
