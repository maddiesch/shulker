package model

import (
	"context"
	"errors"

	"github.com/maddiesch/go-raptor"
	"github.com/maddiesch/go-raptor/statement"
	"golang.org/x/crypto/bcrypt"
)

type UserPermission uint64

const (
	UserPermissionLogin UserPermission = 1 << iota
	UserPermissionEditor
	UserPermissionAdmin
)

type CreateUserParams struct {
	Username    string `validate:"required,min=3,max=64"`
	Password    string `validate:"required,min=6"`
	Permissions UserPermission
}

func CreateUser(ctx context.Context, x raptor.Executor, p CreateUserParams) error {
	if err := Validate.Struct(p); err != nil {
		return err
	}

	password, err := bcrypt.GenerateFromPassword([]byte(p.Password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	stmt := statement.Insert().Into("Users").ValueMap(map[string]any{
		"Username":    p.Username,
		"Password":    password,
		"Permissions": p.Permissions,
	})

	_, err = raptor.ExecStatement(ctx, x, stmt)

	return err
}

func CheckUsernamePassword(ctx context.Context, db raptor.Querier, username string) (UserPermission, error) {
	return 0, errors.New("invalid username or password")
}
