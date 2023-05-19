package services

import (
	"fmt"
	"time"

	"gin-app/models"
)

type UserService Service

func (s *UserService) GetAll() ([]*models.User, error) {
	var users []*models.User

	err := s.DB.Select(&users, "select * from user limit 10")
	if err != nil {
		return nil, err
	}
	return users, nil
}

func (s *UserService) GetByID(id uint) (*models.User, error) {
	var user models.User
	field := "*"
	sqlString := fmt.Sprintf("select %s from users where id = %d", field, id)
	err := s.DB.QueryRowx(sqlString).StructScan(&user)
	if err != nil {
		return nil, err
	}

	return &user, nil
}

func (s *UserService) Create(user *models.User) error {
	now := time.Now()
	user.CreatedAt = now
	user.UpdatedAt = now
	res := s.DB.MustExec("insert into user (name, age, email, created_at, updated_at) values (?,?,?,?,?)", user.Name, user.Age, user.Email, user.CreatedAt, user.UpdatedAt)
	if _, err := res.LastInsertId(); err != nil {
		return err
	}
	return nil
}

func (s *UserService) Update(id uint, user *models.User) error {
	//var oldUser models.User
	//err := s.db.First(&oldUser, id).Error
	//if err != nil {
	//	return err
	//}
	//
	//oldUser.Name = user.Name
	//oldUser.Age = user.Age
	//oldUser.Email = user.Email
	//oldUser.UpdatedAt = time.Now()
	//
	//err = s.db.Save(&oldUser).Error
	//if err != nil {
	//	return err
	//}
	return nil
}

func (s *UserService) Delete(id uint) error {
	sqlString := fmt.Sprintf("delect from users where id = %d", id)
	rows, err := s.DB.Query(sqlString)
	if err != nil {
		return err
	}
	fmt.Println(rows)
	return nil
}
