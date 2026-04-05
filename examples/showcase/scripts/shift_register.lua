local bits = tonumber(config["bits"]) or 8
local speed = tonumber(config["speed"]) or 500
local clock = inputs["clk"]
local active = clock == "1" or clock == "on" or clock == "true"

local result = {}
if active then
    local shift = math.floor(time.now() / speed) % bits
    local display = {}
    for i = 0, bits - 1 do
        local val = (i == shift) and "1" or "0"
        result["b" .. i] = val
        display[#display + 1] = val
    end
    result._display = table.concat(display)
else
    for i = 0, bits - 1 do
        result["b" .. i] = "0"
    end
    result._display = string.rep("-", bits)
end
return result
