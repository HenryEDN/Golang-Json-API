package main

import (
	"database/sql"
	"fmt"

	_ "github.com/lib/pq"
)

type Storage interface{
	CreateAccount(*Account) error
	DeleteAccount(int) error
	UpdateAccount(*Account) error
	GetAccountById(int) (*Account, error)
	GetAccounts() ([]*Account, error)
	GetAccountByNumber(int) (*Account, error)
}

type PostgresStore struct{
	db *sql.DB
}

func NewPostgresStore() (*PostgresStore, error){
	connStr := "user=postgres dbname=postgres password=gobank sslmode=disable"
	db, err := sql.Open("postgres", connStr)
	if err != nil{
		return nil, err
	}

	if err := db.Ping(); err != nil{
		return nil, err
	}

	return &PostgresStore{
		db: db,
	}, nil
}

func (s *PostgresStore) Init() error{
	return s.createAccountTable()
}

func (s *PostgresStore) createAccountTable() error{
	query := `CREATE TABLE IF NOT EXISTS ACCOUNT(
			  id serial primary key, 
			  first_name varchar(50), 
			  lastname varchar(50), 
			  number serial, 
			  encrypted_password varchar(256),
			  balance serial, 
			  created_at timestamp)`
	_, err := s.db.Exec(query)
	return err
}

func (s *PostgresStore) CreateAccount (acc *Account) error{
	query := `
	INSERT INTO ACCOUNT 
	(first_name, lastname, number, encrypted_password, balance, created_at)
	VALUES ($1, $2, $3, $4, $5, $6)`

	_, err := s.db.Query(
		query, 
		acc.FirstName,  
		acc.LastName, 
		acc.Number, 
		acc.EncryptedPassword,
		acc.Balance, 
		acc.CreatedAt)

	if err != nil{
		return err
	}

	// fmt.Printf("%+v\n", resp)

	return nil
}

func (s *PostgresStore) UpdateAccount (*Account) error{
	return nil
}

func (s *PostgresStore) DeleteAccount (id int) error{
	_, err := s.db.Query(`DELETE FROM ACCOUNT WHERE id = $1`, id)
	return err
}

func (s *PostgresStore) GetAccountByNumber(number int) (*Account, error){
	rows, err := s.db.Query(`SELECT * FROM ACCOUNT WHERE number = $1`, number)
	if err != nil{
		return nil, err
	}
	for rows.Next(){
		return scanIntoAccount(rows)
	}
	return nil, fmt.Errorf("account with number [%d] not found", number)
}

func (s *PostgresStore) GetAccountById (id int) (*Account, error){
	rows, err := s.db.Query(`SELECT * FROM ACCOUNT WHERE id = $1`, id)
	if err != nil{
		return nil, err
	}
	for rows.Next(){
		return scanIntoAccount(rows)
	}
	return nil, fmt.Errorf("account %d not found", id)
}

func (s *PostgresStore) GetAccounts() ([]*Account, error){
	query := `SELECT * FROM ACCOUNT`

	rows, err := s.db.Query(query)

	if err != nil{
		return nil, err
	}

	accounts := []*Account{}
	for rows.Next(){
		account, err := scanIntoAccount(rows)
			
		if err != nil{
			return nil, err
		}
	
		accounts = append(accounts, account)
	}

	return accounts, nil

}

func scanIntoAccount(rows *sql.Rows) (*Account, error){
	account := new(Account)
	err := rows.Scan(
		&account.ID,
		&account.FirstName,
		&account.LastName, 
		&account.Number,
		&account.EncryptedPassword,
		&account.Balance,
		&account.CreatedAt)
		
	return account, err
}