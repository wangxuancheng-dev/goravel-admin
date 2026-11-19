package feedback

import (
	"context"
	"fmt"
	"strings"

	"goravel/packages/feedback/contracts"
	"goravel/packages/feedback/models"
)

// Service implements contracts.Service using ResolveQuery (tenant-safe when host injects it).
type Service struct{}

func NewService() *Service {
	return &Service{}
}

var _ contracts.Service = (*Service)(nil)

func (s *Service) List(ctx context.Context, page, pageSize int, status *uint8) ([]models.Feedback, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}
	if pageSize > 100 {
		pageSize = 100
	}

	countQ := query(ctx).Model(&models.Feedback{})
	listQ := query(ctx).Model(&models.Feedback{})
	if status != nil {
		countQ = countQ.Where("status", *status)
		listQ = listQ.Where("status", *status)
	}

	total, err := countQ.Count()
	if err != nil {
		return nil, 0, err
	}
	var list []models.Feedback
	if err := listQ.OrderBy("id", "desc").Offset((page-1)*pageSize).Limit(pageSize).Find(&list); err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

func (s *Service) Get(ctx context.Context, id uint) (*models.Feedback, error) {
	if id == 0 {
		return nil, fmt.Errorf("feedback not found")
	}
	var row models.Feedback
	if err := query(ctx).Model(&models.Feedback{}).Where("id", id).First(&row); err != nil || row.ID == 0 {
		return nil, fmt.Errorf("feedback not found")
	}
	return &row, nil
}

func (s *Service) Create(ctx context.Context, contact, content string) (*models.Feedback, error) {
	content = strings.TrimSpace(content)
	if content == "" {
		return nil, fmt.Errorf("content is required")
	}
	contact = strings.TrimSpace(contact)
	if len(contact) > 128 {
		contact = contact[:128]
	}
	row := models.Feedback{
		Contact:   contact,
		Content:   content,
		Status:    0,
		CreatedBy: actorID(ctx),
	}
	if err := query(ctx).Create(&row); err != nil {
		return nil, err
	}
	return &row, nil
}

func (s *Service) MarkDone(ctx context.Context, id uint) (*models.Feedback, error) {
	row, err := s.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	row.Status = 1
	if err := query(ctx).Save(row); err != nil {
		return nil, err
	}
	return row, nil
}

func (s *Service) Delete(ctx context.Context, id uint) error {
	row, err := s.Get(ctx, id)
	if err != nil {
		return err
	}
	_, err = query(ctx).Delete(row)
	return err
}
