package main

import (
	"database/sql"
	"log"

	_ "github.com/lib/pq"
)

type Storage interface {
	CreateAccount(*Account) error
	DeleteAccount(int) error
	UpdateAccount(*Account) error
	GetAccountById(int) (*Account, error)
	GetAccounts() ([]*Account, error)
}

type PostgressStore struct {
	db *sql.DB
}

func NewPostgressStore() *PostgressStore {
	connStr := "host=127.0.0.1 port=5444 password=mysecret user=postgres dbname=practicedb sslmode=disable"
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal(err)
	}

	if err = db.Ping(); err != nil {
		log.Fatal(err)
	}

	return &PostgressStore{
		db: db,
	}
}

func (p *PostgressStore) Init() error {
	return p.createAccountTable()
}

func (p *PostgressStore) createAccountTable() error {
	query := `CREATE TABLE IF NOT EXISTS account (
		id serial primary key,
		first_name varchar(50),
		last_name varchar(50),
		number serial,
		balance serial,
		created_at timestamp
	)`

	_, err := p.db.Exec(query)

	return err
}

/*
===============
Psql Functions
===============
*/

func (p *PostgressStore) CreateAccount(a *Account) error {
	query := `INSERT INTO account (first_name, last_name, number, balance, created_at)
	values ($1, $2, $3, $4, $5)`

	_, err := p.db.Exec(query, a.FirstName, a.LastName, a.Number, a.Balance, a.CreatedAt)

	return err
}

func (p *PostgressStore) GetAccounts() ([]*Account, error) {
	query := `SELECT * FROM account`

	rows, err := p.db.Query(query)
	if err != nil {
		return nil, err
	}

	accounts := []*Account{}

	for rows.Next() {
		account := &Account{}
		err := rows.Scan(
			&account.ID,
			&account.FirstName,
			&account.LastName,
			&account.Number,
			&account.Balance,
			&account.CreatedAt,
		)
		if err != nil {
			return nil, err
		}

		accounts = append(accounts, account)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return accounts, nil
}

func (p *PostgressStore) DeleteAccount(id int) error {
	// in production you would soft delete it... or tranctions
	query := `DELETE FROM account WHERE id=$1`

	_, err := p.db.Exec(query, id)
	return err
}

func (p *PostgressStore) UpdateAccount(account *Account) error {
	return nil
}

func (p *PostgressStore) GetAccountById(id int) (*Account, error) {
	query := `SELECT id, first_name, last_name, number, balance, created_at FROM account WHERE id=$1`

	account := &Account{}

	row := p.db.QueryRow(query, id)
	err := row.Scan(
		&account.ID,
		&account.FirstName,
		&account.LastName,
		&account.Number,
		&account.Balance,
		&account.CreatedAt,
	)

	switch err {
	case nil:
		return account, nil
	default:
		log.Println("ERROR:", err)
		return nil, err
	}
}
