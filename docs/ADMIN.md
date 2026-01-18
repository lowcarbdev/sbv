# SBV Administration Guide

This guide covers administrative features in SBV.

## User Management

### List All Users

To see all registered users along with their UUIDs and ingest directories:

Docker:
```bash
docker exec -it <container_name> /app/sbv -list-users
```

Binary:
```bash
./sbv -list-users
```

Example output:
```
USERNAME  UUID                                  INGEST DIRECTORY
--------  ----                                  ----------------
alice     46f958dc-e022-41de-b298-4ab060b5ca24  data/46f958dc-e022-41de-b298-4ab060b5ca24/ingest
bob       3cf92802-b3c5-4d38-8b11-9d34aafc7d46  data/3cf92802-b3c5-4d38-8b11-9d34aafc7d46/ingest
```

### Reset a User's Password

To reset a user's password:

Docker:
```bash
docker exec -it <container_name> /app/sbv -reset-password <username>
```

Binary:
```bash
./sbv -reset-password <username>
```

You will be prompted to enter and confirm the new password. Passwords must be at least 6 characters.

## Auto-Import

SBV can automatically import XML backup files placed in a user's ingest directory. This is useful for automated backup workflows or when you want to import files without using the web interface.

### How It Works

1. SBV scans each user's ingest directory every minute
2. When an XML file is detected and stable (not being written to), it is automatically imported
3. After successful import, the file is moved to a `complete` subdirectory
4. A `.log` file is created alongside each import with details about the process

### Ingest Directory Location

Each user has their own ingest directory based on their UUID:

```
<data_dir>/<user-uuid>/ingest/
```

Use `-list-users` to find the correct ingest directory for each user.

### Docker Volume Setup

When running with Docker, make sure the ingest directories are accessible. With the standard volume mount:

```yaml
volumes:
  - ./data:/data
```

The ingest directories will be at:
```
./data/<user-uuid>/ingest/
```

### Example Workflow

1. Find the user's ingest directory:
   ```bash
   docker exec -it sbv /app/sbv -list-users
   ```

2. Copy your backup file to the ingest directory:
   ```bash
   cp sms-backup.xml ./data/<user-uuid>/ingest/
   ```

3. SBV will automatically detect and import the file within 1 minute

4. Check the import results:
   - The XML file will be moved to `./data/<user-uuid>/complete/`
   - A log file (`sms-backup.xml.log`) will contain import details

### Supported File Types

- XML files from SMS Backup & Restore (`.xml`)

### Troubleshooting

**File not being imported:**
- Ensure the file has a `.xml` extension
- Check that the file is not still being written (SBV waits for files to be stable)
- Verify the file is in the correct user's ingest directory
- Check the application logs for errors

**Import failed:**
- Check the `.log` file in the ingest directory for error details
- The original file remains in the ingest directory for manual review if import fails
