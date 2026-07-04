# CAO CORS Configuration

For local AgentVerse development, run the SPA on `http://localhost:5173` and configure CAO with these environment variables exactly:

```env
CAO_CORS_ORIGINS=http://localhost:5173
CAO_ALLOWED_HOSTS=127.0.0.1,localhost
CAO_WS_ALLOWED_CLIENTS=http://localhost:5173
```

Set them in the CAO process environment before starting the CAO FastAPI server. Each variable is a comma-separated allow-list; do not wrap individual values in quotes inside the value.

PowerShell example:

```powershell
$env:CAO_CORS_ORIGINS = 'http://localhost:5173'
$env:CAO_ALLOWED_HOSTS = '127.0.0.1,localhost'
$env:CAO_WS_ALLOWED_CLIENTS = 'http://localhost:5173'
```

With these values, the Vite dev server on `http://localhost:5173` can call CAO REST endpoints on `http://127.0.0.1:9889` and open terminal WebSocket connections.