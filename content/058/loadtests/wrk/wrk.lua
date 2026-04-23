-- wrk multi-endpoint round-robin script
-- Run: wrk -t4 -c50 -d15s -s wrk.lua http://127.0.0.1:8765
local paths = { "/simple", "/medium?n=2000", "/heavy?n=25" }
local i = 0

request = function()
  i = i + 1
  local p = paths[(i % #paths) + 1]
  return wrk.format("GET", p)
end
