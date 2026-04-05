local bits = tonumber(config["bits"]) or 8
local speed = tonumber(config["speed"]) or 500
local clock = inputs["clk"]
local active = clock == "1" or clock == "on" or clock == "true"

-- Compute shift position from time.
-- When clock is high, position advances. When low, it holds at the
-- current period boundary (quantized to speed intervals).
local now = time.now()
local pos
if active then
    pos = math.floor(now / speed) % bits
else
    -- Hold: snap to the nearest period boundary so the position is stable
    -- across ticks while clock is low. Uses a slower quantization.
    pos = math.floor(now / (speed * bits)) * 1 % bits
end

local result = {}
local display = {}
for i = 0, bits - 1 do
    local val = (i == pos) and "1" or "0"
    result["b" .. i] = val
    display[#display + 1] = val
end
result._display = table.concat(display)
return result
