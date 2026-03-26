package db

import (
	"database/sql"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type Note struct {
	ID    int64
	Title string
	Body  string
}

type Store struct {
	conn *sql.DB
}

func (s *Store) Init() error {
	var err error
	s.conn, err = sql.Open("sqlite3", "./bin/notes.db")
	if err != nil {
		return err
	}

	createTableStmt := `CREATE TABLE IF NOT EXISTS notes (
		id integer not null primary key,
		title text not null,
		body text not null
	)`

	if _, err = s.conn.Exec(createTableStmt); err != nil {
		return err
	}

	createHistoryStmt := `CREATE TABLE IF NOT EXISTS command_history (
		id integer not null primary key,
		command text not null,
		ran_at datetime not null
	)`

	if _, err = s.conn.Exec(createHistoryStmt); err != nil {
		return err
	}

	return nil
}

func (s *Store) SaveCommand(command string) error {
	_, err := s.conn.Exec(
		`INSERT INTO command_history (id, command, ran_at) VALUES (?, ?, ?)`,
		time.Now().UTC().UnixNano(), command, time.Now().UTC(),
	)
	return err
}

func (s *Store) GetCommandHistory(limit int) ([]string, error) {
	rows, err := s.conn.Query(
		`SELECT command FROM command_history ORDER BY ran_at DESC LIMIT ?`, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var cmds []string
	for rows.Next() {
		var cmd string
		rows.Scan(&cmd)
		cmds = append(cmds, cmd)
	}
	// reverse so oldest is first (index 0), newest is last
	for i, j := 0, len(cmds)-1; i < j; i, j = i+1, j-1 {
		cmds[i], cmds[j] = cmds[j], cmds[i]
	}
	return cmds, nil
}

func (s *Store) GetNotes() ([]Note, error) {
	rows, err := s.conn.Query("SELECT * FROM notes")
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	notes := []Note{}
	for rows.Next() {
		var note Note
		rows.Scan(&note.ID, &note.Title, &note.Body)
		notes = append(notes, note)
	}

	return notes, nil
}

func (s *Store) SaveNote(note Note) error {
	if note.ID == 0 {
		note.ID = time.Now().UTC().UnixNano()
	}

	upsertQuery := `INSERT INTO notes (id, title, body)
	VALUES (?, ?, ?)
	ON CONFLICT(id) DO UPDATE
	SET title=excluded.title, body=excluded.body`

	if _, err := s.conn.Exec(upsertQuery, note.ID, note.Title, note.Body); err != nil {
		return err
	}

	return nil
}
