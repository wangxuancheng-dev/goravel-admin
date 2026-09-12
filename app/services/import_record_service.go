package services

import (
	"context"

	apperrors "goravel/app/errors"
	appfacades "goravel/app/facades"
	"goravel/app/models"
)

type ImportRecordService interface {
	GetByID(id uint) (*models.Import, error)
}

type ImportRecordServiceImpl struct {
	ctx context.Context
}

func NewImportRecordService(ctx context.Context) ImportRecordService {
	return &ImportRecordServiceImpl{ctx: ctx}
}

func (s *ImportRecordServiceImpl) GetByID(id uint) (*models.Import, error) {
	var item models.Import
	if err := appfacades.OrmQuery(s.ctx).Where("id", id).FirstOrFail(&item); err != nil {
		return nil, apperrors.ErrRecordNotFound.WithError(err)
	}
	return &item, nil
}
