# © 2025 Vlad-Stefan Harbuz <vlad@vlad.website>
# SPDX-License-Identifier: Apache-2.0

# This file deploys to a machine named “yavin” that only Vlad has access to.

.PHONY: publish-prod

bin/api.endowment.dev: main.go
	go build -o bin/ .

publish-prod: bin/api.endowment.dev
	rsync -Pvrthl --delete --exclude .git --exclude donors.csv --info=progress2 ./ yavin:/srv/www/dev.endowment.api
	ssh yavin 'doas systemctl restart api.endowment.dev'
