package store

import (
	"chatgpt/config"
	"chatgpt/models"
	"context"
	"errors"
	"fmt"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type DbClientReal struct {
	Db *gorm.DB
}

func NewConn(db *gorm.DB) *DbClientReal {
	return &DbClientReal{db}
}

func NewDB(config *config.Config, out *models.DbClient) error {
	logLevel := logger.Silent
	if config.DbLogMode {
		logLevel = logger.Info
	}

	connectionString := fmt.Sprintf("host=%v port=%v user=%v dbname=%v password=%v sslmode=%v",
		config.DbHost, config.DbPort, config.DbUser, config.DbName, config.DbPass, config.DbMode)

	db, err := gorm.Open(postgres.Open(connectionString), &gorm.Config{
		Logger: logger.Default.LogMode(logLevel),
	})
	if err != nil {
		return err
	}

	conn := NewConn(db)
	if out != nil && db != nil {
		*out = conn
	}
	return nil
}

func (this *DbClientReal) CloseClient() error {
	sqlDb, err := this.Db.DB()
	if err != nil {
		return err
	}
	return sqlDb.Close()
}

func (this *DbClientReal) PingClient(ctx context.Context) error {
	sqlDB, err := this.Db.DB()
	if err != nil {
		return err
	}
	return sqlDB.PingContext(ctx)
}

func (this *DbClientReal) Select(ctx context.Context, table string, params models.FilterParams, out interface{}) error {
	exec := this.Db.WithContext(ctx).Table(table).Select(params.Select).Where(params.Filter).Scan(out)
	if exec.Error != nil {
		return exec.Error
	}
	if exec.RowsAffected == 0 {
		return errors.New(models.DB_ERROR_NOT_FOUND)
	}
	return nil
}

func (this *DbClientReal) Get(ctx context.Context, query models.FilterParams, out interface{}) error {
	exec := this.Db.WithContext(ctx).Order(query.Orderings).Where(query.Filter).Limit(query.ValidLimit()).Find(out)

	if exec.Error != nil {
		return exec.Error
	}
	if exec.RowsAffected == 0 {
		return errors.New(models.DB_ERROR_NOT_FOUND)
	}
	return nil
}

func (this *DbClientReal) GetView(ctx context.Context, viewName string, params models.FilterParams, out interface{}) error {
	exec := this.Db.WithContext(ctx).Table(viewName).Order(params.Orderings).Where(params.Filter).Limit(params.ValidLimit()).Offset(params.Offset).Find(out)
	if exec.Error != nil {
		return exec.Error
	}
	if exec.RowsAffected == 0 {
		return errors.New(models.DB_ERROR_NOT_FOUND)
	}
	return nil
}

func (this *DbClientReal) Create(ctx context.Context, input interface{}) error {
	if input == nil {
		return errors.New("creation data is nil")
	}
	return this.Db.WithContext(ctx).Create(input).Error
}

func (d *DbClientReal) Update(ctx context.Context, params models.FilterParams, input interface{}) error {
	if input == nil {
		return errors.New("updated data is nil")
	}
	exec := d.Db.WithContext(ctx).Where(params.Filter).Updates(input)
	if exec.Error != nil {
		return exec.Error
	}
	if exec.RowsAffected == 0 {
		return errors.New(models.DB_ERROR_NOT_FOUND)
	}
	return nil
}

func (d *DbClientReal) Upsert(ctx context.Context, params models.FilterParams, input interface{}) error {
	if input == nil {
		return errors.New("data is nil")
	}
	err := d.Update(ctx, params, input)
	if models.IsErrNotFound(err) {
		return d.Create(ctx, input)
	}
	return err
}

func (d *DbClientReal) Delete(ctx context.Context, params models.FilterParams, input interface{}) error {
	if input == nil {
		return errors.New("delete data is nil")
	}
	exec := d.Db.WithContext(ctx).Where(params.Filter).Delete(input)
	if exec.Error != nil {
		return exec.Error
	}
	if exec.RowsAffected == 0 {
		return errors.New(models.DB_ERROR_NOT_FOUND)
	}
	return nil
}
