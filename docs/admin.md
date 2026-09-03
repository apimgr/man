# Admin Panel

## Access

- URL: `/admin`
- First-run setup wizard creates admin account
- Session-based authentication

## First-Run Setup

On first run, a setup token is displayed in the server logs:

```
Setup token: abc123-def456-ghi789
Visit: http://localhost:64580/admin/setup
```

Use this token to create the primary admin account.

## Features

### Dashboard

- Server status and health
- Man page statistics
- Recent activity

### Server Settings

- Port and address configuration
- SSL/TLS management
- Mode (production/development)

### Database

- View statistics
- Rebuild search index
- Backup/restore

### Backup & Restore

- Create encrypted backups
- Restore from backup
- Scheduled automatic backups

### SSL/TLS

- Let's Encrypt integration
- Custom certificate upload
- Auto-renewal management

### Monitoring

- Server metrics
- Request logs
- Error tracking

## Admin API

Programmatic access via `/api/v1/admin/` with bearer token authentication.

```bash
curl -H "Authorization: Bearer adm_xxxxx" \
  http://localhost:64580/api/v1/admin/stats
```
