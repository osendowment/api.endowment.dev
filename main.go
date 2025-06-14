// © 2025 Vlad-Stefan Harbuz <vlad@vlad.website>
//
// SPDX-License-Identifier: Apache-2.0

package main

import (
    "bytes"
    "encoding/json"
    "io"
    "log"
    "net/http"
    "github.com/stripe/stripe-go/v82"
    "github.com/stripe/stripe-go/v82/checkout/session"
)

func main() {
  // This is our secret API key for the testing environment
  stripe.Key = "sk_test_51RDs2UBNfHh1TmlN4IkM1YqbBD2mirTKWwXnRi0NwWNtpaDqsSbmoExALdrfgYHqexs0ftFt66bhmxdVinfDP8Re00siikHUPQ"

  http.HandleFunc("/create-checkout-session", createCheckoutSession)
  http.HandleFunc("/session-status", retrieveCheckoutSession)

  addr := "localhost:3003"
  log.Printf("Listening on %s", addr)
  log.Fatal(http.ListenAndServe(addr, nil))
}

func createCheckoutSession(w http.ResponseWriter, r *http.Request) {
  domain := "http://localhost:4321"
  params := &stripe.CheckoutSessionParams{
    UIMode: stripe.String("embedded"),
    ReturnURL: stripe.String(domain + "/thank-you?session_id={CHECKOUT_SESSION_ID}"),
    LineItems: []*stripe.CheckoutSessionLineItemParams{
      &stripe.CheckoutSessionLineItemParams{
        Price: stripe.String("price_1RZq75BNfHh1TmlNRzCJj8Bx"),
        Quantity: stripe.Int64(1),
      },
    },
    Mode: stripe.String(string(stripe.CheckoutSessionModePayment)),
    SubmitType: stripe.String("donate"),
  }

  s, err := session.New(params)

  if err != nil {
    log.Printf("session.New: %v", err)
  }

  writeJSON(w, struct {
    ClientSecret string `json:"clientSecret"`
  }{
    ClientSecret: s.ClientSecret,
  })
}

func retrieveCheckoutSession(w http.ResponseWriter, r *http.Request) {
  s, _ := session.Get(r.URL.Query().Get("session_id"), nil)

  writeJSON(w, struct {
    Status string `json:"status"`
    CustomerEmail string `json:"customer_email"`
  }{
    Status: string(s.Status),
    CustomerEmail: string(s.CustomerDetails.Email),
  })
}

func writeJSON(w http.ResponseWriter, v interface{}) {
  var buf bytes.Buffer
  if err := json.NewEncoder(&buf).Encode(v); err != nil {
    http.Error(w, err.Error(), http.StatusInternalServerError)
    log.Printf("json.NewEncoder.Encode: %v", err)
    return
  }
  w.Header().Set("Content-Type", "application/json")
  if _, err := io.Copy(w, &buf); err != nil {
    log.Printf("io.Copy: %v", err)
    return
  }
}