-- Sliding window rate limiter lua script

-- KEYS[1]: rate limit key
-- ARGV[1]: current timestamp in ms
-- ARGV[2]: window size in ms
-- ARGV[3]: max allowed requests

local key = KEYS[1]
local now = tonumber(ARGV[1])
local window = tonumber(ARGV[2])
local limit = tonumber(ARGV[3])
local clearBefore = now - window

-- Remove timestamps older than the current window
redis.call('ZREMRANGEBYSCORE', key, 0, clearBefore)

-- count requests in current window
local currentRequests = redis.call('ZCARD', key)

if currentRequests < limit then
    -- add current timestamp to the sorted set
    redis.call('ZADD', key, now, now)
	-- set TTL for this key
	redis.call('PEXPIRE', key, window)
    return { 1, limit - currentRequests - 1 } -- {allowed, remaining}
else
    return {0, 0}
end
