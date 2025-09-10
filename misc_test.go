package gopager

import (
	"github.com/DATA-DOG/go-sqlmock"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func newGORMMySQLMock() (string, *gorm.DB, sqlmock.Sqlmock, error) {
	mockDB, mock, err := sqlmock.New()
	if err != nil {
		return "", nil, nil, err
	}

	dialector := mysql.New(mysql.Config{
		Conn:                      mockDB,
		SkipInitializeWithVersion: true,
	})

	db, err := gorm.Open(dialector, &gorm.Config{})
	if err != nil {
		return "", nil, nil, err
	}

	return "mysql", db.Debug(), mock, nil
}

func newGORMPostgresMock() (string, *gorm.DB, sqlmock.Sqlmock, error) {
	mockDB, mock, err := sqlmock.New()
	if err != nil {
		return "", nil, nil, err
	}

	dialector := postgres.New(postgres.Config{
		Conn: mockDB,
	})

	db, err := gorm.Open(dialector, &gorm.Config{})
	if err != nil {
		return "", nil, nil, err
	}

	return "postgres", db.Debug(), mock, nil
}
