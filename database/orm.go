package database

import (
	"context"
	"fmt"
	"gorm.io/gorm"

	"github.com/gookit/color"
	"github.com/pkg/errors"

	contractsorm "github.com/sujit-baniya/framework/contracts/database/orm"
	"github.com/sujit-baniya/framework/facades"
)

type Orm struct {
	ctx             context.Context
	connection      string
	defaultInstance contractsorm.DB
	instances       map[string]contractsorm.DB
	config          *gorm.Config
	disableLog      bool
}

func NewOrm(ctx context.Context, config *gorm.Config, disableLog bool) contractsorm.Orm {
	orm := &Orm{ctx: ctx, config: config, disableLog: disableLog}

	return orm.Connection("", config, disableLog)
}

func (r *Orm) Connection(name string, config *gorm.Config, disableLog bool) contractsorm.Orm {
	defaultConnection := facades.Config.GetString("database.default")
	if name == "" {
		name = defaultConnection
	}

	r.connection = name
	if r.instances == nil {
		r.instances = make(map[string]contractsorm.DB)
	}

	if _, exist := r.instances[name]; exist {
		return r
	}

	g, err := NewGormDB(r.ctx, name, config, disableLog)
	if err != nil {
		color.Redln(fmt.Sprintf("[Orm] Init connection error, %v", err))

		return nil
	}
	if g == nil {
		return nil
	}

	r.instances[name] = g

	if name == defaultConnection {
		r.defaultInstance = g
	}

	return r
}

func (r *Orm) Query() contractsorm.DB {
	if r.connection == "" {
		if r.defaultInstance == nil {
			r.Connection("", r.config, r.disableLog)
		}

		return r.defaultInstance
	}

	instance, exist := r.instances[r.connection]
	if !exist {
		return nil
	}

	r.connection = ""

	return instance
}

func (r *Orm) Transaction(txFunc func(tx contractsorm.Transaction) error) error {
	tx, err := r.Query().Begin()
	if err != nil {
		return err
	}

	if err := txFunc(tx); err != nil {
		if err := tx.Rollback().Error; err != nil {
			return errors.Wrapf(err, "rollback error: %v", err)
		}

		return err
	} else {
		return tx.Commit().Error
	}
}

func (r *Orm) WithContext(ctx context.Context) contractsorm.Orm {
	return NewOrm(ctx, r.config, r.disableLog)
}
