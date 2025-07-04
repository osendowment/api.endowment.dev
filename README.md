<!--
© 2025 Vlad-Stefan Harbuz <vlad@vlad.website>
SPDX-License-Identifier: Apache-2.0
-->
# Open Source Endowment API

This is the API for the [Open Source Endowment][endowment] website. It currently handles the payment form.

## Running

To run locally:

```
API_HOST="http://localhost:3003" \
WEBSITE_HOST="http://localhost:4321" \
STRIPE_SECRET_KEY="..." \
MERCURY_API_TOKEN="..." \
RESEND_API_KEY="..." \
DATABASE_URL="..." \
USE_CORS="true" \
go run .
```

## Authorship

This repository is managed by [Vlad-Stefan Harbuz][vladh].

This code is open source — licensing information is included with each file, or in [REUSE.toml](REUSE.toml).

[endowment]: https://endowment.dev
[vladh]: https://vlad.website
