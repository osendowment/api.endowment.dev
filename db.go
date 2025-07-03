// © 2025 Vlad-Stefan Harbuz <vlad@vlad.website>
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"log"
	"time"
)

type BankAccount struct {
	Bank         string     `json:"bank"`
	Id           string     `json:"id"`
	LastSyncedAt *time.Time `json:"last_synced_at"`
	Nickname     *string    `json:"nickname"`
}

type DonationAttempt struct {
	Email          string
	Name           *string
	UsdCentAmount  int64
	DonationMethod string
	BankReference  *string
}

type Donor struct {
	Email string
	Name  *string
}

type Donation struct {
	UsdCentAmount  int64
	DonationMethod string
	BankReference  *string
	Donor          *string
}

func getDb() *pgxpool.Pool {
	dbpool, err := pgxpool.New(context.Background(), DATABASE_URL)
	if err != nil {
		log.Printf("Failed to connect to the database: %v", err)
	}
	return dbpool
}

func saveDonationAttempt(attempt DonationAttempt) {
	dbpool := getDb()
	defer dbpool.Close()
	_, err := dbpool.Exec(context.Background(), `
		INSERT INTO donation_attempt
			(email, name, usd_cent_amount, donation_method, bank_reference)
		VALUES
			($1, $2, $3, $4, $5)
		`,
		attempt.Email, attempt.Name, attempt.UsdCentAmount, attempt.DonationMethod, attempt.BankReference)
	if err != nil {
		log.Printf("Failed to execute query: %v", err)
	}
}

func saveDonation(donor Donor, donation Donation) error {
	dbpool := getDb()
	defer dbpool.Close()

	_, err := dbpool.Exec(context.Background(), `
		INSERT INTO donor
			(email, name)
		VALUES
			($1, $2)
		ON CONFLICT (email) DO UPDATE SET
			name = EXCLUDED.name
		`,
		donor.Email, donor.Name)
	if err != nil {
		log.Printf("Failed to execute query: %v", err)
		return err
	}

	_, err = dbpool.Exec(context.Background(), `
		INSERT INTO donation
			(donor, usd_cent_amount, donation_method, bank_reference)
		VALUES
			($1, $2, $3, $4)
		`,
		donor.Email, donation.UsdCentAmount, donation.DonationMethod, donation.BankReference)
	if err != nil {
		log.Printf("Failed to execute query: %v", err)
		return err
	}

	return nil
}

func getMercuryBankAccounts() ([]BankAccount, error) {
	bankAccounts := []BankAccount{}
	dbpool := getDb()
	defer dbpool.Close()

	rows, err := dbpool.Query(context.Background(), `
		SELECT id, last_synced_at, nickname
		FROM bank_account
		WHERE bank = 'mercury'
		`)
	if err != nil {
		return bankAccounts, err
	}

	var id string
	var last_synced_at *time.Time
	var nickname *string
	_, err = pgx.ForEachRow(
		rows,
		[]any{&id, &last_synced_at, &nickname},
		func() error {
			bankAccounts = append(bankAccounts, BankAccount{
				Bank:         "mercury",
				Id:           id,
				LastSyncedAt: last_synced_at,
				Nickname:     nickname,
			})
			return nil
		},
	)

	return bankAccounts, err
}

func updateBankAccountSyncTime(
	bank string,
	id string,
	lastSyncedAt time.Time,
) error {
	dbpool := getDb()
	defer dbpool.Close()

	_, err := dbpool.Exec(context.Background(), `
		UPDATE bank_account
		SET last_synced_at = $1
		WHERE bank = $2 AND id = $3
		`,
		lastSyncedAt.Format(MERCURY_DATE_FORMAT), bank, id)
	if err != nil {
		log.Printf("Failed to execute query: %v", err)
	}
	return err
}

func getDonationAttemptForBankReference(
	bankReference string,
) (DonationAttempt, error) {
	donationAttempt := DonationAttempt{}
	dbpool := getDb()
	defer dbpool.Close()

	rows, err := dbpool.Query(context.Background(), `
		SELECT email, name, usd_cent_amount
		FROM donation_attempt
		WHERE bank_reference = $1
		`, bankReference)
	if err != nil {
		return donationAttempt, err
	}

	var email string
	var name *string
	var usdCentAmount int64
	_, err = pgx.ForEachRow(
		rows,
		[]any{&email, &name, &usdCentAmount},
		func() error {
			donationAttempt = DonationAttempt{
				Email:          email,
				Name:           name,
				UsdCentAmount:  usdCentAmount,
				DonationMethod: "bank",
				BankReference:  &bankReference,
			}
			return nil
		},
	)

	return donationAttempt, err
}

func getDonationForBankReference(
	bankReference string,
) (Donation, error) {
	donation := Donation{}
	dbpool := getDb()
	defer dbpool.Close()

	rows, err := dbpool.Query(context.Background(), `
		SELECT usd_cent_amount, donor
		FROM donation
		WHERE bank_reference = $1
		`, bankReference)
	if err != nil {
		return donation, err
	}

	var usdCentAmount int64
	var donor *string
	_, err = pgx.ForEachRow(
		rows,
		[]any{&usdCentAmount, &donor},
		func() error {
			donation = Donation{
				UsdCentAmount:  usdCentAmount,
				DonationMethod: "bank",
				BankReference:  &bankReference,
				Donor:          donor,
			}
			return nil
		},
	)

	return donation, err
}
