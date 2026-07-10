# Deploying qrsurvey

1. Build a static binary for the target host:
   ```
   CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o qrsurvey ./cmd/qrsurvey
   ```
2. On the server, create `/opt/qrsurvey/` and copy the `qrsurvey` binary there.
3. Generate an admin password hash and fill in `/etc/qrsurvey.env` from
   `deploy/qrsurvey.env.example`:
   ```
   go run ./cmd/hashpw 'your-password-here'
   ```
4. Install `deploy/qrsurvey.service` to `/etc/systemd/system/`, then:
   ```
   systemctl daemon-reload
   systemctl enable --now qrsurvey
   ```
5. Point Caddy at it: copy `Caddyfile` (with the real domain filled in) to
   wherever your Caddy install reads its config, then `systemctl reload caddy`
   (or `caddy reload`).
6. Add `deploy/backup.sh` to cron for daily SQLite snapshots (see the script
   header for the crontab line).

Redeploys: rebuild, copy the new binary over the old one, `systemctl restart
qrsurvey`. Migrations in `internal/db/migrations/` run automatically on
startup.
