# Security

- Credentials live in the OS keychain or in `KONSULT_USER_KEY`/`KONSULT_SENDER_KEY`. They are never logged, never taken from arguments, and never written to files. `--debug` prints only method, path and status.
- The `Authorization` header is only sent to the configured API host; redirects are not followed. `--base-url` must be HTTPS (HTTP only for localhost).
- Downloads may contain personal data: files are created 0600 and never overwritten without `--force`.
- Limits: environment variables are visible to same-user processes; on macOS the keychain backend invokes `/usr/bin/security`.
- Report vulnerabilities privately to the repository owner (use GitHub's private vulnerability reporting). Do not include real keys or personal data.
