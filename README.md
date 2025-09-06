# volback

CLI & libraries to backup data from various locations.

# Usage

```bash
volback \
	--src.kind="fs" \
	--src.path="$backup_path" \
	--enc.key="${BACKUP_ENCRYPTION_KEY}"  \
	--dst.kind="s3" \
	--dst.path="backups/vw" \
	--dst.s3-endpoint="${BACKUP_DEST_ENDPOINT}" \
	--dst.s3-bucket="${BACKUP_DEST_BUCKET}" \
	--dst.s3-access-key-id="${BACKUP_DEST_ACCESS_KEY_ID}" \
	--dst.s3-secret-access-key="${BACKUP_DEST_SECRET_ACCESS_KEY}" \
	--dst.s3-region="us-east-1"
```

# Development

See DEVELOPMENT.md
