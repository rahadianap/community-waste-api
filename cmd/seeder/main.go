package main

import (
	"context"
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	db "github.com/rahadianap/community-waste-api/db/sqlc"
	"github.com/rahadianap/community-waste-api/internal/config"
	"github.com/rahadianap/community-waste-api/internal/database"
	"github.com/rahadianap/community-waste-api/internal/domain"
)

func main() {
	log.Println("Menjalankan database seeder...")

	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Gagal memuat config: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	pool, err := database.NewPostgresPool(ctx, cfg.DatabaseURL())
	if err != nil {
		log.Fatalf("Gagal terhubung ke database: %v", err)
	}
	defer pool.Close()

	queries := db.New(pool)

	log.Println("Seeding Households...")
	h1, err := queries.CreateHousehold(ctx, db.CreateHouseholdParams{
		OwnerName: "Wowo",
		Address:   "Jl. Merdeka No. 10, RT 01/RW 02",
	})
	if err != nil {
		log.Fatalf("Gagal seed household 1: %v", err)
	}

	h2, err := queries.CreateHousehold(ctx, db.CreateHouseholdParams{
		OwnerName: "Fufufafa",
		Address:   "Jl. Mawar No. 45, RT 03/RW 02",
	})
	if err != nil {
		log.Fatalf("Gagal seed household 2: %v", err)
	}

	h3, err := queries.CreateHousehold(ctx, db.CreateHouseholdParams{
		OwnerName: "Meme",
		Address:   "Jl. Diponegoro No. 88, RT 05/RW 04",
	})
	if err != nil {
		log.Fatalf("Gagal seed household 3: %v", err)
	}

	log.Println("Seeding Waste Pickups...")
	p1, err := queries.CreatePickup(ctx, db.CreatePickupParams{
		HouseholdID: h1.ID,
		Type:        db.WasteTypeOrganic,
		SafetyCheck: false,
	})
	if err == nil {
		_, _ = queries.UpdatePickupSchedule(ctx, db.UpdatePickupScheduleParams{
			ID: p1.ID,
			PickupDate: pgtype.Timestamptz{
				Time:  time.Now().Add(24 * time.Hour),
				Valid: true,
			},
		})
	}

	p2, err := queries.CreatePickup(ctx, db.CreatePickupParams{
		HouseholdID: h1.ID,
		Type:        db.WasteTypePlastic,
		SafetyCheck: false,
	})
	if err == nil {
		_, _ = queries.UpdatePickupStatus(ctx, db.UpdatePickupStatusParams{
			ID:     p2.ID,
			Status: db.PickupStatusCompleted,
		})
	}

	_, _ = queries.CreatePickup(ctx, db.CreatePickupParams{
		HouseholdID: h2.ID,
		Type:        db.WasteTypeElectronic,
		SafetyCheck: true,
	})

	_, _ = queries.CreatePickup(ctx, db.CreatePickupParams{
		HouseholdID: h2.ID,
		Type:        db.WasteTypePaper,
		SafetyCheck: false,
	})

	p5, _ := queries.CreatePickup(ctx, db.CreatePickupParams{
		HouseholdID: h3.ID,
		Type:        db.WasteTypeElectronic,
		SafetyCheck: false,
	})

	log.Println("Seeding Payments...")
	amount50k, _ := domain.Float64ToNumeric(50000.00)
	amount100k, _ := domain.Float64ToNumeric(100000.00)
	pay1, err := queries.CreatePayment(ctx, db.CreatePaymentParams{
		HouseholdID: h1.ID,
		WasteID:     p2.ID,
		Amount:      amount50k,
		Status:      db.PaymentStatusPending,
	})
	if err == nil {
		_, _ = queries.ConfirmPayment(ctx, db.ConfirmPaymentParams{
			ID:           pay1.ID,
			ProofFileUrl: pgtype.Text{String: "/uploads/payments/sample_proof.jpg", Valid: true},
		})
	}

	_, _ = queries.CreatePayment(ctx, db.CreatePaymentParams{
		HouseholdID: h3.ID,
		WasteID:     p5.ID,
		Amount:      amount100k,
		Status:      db.PaymentStatusPending,
	})

	log.Println("Database berhasil di-seed!")
}
