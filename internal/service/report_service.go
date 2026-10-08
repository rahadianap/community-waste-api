package service

import (
	"context"
	"fmt"

	"github.com/rahadianap/community-waste-api/internal/domain"
	"github.com/rahadianap/community-waste-api/internal/repository"
)

type ReportService interface {
	GetWasteSummary(ctx context.Context) ([]domain.WasteSummaryItem, error)
	GetPaymentSummary(ctx context.Context) (*domain.PaymentSummaryReport, error)
}

type reportService struct {
	store repository.Store
}

func NewReportService(store repository.Store) ReportService {
	return &reportService{store: store}
}

func (s *reportService) GetWasteSummary(ctx context.Context) ([]domain.WasteSummaryItem, error) {
	rows, err := s.store.GetWasteSummaryReport(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get waste summary report: %w", err)
	}

	items := make([]domain.WasteSummaryItem, len(rows))
	for i, r := range rows {
		items[i] = domain.WasteSummaryItem{
			Type:   string(r.Type),
			Status: string(r.Status),
			Total:  r.Total,
		}
	}

	return items, nil
}

func (s *reportService) GetPaymentSummary(ctx context.Context) (*domain.PaymentSummaryReport, error) {
	rows, err := s.store.GetPaymentSummaryReport(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get payment summary report: %w", err)
	}

	byStatus := make([]domain.PaymentStatusSummary, len(rows))
	for i, r := range rows {
		byStatus[i] = domain.PaymentStatusSummary{
			Status:      string(r.Status),
			Count:       r.Count,
			TotalAmount: domain.NumericToFloat64(r.TotalAmount),
		}
	}

	totalRevNumeric, err := s.store.GetTotalRevenue(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get total revenue: %w", err)
	}

	report := &domain.PaymentSummaryReport{
		ByStatus:     byStatus,
		TotalRevenue: domain.NumericToFloat64(totalRevNumeric),
	}

	return report, nil
}
