# Reset-Claim Guards

1. **Idempotency**: Repeated `redeem` calls for the same account must be safe and not consume multiple credits for the same reset window.
2. **Cooldown**: Minimum interval between auto-redeem attempts per account to prevent spamming the endpoint.
3. **Audit event**: Every redeem attempt must be logged (success, failure, or skipped) for full traceability.
4. **Fail-safe**: If redeem fails, the system must NOT retry aggressively (implement exponential backoff or hard circuit breaker).
