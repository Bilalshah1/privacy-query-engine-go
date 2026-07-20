-- Atomic check-and-deduct for a user's daily privacy budget.
--
-- KEYS[1] = budget key, e.g. "budget:alice_1:2026-07-19"
-- ARGV[1] = epsilon requested by this query
-- ARGV[2] = max daily epsilon budget (e.g. 10.0)
-- ARGV[3] = seconds until the key should expire

local current = tonumber(redis.call("GET", KEYS[1]) or "0")
local requested = tonumber(ARGV[1])
local max_budget = tonumber(ARGV[2])
local ttl_seconds = tonumber(ARGV[3])

if requested <= 0 then
  return {0, tostring(current), tostring(max_budget - current)}
end

if current + requested > max_budget then
  return {0, tostring(current), tostring(max_budget - current)}
end

local new_total = current + requested
redis.call("SET", KEYS[1], tostring(new_total), "EX", ttl_seconds)
return {1, tostring(new_total), tostring(max_budget - new_total)}