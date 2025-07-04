// © 2025 Vlad-Stefan Harbuz <vlad@vlad.website>
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"fmt"
	"github.com/davecgh/go-spew/spew"
	"log"
	"os"
	"time"
)

const MERCURY_DATE_FORMAT = "2006-01-02T15:04:05.999Z"

type MercuryTransaction struct {
	BankDescription *string `json:"bankDescription"`
	Amount          float64 `json:"amount"`
}

type MercuryTransactionList struct {
	Transactions []MercuryTransaction `json:"transactions"`
}

func getMercuryUrlForTransactions(accountId string, start *time.Time) string {
	if start != nil {
		startString := start.Format(MERCURY_DATE_FORMAT)
		return fmt.Sprintf(
			"https://api.mercury.com/api/v1/account/%s/transactions?start=%s",
			accountId, startString)
	} else {
		return fmt.Sprintf(
			"https://api.mercury.com/api/v1/account/%s/transactions",
			accountId)
	}
}

func syncMercury() {
	bankAccounts, err := getMercuryBankAccounts()
	if err != nil {
		log.Printf("Failed to get Mercury bank accounts: %v", err)
		return
	}

	for _, bankAccount := range bankAccounts {
		url := getMercuryUrlForTransactions(
			bankAccount.Id, bankAccount.LastSyncedAt)
		transactionList := MercuryTransactionList{}
		err := getJson(url, &transactionList, MERCURY_API_TOKEN)
		if err != nil {
			log.Printf("Failed to get Mercury transaction list: %v", err)
			return
		}

		for _, transaction := range transactionList.Transactions {
			if transaction.Amount <= 0 || transaction.BankDescription == nil {
				continue
			}
			spew.Fdump(os.Stderr, transaction)
			bankReference := BANK_REFERENCE_REGEXP.FindString(
				*transaction.BankDescription)
			if len(bankReference) == 0 {
				continue
			}

			existingDonation, err := getDonationForBankReference(bankReference)
			if err != nil {
				log.Printf("Couldn't query for donation: %v", err)
				return
			}
			if existingDonation.BankReference != nil {
				log.Printf(
					"We have just seen a bank transaction with bank reference %s, but this bank reference is already known in the donation table",
					bankReference)
				sendDuplicateDonationEmail(Donation{
					UsdCentAmount: int64(transaction.Amount * 100),
					DonationMethod: "bank",
					BankReference: existingDonation.BankReference,
					Donor: existingDonation.Donor,
				})
				continue
			}

			donationAttempt, err := getDonationAttemptForBankReference(
				bankReference)
			if err != nil {
				log.Printf("Couldn't query for donation attempt: %v", err)
				return
			}
			if donationAttempt.BankReference == nil {
				log.Printf(
					"We have just seen a bank transaction with bank reference %s, but this bank reference has not been previously seen because it is not recorded in the donation_attempt table",
					bankReference)
				sendUnexpectedDonationEmail(Donation{
					UsdCentAmount: int64(transaction.Amount * 100),
					DonationMethod: "bank",
					BankReference: &bankReference,
				})
				continue
			}

			newDonor := Donor{
				Email: donationAttempt.Email,
				Name:  donationAttempt.Name,
			}
			newDonation := Donation{
				UsdCentAmount:  donationAttempt.UsdCentAmount,
				DonationMethod: "bank",
				BankReference:  &bankReference,
			}
			err = saveDonation(newDonor, newDonation)
			if err != nil {
				log.Printf("Couldn't save donation: %v", err)
			}
			sendNewDonationEmail(newDonor, newDonation)
		}

		updateBankAccountSyncTime(
			bankAccount.Bank, bankAccount.Id, time.Now())
	}
}
