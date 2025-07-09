# Go Chat

A real-time chat application built with Go, Redis, and WebSocket.

## Features

- User authentication with JWT
- Private messaging
- Group chats
- Broadcast messages
- Real-time communication using WebSocket
- Message history persistence in Redis


## Getting Started

1. Clone the repository:
```bash
git clone github.com/bm-197/go-chat
cd go-chat
```

2. Create a `.env` file in the root directory with the following content:
```
REDIS_HOST=redis_host
REDIS_PORT=6379
GO_ENV=development
APP_PORT=5000
REDIS_COMMANDER_PORT=8081
JWT_SECRET=your-super-secret-key-change-in-production
```

3. Start the application using Docker Compose:
```bash
docker compose up
```

The application will be available at:
- API: http://localhost:5000

## API Endpoints

* NOTE: All end points require the bearer token in the request header
- `header: { Authorization: Bearer <jwt-token>
}`

### Authentication
- `POST /api/register` - Register a new user
- `POST /api/login` - Login and get JWT token

### User
- `GET /api/profile` - Get user profile

### Groups
- `POST /api/groups` - Create a new group
- `GET /api/groups` - List user's groups
- `GET /api/groups/:id` - Get group details
- `POST /api/groups/:id/members` - Add member to group
- `DELETE /api/groups/:id/members/:memberID` - Remove member from group
- `DELETE /api/groups/:id` - Delete group

### Messages
- `POST /api/messages` - Send a message
- `GET /api/messages/private/:userID` - Get private messages with user
- `GET /api/messages/group/:groupID` - Get group messages
- `GET /api/messages/broadcast` - Get broadcast messages

### WebSocket
- `GET /api/ws` - WebSocket endpoint for real-time messaging

## WebSocket Message Format

```json
{
  "type": "message",
  "payload": {
    "type": "private|group|broadcast",
    "content": "message content",
    "to_user": "user_id",  // for private messages
    "to_group": "group_id" // for group messages
  }
}
```


## Production

For production deployment:
1. Update the JWT_SECRET in .env
2. Set GO_ENV=production in .env