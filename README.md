Very simple package that allows creating single-use tokens to perform different actions. 
Redis is used as storage backend.

Example use case: email verification:
1) Call `RegisterAction()` to initiate email verification flow
2) Send email to user with e.g. a link `https://example.com/email-verify?token=<your-token-base64>`.
3) Handle requests on `/email-verify` path and call `ConfirmAction()` to gain access to data associated with this token and finish flow.

Things to consider:
1) Generic action type MUST be supported by [go-redis](https://github.com/redis/go-redis) `HSet()` function.
2) Although token is returned as `string` value, it is not url-safe and should be treated as immutable byte slice.