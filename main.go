// © 2025 Vlad-Stefan Harbuz <vlad@vlad.website>
//
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"fmt"
	"github.com/stripe/stripe-go/v82"
	"github.com/stripe/stripe-go/v82/checkout/session"
	"log"
	"net/http"
	"os"
	"strconv"
)

var LISTEN_ADDR string = "localhost:3003"
var WEBSITE_HOST string = os.Getenv("WEBSITE_HOST")
var STRIPE_SECRET_KEY = os.Getenv("STRIPE_SECRET_KEY")

func main() {
	stripe.Key = STRIPE_SECRET_KEY
	http.HandleFunc("/create-checkout-session", createCheckoutSession)
	log.Printf("Listening on %s", LISTEN_ADDR)
	log.Fatal(http.ListenAndServe(LISTEN_ADDR, nil))
}

func redirectToCheckoutMessage(w http.ResponseWriter, r *http.Request, message string) {
	http.Redirect(w, r, fmt.Sprintf("%s?checkoutMessage=%s#donate", WEBSITE_HOST, message), http.StatusSeeOther)
}

func createCheckoutSession(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		log.Printf("r.ParseForm:", err)
		redirectToCheckoutMessage(w, r, "An unknown error occurred. Please contact us.")
		return
	}

	customerEmail := r.PostFormValue("customerEmail")
	// customerName := r.PostFormValue("customerName")
	unitAmountString := r.PostFormValue("presetAmount")
	if unitAmountString == "custom" {
		unitAmountString = r.PostFormValue("customAmount")
	}
	unitAmount, err := strconv.ParseInt(unitAmountString, 10, 64)
	if err != nil || unitAmount <= 0 {
		log.Printf("strconv.Atoi:", err)
		redirectToCheckoutMessage(w, r, "Invalid amount.")
		return
	}

	params := &stripe.CheckoutSessionParams{
		CustomerEmail: stripe.String(customerEmail),
		SuccessURL:    stripe.String(WEBSITE_HOST + "/thank-you?session_id={CHECKOUT_SESSION_ID}"),
		CancelURL:     stripe.String(WEBSITE_HOST),
		LineItems: []*stripe.CheckoutSessionLineItemParams{
			&stripe.CheckoutSessionLineItemParams{
				PriceData: &stripe.CheckoutSessionLineItemPriceDataParams{
					Currency: stripe.String("usd"),
					ProductData: &stripe.CheckoutSessionLineItemPriceDataProductDataParams{
						Name: stripe.String("Open Source Endowment Donation"),
					},
					UnitAmount: stripe.Int64(unitAmount * 100),
				},
				Quantity: stripe.Int64(1),
			},
		},
		Mode:       stripe.String(string(stripe.CheckoutSessionModePayment)),
		SubmitType: stripe.String("donate"),
	}
	s, err := session.New(params)

	if err != nil {
		log.Printf("session.New: %v", err)
		if stripeErr, ok := err.(*stripe.Error); ok {
			redirectToCheckoutMessage(w, r, stripeErr.Msg)
			return
		}
		redirectToCheckoutMessage(w, r, "An unknown error occurred. Please contact us.")
		return
	}

	http.Redirect(w, r, s.URL, http.StatusSeeOther)
}
