package service

import (
	"context"

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
		return nil, wrap(err, "gagal memuat data waste summary report")
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
		return nil, wrap(err, "gagal memuat data payment summary report")
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
		return nil, wrap(err, "gagal mendapatkan total revenue")
	}

	report := &domain.PaymentSummaryReport{
		ByStatus:     byStatus,
		TotalRevenue: domain.NumericToFloat64(totalRevNumeric),
	}

	return report, nil
}
