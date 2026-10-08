package main

import (
	"context"
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	db "github.com/rahadianap/community-waste-api/db/sqlc"
	"github.com/rahadianap/community-waste-api/internal/config"
	"github.com/rahadianap/community-waste-api/internal/domain"
	"github.com/rahadianap/community-waste-api/pkg/database"
)

func main() {
	log.Println("Running database seeder...")

	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	pool, err := database.NewPostgresPool(ctx, cfg.DatabaseURL())
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer pool.Close()

	queries := db.New(pool)

	// 1. Seed Households
	log.Println("Seeding Households...")
	h1, err := queries.CreateHousehold(ctx, db.CreateHouseholdParams{
		OwnerName: "Budi Santoso",
		Address:   "Jl. Merdeka No. 10, RT 01/RW 02",
	})
	if err != nil {
		log.Fatalf("Failed to seed household 1: %v", err)
	}

	h2, err := queries.CreateHousehold(ctx, db.CreateHouseholdParams{
		OwnerName: "Siti Rahmawati",
		Address:   "Jl. Mawar No. 45, RT 03/RW 02",
	})
	if err != nil {
		log.Fatalf("Failed to seed household 2: %v", err)
	}

	h3, err := queries.CreateHousehold(ctx, db.CreateHouseholdParams{
		OwnerName: "Ahmad Fauzi",
		Address:   "Jl. Diponegoro No. 88, RT 05/RW 04",
	})
	if err != nil {
		log.Fatalf("Failed to seed household 3: %v", err)
	}

	// 2. Seed Pickups
	log.Println("Seeding Waste Pickups...")
	// Pickup 1: Organic for Budi - scheduled
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

	// Pickup 2: Plastic for Budi - completed
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

	// Pickup 3: Electronic for Siti - pending, safety_check: true (ready to schedule)
	_, _ = queries.CreatePickup(ctx, db.CreatePickupParams{
		HouseholdID: h2.ID,
		Type:        db.WasteTypeElectronic,
		SafetyCheck: true,
	})

	// Pickup 4: Paper for Siti - pending
	_, _ = queries.CreatePickup(ctx, db.CreatePickupParams{
		HouseholdID: h2.ID,
		Type:        db.WasteTypePaper,
		SafetyCheck: false,
	})

	// Pickup 5: Electronic for Ahmad - pending, safety_check: false (testing BR-03 rejection)
	p5, _ := queries.CreatePickup(ctx, db.CreatePickupParams{
		HouseholdID: h3.ID,
		Type:        db.WasteTypeElectronic,
		SafetyCheck: false,
	})

	// 3. Seed Payments
	log.Println("Seeding Payments...")
	amount50k, _ := domain.Float64ToNumeric(50000.00)
	amount100k, _ := domain.Float64ToNumeric(100000.00)

	// Payment 1: Paid payment for Budi's completed plastic pickup
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

	// Payment 2: Pending payment for Ahmad (testing BR-01: Ahmad cannot create new pickup)
	_, _ = queries.CreatePayment(ctx, db.CreatePaymentParams{
		HouseholdID: h3.ID,
		WasteID:     p5.ID,
		Amount:      amount100k,
		Status:      db.PaymentStatusPending,
	})

	log.Println("Database seeded successfully with realistic test records!")
	log.Printf("Seeded 3 Households: %s, %s, %s\n", domain.UUIDToString(h1.ID), domain.UUIDToString(h2.ID), domain.UUIDToString(h3.ID))
}
