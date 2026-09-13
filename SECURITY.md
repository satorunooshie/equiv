# Security

Hash seeds are generated per hash domain and are never exported or serialized.
Do not use `equiv` hashes as stable identifiers or cryptographic digests.
Resident key identity must not be changed by the caller while stored.
