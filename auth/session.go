package auth

import (
	"errors"
	"github.com/sujit-baniya/frame"
	"reflect"
	"strings"
	"time"

	contractauth "github.com/sujit-baniya/framework/contracts/auth"
	"github.com/sujit-baniya/framework/facades"
	supporttime "github.com/sujit-baniya/framework/support/time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/spf13/cast"
)

type Session struct {
	guard string
}

func NewSession(guard string) contractauth.Auth {
	return &Session{
		guard: guard,
	}
}

func (app *Session) Guard(name string) contractauth.Auth {
	return NewAuth(name)
}

// User need parse token first.
func (app *Session) User(ctx *frame.Context, user any) error {
	auth, ok := ctx.Value(ctxKey).(Auth)
	if !ok || auth[app.guard] == nil {
		return ErrorParseTokenFirst
	}
	if auth[app.guard].Claims == nil {
		return ErrorParseTokenFirst
	}
	if auth[app.guard].Token == "" {
		return ErrorTokenExpired
	}
	if err := facades.Orm.Query().Find(user, auth[app.guard].Claims.Key); err != nil {
		return err
	}

	return nil
}

func (app *Session) Parse(ctx *frame.Context, token string) error {
	token = strings.ReplaceAll(token, "Bearer ", "")
	if tokenIsDisabled(token) {
		return ErrorTokenDisabled
	}

	jwtSecret := facades.Config.GetString("jwt.secret")
	tokenClaims, err := jwt.ParseWithClaims(token, &Claims{}, func(token *jwt.Token) (any, error) {
		return []byte(jwtSecret), nil
	})
	if err != nil {
		if strings.Contains(err.Error(), jwt.ErrTokenExpired.Error()) && tokenClaims != nil {
			claims, ok := tokenClaims.Claims.(*Claims)
			if !ok {
				return ErrorInvalidClaims
			}

			app.makeAuthContext(ctx, claims, "")

			return ErrorTokenExpired
		} else {
			return err
		}
	}
	if tokenClaims == nil || !tokenClaims.Valid {
		return ErrorInvalidToken
	}

	claims, ok := tokenClaims.Claims.(*Claims)
	if !ok {
		return ErrorInvalidClaims
	}

	app.makeAuthContext(ctx, claims, token)

	return nil
}

func (app *Session) Login(ctx *frame.Context, user any) (token string, err error) {
	t := reflect.TypeOf(user).Elem()
	v := reflect.ValueOf(user).Elem()
	for i := 0; i < t.NumField(); i++ {
		if t.Field(i).Name == "Model" {
			if v.Field(i).Type().Kind() == reflect.Struct {
				structField := v.Field(i).Type()
				for j := 0; j < structField.NumField(); j++ {
					if structField.Field(j).Tag.Get("gorm") == "primaryKey" {
						return app.LoginUsingID(ctx, v.Field(i).Field(j).Interface())
					}
				}
			}
		}
		if t.Field(i).Tag.Get("gorm") == "primaryKey" {
			return app.LoginUsingID(ctx, v.Field(i).Interface())
		}
	}

	return "", ErrorNoPrimaryKeyField
}

func (app *Session) LoginUsingID(ctx *frame.Context, id any) (token string, err error) {
	jwtSecret := facades.Config.GetString("jwt.secret")
	if jwtSecret == "" {
		return "", ErrorEmptySecret
	}

	nowTime := supporttime.Now()
	ttl := facades.Config.GetInt("jwt.ttl")
	expireTime := nowTime.Add(time.Duration(ttl) * unit)
	claims := Claims{
		cast.ToString(id),
		jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expireTime),
			IssuedAt:  jwt.NewNumericDate(nowTime),
			Subject:   app.guard,
		},
	}

	tokenClaims := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	token, err = tokenClaims.SignedString([]byte(jwtSecret))
	if err != nil {
		return "", err
	}

	app.makeAuthContext(ctx, &claims, token)

	return
}

// Refresh need parse token first.
func (app *Session) Refresh(ctx *frame.Context) (token string, err error) {
	auth, ok := ctx.Value(ctxKey).(Auth)
	if !ok || auth[app.guard] == nil {
		return "", ErrorParseTokenFirst
	}
	if auth[app.guard].Claims == nil {
		return "", ErrorParseTokenFirst
	}

	nowTime := supporttime.Now()
	refreshTtl := facades.Config.GetInt("jwt.refresh_ttl")
	expireTime := auth[app.guard].Claims.ExpiresAt.Add(time.Duration(refreshTtl) * unit)
	if nowTime.Unix() > expireTime.Unix() {
		return "", ErrorRefreshTimeExceeded
	}

	return app.LoginUsingID(ctx, auth[app.guard].Claims.Key)
}

func (app *Session) Logout(ctx *frame.Context) error {
	auth, ok := ctx.Value(ctxKey).(Auth)
	if !ok || auth[app.guard] == nil || auth[app.guard].Token == "" {
		return nil
	}

	if facades.Cache == nil {
		return errors.New("cache support is required")
	}

	if err := facades.Cache.Put(getDisabledCacheKey(auth[app.guard].Token),
		true,
		time.Duration(facades.Config.GetInt("jwt.ttl"))*unit,
	); err != nil {
		return err
	}

	delete(auth, app.guard)
	ctx.Set(ctxKey, auth)

	return nil
}

func (app *Session) makeAuthContext(ctx *frame.Context, claims *Claims, token string) {
	ctx.Set(ctxKey, Auth{
		app.guard: {claims, token},
	})
}
