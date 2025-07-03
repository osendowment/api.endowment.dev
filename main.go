// © 2025 Vlad-Stefan Harbuz <vlad@vlad.website>
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
)

const LISTEN_ADDR = "localhost:3003"

var API_HOST string = os.Getenv("API_HOST")
var WEBSITE_HOST string = os.Getenv("WEBSITE_HOST")
var STRIPE_SECRET_KEY = os.Getenv("STRIPE_SECRET_KEY")
var MERCURY_API_TOKEN = os.Getenv("MERCURY_API_TOKEN")
var RESEND_API_KEY string = os.Getenv("RESEND_API_KEY")
var DATABASE_URL string = os.Getenv("DATABASE_URL")
var USE_CORS = len(os.Getenv("USE_CORS")) > 0

func main() {
	http.HandleFunc("/stripe-checkout-session", routeStripeCheckoutSession)
	http.HandleFunc("/stripe-success", routeStripeSuccess)
	http.HandleFunc("/bank-details", routeBankDetails)
	http.HandleFunc("/bank-account-sync", routeBankAccountSync)
	log.Printf("Listening on %s", LISTEN_ADDR)
	log.Fatal(http.ListenAndServe(LISTEN_ADDR, nil))
}

func maybeEnableCors(w *http.ResponseWriter) {
	if USE_CORS {
		(*w).Header().Set("Access-Control-Allow-Origin", "*")
	}
}

func redirectToCheckoutMessage(w http.ResponseWriter, r *http.Request, message string) {
	http.Redirect(w, r, fmt.Sprintf("%s?checkoutMessage=%s#donate", WEBSITE_HOST, message), http.StatusSeeOther)
}

func redirectToThankYou(w http.ResponseWriter, r *http.Request, donorName string) {
	http.Redirect(w, r,
		fmt.Sprintf("%s/thank-you?name=%s", WEBSITE_HOST, donorName),
		http.StatusSeeOther)
}

func routeStripeCheckoutSession(w http.ResponseWriter, r *http.Request) {
	maybeEnableCors(&w)
	err := r.ParseForm()
	if err != nil {
		log.Printf("r.ParseForm: %v", err)
		redirectToCheckoutMessage(w, r, "An unknown error occurred. Please contact us.")
		return
	}

	donorEmail := r.PostFormValue("donorEmail")
	donorName := r.PostFormValue("donorName")
	usdCentAmountString := r.PostFormValue("presetUsdAmount")
	if usdCentAmountString == "custom" {
		usdCentAmountString = r.PostFormValue("customUsdAmount")
	}
	usdCentAmountFloat, err := strconv.ParseFloat(usdCentAmountString, 64)
	if err != nil || usdCentAmountFloat <= 0 {
		log.Printf("strconv.Atoi: %v", err)
		redirectToCheckoutMessage(w, r, "Invalid amount.")
		return
	}
	usdCentAmount := int64(usdCentAmountFloat * 100)

	s, err := makeStripeCheckoutSession(
		donorName, donorEmail, usdCentAmount, usdCentAmountString)

	if err != nil {
		log.Printf("makeStripeCheckoutSession: %v", err)
		redirectToCheckoutMessage(w, r, err.Error())
		return
	}

	saveDonationAttempt(DonationAttempt{
		Email:          donorEmail,
		Name:           &donorName,
		UsdCentAmount:  usdCentAmount,
		DonationMethod: "stripe",
		BankReference:  nil,
	})

	http.Redirect(w, r, s.URL, http.StatusSeeOther)
}

func routeStripeSuccess(w http.ResponseWriter, r *http.Request) {
	sessionId := r.URL.Query().Get("session_id")

	donorName, err := saveStripeSuccessfulCheckout(sessionId)
	if err != nil {
		redirectToCheckoutMessage(w, r, err.Error())
		return
	}

	redirectToThankYou(w, r, donorName)
}

func routeBankDetails(w http.ResponseWriter, r *http.Request) {
	maybeEnableCors(&w)
	err := r.ParseForm()
	if err != nil {
		log.Printf("r.ParseForm: %v", err)
		redirectToCheckoutMessage(w, r, "An unknown error occurred. Please contact us.")
		return
	}

	donorEmail := r.PostFormValue("donorEmail")
	donorName := r.PostFormValue("donorName")
	usdCentAmountString := r.PostFormValue("presetUsdAmount")
	if usdCentAmountString == "custom" {
		usdCentAmountString = r.PostFormValue("customUsdAmount")
	}
	usdCentAmountFloat, err := strconv.ParseFloat(usdCentAmountString, 64)
	if err != nil || usdCentAmountFloat <= 0 {
		log.Printf("strconv.Atoi: %v", err)
		redirectToCheckoutMessage(w, r, "Invalid amount.")
		return
	}
	usdCentAmount := int64(usdCentAmountFloat * 100)

	bankDetails, err := createBankDetails(donorEmail, usdCentAmount)
	if err != nil {
		redirectToCheckoutMessage(w, r,
			"An unknown error occurred. Please contact us.")
		return
	}

	go func() {
		saveDonationAttempt(DonationAttempt{
			Email:          donorEmail,
			Name:           &donorName,
			UsdCentAmount:  usdCentAmount,
			DonationMethod: "bank",
			BankReference:  &bankDetails.BankReference,
		})
	}()

	io.WriteString(w, bankDetails.BankInstructions)
}

func routeBankAccountSync(w http.ResponseWriter, r *http.Request) {
	syncMercury()
}
