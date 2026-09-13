# 0015: Monotonic cache expiration

TTL and TTI use `time.Duration` deadlines derived from monotonic clock values;
absolute expiration is a separate wall-clock deadline.
