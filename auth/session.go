package auth

import (
	"errors"

	"github.com/oarkflow/frame"
	"github.com/oarkflow/frame/middlewares/server/session"
	"github.com/oarkflow/pkg/dto"

	"github.com/oarkflow/framework/contracts/auth"
)

type Session struct {
	guard string
}

func NewSession(guard string) auth.Auth {
	return &Session{
		guard: guard,
	}
}

func (app *Session) Guard(name string) auth.Auth {
	return GetAuth(name)
}

// User need parse token first.
func (app *Session) User(ctx *frame.Context, user auth.User) error {
	u, err := session.Get(ctx, ctx.AuthUserKey)
	if err != nil {
		return err
	}
	if u == nil {
		user = nil
		return errors.New("not logged in")
	}
	switch v := u.(type) {
	case auth.User:
		user = v
	default:
		err := dto.Map(user, v)
		if err != nil {
			return err
		}
	}
	if _, ok := ctx.Get(ctx.AuthUserKey); !ok {
		ctx.Set(ctx.AuthUserKey, user)
	}
	return nil
}

func (app *Session) Parse(ctx *frame.Context, token string) error {
	return nil
}

func (app *Session) Login(ctx *frame.Context, user auth.User) (token string, err error) {
	err = session.Set(ctx, ctx.AuthUserKey, user)
	if err != nil {
		return
	}
	ctx.Set(ctx.AuthUserKey, user)
	return
}

func (app *Session) LoginUsingID(ctx *frame.Context, id any) (token string, err error) {
	return
}

// Refresh need parse token first.
func (app *Session) Refresh(ctx *frame.Context) (token string, err error) {
	return "", nil
}

func (app *Session) Logout(ctx *frame.Context) error {
	return session.Destroy(ctx)
}
