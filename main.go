// © 2025 Vlad-Stefan Harbuz <vlad@vlad.website>
//
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"crypto/md5"
	"encoding/csv"
	"encoding/hex"
	"fmt"
	"github.com/stripe/stripe-go/v82"
	"github.com/stripe/stripe-go/v82/checkout/session"
	"html/template"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"
)

const LISTEN_ADDR = "localhost:3003"
var WEBSITE_HOST string = os.Getenv("WEBSITE_HOST")
var STRIPE_SECRET_KEY = os.Getenv("STRIPE_SECRET_KEY")
const BANK_DETAILS_TEMPLATE = `
<p>
	Please send a bank transfer using the following details. Make sure you
	include the reference, so we can identify your payment.
</p>

<dl>
	<dt>Beneficiary name</dt>
	<dd>Open Source Endowment Foundation</dd>
	<dt>Beneficiary address</dt>
	<dd>1209 Orange Street, Wilmington, DE 19801, USA</dd>
	<dt>Bank name</dt>
	<dd>Choice Financial Group</dd>
	<dt>Bank address</dt>
	<dd>4501 23rd Avenue S, Fargo, ND 58104, USA</dd>
	<dt>ABA routing number</dt>
	<dd>091311229</dd>
	<dt>Account number</dt>
	<dd>202580213394</dd>
	<dt>Type of account</dt>
	<dd>Checking</dd>
	<dt>SWIFT/BIC code</dt>
	<dd>CHFGUS44021</dd>
	<dt>IBAN/account number</dt>
	<dd>202580213394</dd>
	<dt>Amount</dt>
	<dd>{{ .Amount }} USD</dd>
	<dt>Reference</dt>
	<dd>{{ .Reference }}</dd>
</dl>
`

type BankDetailsParams struct {
	Amount string
	Reference string
}

type DonationMethod string
const (
	MethodStripe DonationMethod = "stripe"
	MethodBank = "bank"
)

type Donor struct {
	Email string
	Name string
	Amount int64
	Method DonationMethod
	Reference string
	Time time.Time
}

func main() {
	stripe.Key = STRIPE_SECRET_KEY
	http.HandleFunc("/create-checkout-session", createCheckoutSession)
	http.HandleFunc("/create-bank-details", createBankDetails)
	http.HandleFunc("/record-bank-transfer", recordBankTransfer)
	log.Printf("Listening on %s", LISTEN_ADDR)
	log.Fatal(http.ListenAndServe(LISTEN_ADDR, nil))
}

func enableCors(w *http.ResponseWriter) {
	(*w).Header().Set("Access-Control-Allow-Origin", "*")
}

func redirectToCheckoutMessage(w http.ResponseWriter, r *http.Request, message string) {
	http.Redirect(w, r, fmt.Sprintf("%s?checkoutMessage=%s#donate", WEBSITE_HOST, message), http.StatusSeeOther)
}

func redirectToThankYou(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, fmt.Sprintf("%s/thank-you", WEBSITE_HOST), http.StatusSeeOther)
}

func logDonor(donor Donor) {
	f, err := os.OpenFile("donors.csv", os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		log.Printf("os.OpenFile: %v", err)
		return
	}
	w := csv.NewWriter(f)
	w.Write([]string{
		donor.Time.Format("2006-01-02 15:04:05"),
		donor.Email,
		donor.Name,
		strconv.FormatInt(donor.Amount, 10),
		string(donor.Method),
		donor.Reference,
	})
	w.Flush()
}

func createCheckoutSession(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		log.Printf("r.ParseForm:", err)
		redirectToCheckoutMessage(w, r, "An unknown error occurred. Please contact us.")
		return
	}

	donorEmail := r.PostFormValue("donorEmail")
	donorName := r.PostFormValue("donorName")
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
		CustomerEmail: stripe.String(donorEmail),
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

	logDonor(Donor{
		Name: donorName,
		Email: donorEmail,
		Amount: unitAmount,
		Method: MethodStripe,
		Reference: "",
		Time: time.Now(),
	})

	http.Redirect(w, r, s.URL, http.StatusSeeOther)
}

func createBankDetails(w http.ResponseWriter, r *http.Request) {
	enableCors(&w)
	err := r.ParseForm()
	if err != nil {
		log.Printf("r.ParseForm:", err)
		redirectToCheckoutMessage(w, r, "An unknown error occurred. Please contact us.")
		return
	}

	donorEmail := r.PostFormValue("donorEmail")
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

	h := md5.New()
	h.Write([]byte(donorEmail))
	h.Write([]byte(unitAmountString))
	reference := hex.EncodeToString(h.Sum(nil))[:6]

	t, err := template.New("bank-details").Parse(BANK_DETAILS_TEMPLATE)
	if err != nil {
		redirectToCheckoutMessage(w, r, "An unknown error occurred. Please contact us.")
		return
	}

	t.Execute(w, BankDetailsParams{
		Amount: unitAmountString,
		Reference: reference,
	})
}

func recordBankTransfer(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		log.Printf("r.ParseForm:", err)
		redirectToCheckoutMessage(w, r, "An unknown error occurred. Please contact us.")
		return
	}

	donorEmail := r.PostFormValue("donorEmail")
	donorName := r.PostFormValue("donorName")
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

	h := md5.New()
	h.Write([]byte(donorEmail))
	h.Write([]byte(unitAmountString))
	reference := hex.EncodeToString(h.Sum(nil))[:6]

	logDonor(Donor{
		Name: donorName,
		Email: donorEmail,
		Amount: unitAmount,
		Method: MethodBank,
		Reference: reference,
		Time: time.Now(),
	})

	redirectToThankYou(w, r)
}
