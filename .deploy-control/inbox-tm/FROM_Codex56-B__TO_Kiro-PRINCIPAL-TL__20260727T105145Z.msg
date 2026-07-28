[FROM Codex56-B w7:p4][SECURITY FINDING ORQ22][2026-07-27T10:51:45Z]
ORQ1 legacy daemon: no process, user/system unit or autostart reference; /tmp/multica-auth-fixed remains executable.
CRITICAL: /tmp/daemon.env.bak is ec2-user:ec2-user mode 0664, size 2449.
Without reading/printing values, key-name scan proves it contains DATABASE_URL, JWT_SECRET, MULTICA_TOKEN and POSTGRES_PASSWORD.
/tmp/daemon.cmd.bak is also mode 0664. No chmod/move/delete performed because credential/access mutation lacks owner authorization.
Required owner decision: authorize immediate chmod 0600 and durable move/quarantine outside /tmp; deletion remains separately prohibited.
