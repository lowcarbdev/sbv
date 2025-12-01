# SMS Backup Viewer (SBV)

A modern web application for viewing SMS and MMS message backups. Import your messages from "SMS Backup & Restore" XML files and browse them in a texting app-like interface.

![conversation view](docs/conversation-view.png "SBV Conversation View")

## Quick Start

### Docker

Run the latest stable version:
```bash
docker run -d \
  -p 8081:8081 \
  -v $(pwd)/data:/data \
  -e PUID=1000 \
  -e PGID=1000 \
  ghcr.io/lowcarbdev/sbv:stable
```

### Docker Compose

```yaml
services:
  sbv:
    image: ghcr.io/lowcarbdev/sbv:stable
    ports:
      - "8081:8081"
    volumes:
      - ./data:/data
    environment:
      - PUID=1000
      - PGID=1000
    restart: unless-stopped
```

## Features

- **Multi-user** - Create a username/password to log in
- **Import SMS Backup & Restore XML** - Upload XML files from the web interface.
- **Idempotent imports** - Upload the same XML file without duplicates.
- **Tested with large backups** - Works with multi-GB backups
- **SMS, MMS, and call logs support** - Read all types of call and message records.
- **Inline image and video** - View images or watch videos as you browse. Even works with Apple HEIC and 3gp videos.
- **Fast conversation filtering** - Skip to the right conversation.
- **Full-text search** - Find what you want fast.
- **Activity view** - See it as it happened.
- **vCard preview** - Preview the contents of contact cards (vCards)
- **Mobile view** - UI works on both desktop and mobile

## Tech Stack

- **Backend**: Go with SQLite database
- **Frontend**: React with Vite and Bootstrap CSS
- **Database**: SQLite (stores messages, including media as BLOBs)

## Environment Variables

- `PUID` - User ID to run the application as (default: `1000`)
- `PGID` - Group ID to run the application as (default: `1000`)

**Note on PUID/PGID**: Setting these to match your host user ensures that files created in the mounted volume have the desired permissions. Find your UID/GID with `id -u` and `id -g`.

## Data Persistence

The Docker setup uses a bind mount to persist the database:
- Host path: `./data/sbv*.db`
- Container path: `/data/sbv*.db`

This ensures your data survives container restarts and updates.

One sqlite database is created per user.

## License

MIT

## Contributing

- Please submit any issues to github issues
- PRs are welcome, but for anything over ~100 lines, please submit a github discussion first

## FAQ

Q: What backups does this program support?

XML backups from the [SMS Backup & Restore app](https://play.google.com/store/apps/details?id=com.riteshsahu.SMSBackupRestore&hl=en_US). Android devices are supported. iPhone (iOS) devices are not supported by SMS Backup & Restore.

Q: What media and attachments can be previewed?

SBV supports most images formats (jpg, png, gif, heic), video formats (mp4, 3gp), audio (mp4). Contact cards (aka vCard or VCF) are supported.

Q: Why do I only see calls or messages, but not both?

SMS Backup & Restore creates two separate backup files, beginning with `calls-` and `sms-`. Make sure you import both to see the complete picture.

Q: Can I save just the sqlite db and delete the XML files?

The sqlite database doesn't save all information that is produced by the XML file. It is recommended to retain the XML file as your backup, then import into this app as needed.

Q: Does this app keep my messages private?

SBV keeps your messages 100% private. SBV does not send any telemetry. SBV does not communicate with remote servers at all. It is highly recommended to use a strong password to secure access to your data.

Do not expose SBV directly to the internet. While SBV is secure, that's just asking for trouble.

Because SBV gathers no telemetry, please star the project on github to show your support.

## Known Issues

- Imports are somewhat slow for large imports on Linux, especially when media is present
- In group MMS, the sender label only shows the phone number (not contact name) because the contact name is not available in the XML file
- There is currently a 100k message limit per conversation. To see older messages, filter by date.

## Screenshots
![login](docs/login.png "Login")
![upload backup](docs/upload-backup.png "Upload Backup")
![upload complete](docs/upload-complete.png "Upload Complete")
![conversation view](docs/conversation-view.png "Conversation View")
![search view](docs/search-view.png "Search View")
![activity view](docs/activity-view.png "Activity View")
![inline images](docs/inline-images.png "Inline Image")
