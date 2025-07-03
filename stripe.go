// © 2025 Vlad-Stefan Harbuz <vlad@vlad.website>
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"errors"
	"github.com/stripe/stripe-go/v82"
	"github.com/stripe/stripe-go/v82/checkout/session"
	"log"
)

func makeStripeCheckoutSession(
	donorName string,
	donorEmail string,
	usdCentAmount int64,
	usdCentAmountString string,
) (*stripe.CheckoutSession, error) {
	stripe.Key = STRIPE_SECRET_KEY
	params := &stripe.CheckoutSessionParams{
		CustomerEmail: stripe.String(donorEmail),
		Metadata: map[string]string{
			"donation_form_name":   donorName,
			"donation_form_email":  donorEmail,
			"donation_form_amount": usdCentAmountString,
		},
		SuccessURL: stripe.String(API_HOST + "/stripe-success?session_id={CHECKOUT_SESSION_ID}"),
		CancelURL:  stripe.String(WEBSITE_HOST),
		LineItems: []*stripe.CheckoutSessionLineItemParams{
			&stripe.CheckoutSessionLineItemParams{
				PriceData: &stripe.CheckoutSessionLineItemPriceDataParams{
					Currency: stripe.String("usd"),
					ProductData: &stripe.CheckoutSessionLineItemPriceDataProductDataParams{
						Name: stripe.String("Open Source Endowment Donation"),
					},
					UnitAmount: &usdCentAmount,
				},
				Quantity: stripe.Int64(1),
			},
		},
		Mode:       stripe.String(string(stripe.CheckoutSessionModePayment)),
		SubmitType: stripe.String("donate"),
	}
	s, err := session.New(params)
	if err != nil {
		if stripeErr, ok := err.(*stripe.Error); ok {
			err = errors.New(stripeErr.Msg)
		} else {
			err = errors.New("An unknown error occurred. Please contact us.")
		}
	}
	return s, err
}

func saveStripeSuccessfulCheckout(sessionId string) (string, error) {
	sc := stripe.NewClient(STRIPE_SECRET_KEY)
	session, err := sc.V1CheckoutSessions.Retrieve(context.TODO(), sessionId, nil)
	if err != nil {
		log.Printf("CheckoutSessions.Retrieve: %s", err)
		err = errors.New(
			"An error occurred when retrieving your donation information. Please contact us.")
		return "", err
	}

	donorEmail := session.Metadata["donation_form_email"]
	donorName := session.Metadata["donation_form_name"]

	newDonor := Donor{
		Email: donorEmail,
		Name:  &donorName,
	}
	newDonation := Donation{
		UsdCentAmount:  session.AmountTotal,
		DonationMethod: "stripe",
		BankReference:  nil,
	}
	err = saveDonation(newDonor, newDonation)
	if err != nil {
		log.Printf("saveDonation: %s", err)
		err = errors.New(
			"Your donation was successful, but an error occurred while recording its details. Please contact us.")
	}
	go func() {
		sendNewDonationEmail(newDonor, newDonation)
	}()

	return donorName, err
}
